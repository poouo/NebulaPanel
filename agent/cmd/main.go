package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/poouo/NebulaPanel/agent/internal/client"
	"github.com/poouo/NebulaPanel/agent/internal/config"
	"github.com/poouo/NebulaPanel/agent/internal/core"
	"github.com/poouo/NebulaPanel/agent/internal/logger"
	"github.com/poouo/NebulaPanel/agent/internal/sysinfo"
)

const Version = "2.1.0"

// runtimeState tracks mutable values adjusted at runtime (fast/slow heartbeat).
type runtimeState struct {
	mu       sync.Mutex
	interval time.Duration
	restart  bool
}

func (s *runtimeState) get() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.interval
}

func (s *runtimeState) set(d time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d <= 0 || d == s.interval {
		return false
	}
	s.interval = d
	return true
}

func main() {
	confPath := flag.String("c", "/opt/nebula-agent/agent.conf", "agent config file (KEY=VALUE)")
	showVer := flag.Bool("v", false, "show version")
	flag.Parse()

	if *showVer {
		fmt.Printf("nebula-agent %s\n", Version)
		return
	}

	cfg, err := config.LoadFile(*confPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// If we cannot create / write to the configured WorkDir (typical for
	// non-root manual runs against the default /opt path), fall back to a
	// directory next to the config file. Production installs always run as
	// root via systemd and keep /opt/nebula-agent.
	if perr := os.MkdirAll(cfg.WorkDir, 0o755); perr != nil {
		base := filepath.Dir(*confPath)
		fallback := filepath.Join(base, "work")
		_ = os.MkdirAll(fallback, 0o755)
		cfg.WorkDir = fallback
		cfg.XrayBin = filepath.Join(fallback, "xray")
	}

	if err := logger.Init(cfg.LogDir); err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %v\n", err)
		os.Exit(1)
	}
	logger.Infof("Agent", "starting nebula-agent v%s panel=%s heartbeat=%ds token_mode=%v",
		Version, cfg.PanelURL, cfg.HeartbeatInterval, cfg.AgentToken != "")

	// auto-locate xray if needed
	if _, err := os.Stat(cfg.XrayBin); err != nil {
		if alt, ok := lookupXray(); ok {
			cfg.XrayBin = alt
			logger.Infof("Agent", "located xray at %s", alt)
		} else {
			logger.Warnf("Agent", "xray binary not found; node generation disabled until installed at %s", cfg.XrayBin)
		}
	}

	// The agent always uses the shared comm_key for AES envelope. When the
	// admin only issued a per-agent token, we still need a comm_key for the
	// envelope. In that case the installer pins comm_key to the same value
	// the panel uses. Token is carried inside the encrypted heartbeat
	// payload for identity matching.
	cli := client.New(cfg.PanelURL, cfg.CommKey)
	if cli.CommKey == "" && cfg.AgentToken != "" {
		// Exchange the registration token for the panel's shared comm key.
		// Retry with a gentle back-off because the panel may still be
		// restarting right after install.
		backoff := 2 * time.Second
		for i := 0; i < 10; i++ {
			if _, _, berr := cli.Bootstrap(cfg.AgentToken); berr == nil {
				logger.Infof("Agent", "bootstrap ok, comm key acquired")
				break
			} else {
				logger.Warnf("Agent", "bootstrap failed (try %d): %v", i+1, berr)
				time.Sleep(backoff)
				if backoff < 30*time.Second {
					backoff *= 2
				}
			}
		}
		if cli.CommKey == "" {
			logger.Errorf("Agent", "bootstrap never succeeded, exiting")
			os.Exit(1)
		}
	}
	mgr := core.NewManager(cfg.XrayBin, cfg.WorkDir)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	state := &runtimeState{interval: time.Duration(cfg.HeartbeatInterval) * time.Second}
	if state.get() <= 0 {
		state.set(15 * time.Second)
	}

	hbTimer := time.NewTimer(0)
	defer hbTimer.Stop()

	for {
		select {
		case <-hbTimer.C:
			next := doHeartbeat(cli, mgr, cfg, state)
			hbTimer.Reset(next)
		case <-stop:
			logger.Infof("Agent", "shutdown signal received")
			mgr.Stop()
			return
		}
	}
}

func doHeartbeat(cli *client.Client, mgr *core.Manager, cfg *config.Config, state *runtimeState) time.Duration {
	stats := sysinfo.Collect(Version)
	if cfg.NodeName != "" {
		stats.Hostname = cfg.NodeName
	}
	if cfg.AgentToken != "" {
		stats.Token = cfg.AgentToken
	}
	stats.XrayVersion = mgr.XrayVersion()

	resp, err := cli.PostEncrypted("/api/agent/heartbeat", stats)
	if err != nil {
		logger.Errorf("Agent", "heartbeat failed: %v", err)
		// Back off on error but don't exceed 60s.
		d := state.get()
		if d < 30*time.Second {
			return 15 * time.Second
		}
		return d
	}
	logger.Infof("Agent", "heartbeat ok host=%s cpu=%.1f%% mem=%.1f%% net=%d/%d xray=%s",
		stats.Host, stats.CPU, stats.Mem, stats.NetIn, stats.NetOut, stats.XrayVersion)

	if len(resp) == 0 {
		return state.get()
	}
	var spec core.Spec
	if err := json.Unmarshal(resp, &spec); err != nil {
		logger.Warnf("Agent", "spec parse err=%v", err)
		return state.get()
	}

	// Honor restart flag from panel.
	if spec.Restart {
		logger.Infof("Agent", "restart requested by panel, exiting now")
		mgr.Stop()
		// systemd restart=always will bring us back up.
		go func() {
			time.Sleep(500 * time.Millisecond)
			os.Exit(0)
		}()
	}

	// Apply heartbeat cadence hint from panel.
	if spec.NextHeartbeat > 0 {
		newD := time.Duration(spec.NextHeartbeat) * time.Second
		if state.set(newD) {
			logger.Infof("Agent", "heartbeat interval updated to %s", newD)
		}
	}

	if len(spec.Nodes) > 0 || len(spec.AuditRules) > 0 {
		if err := mgr.Apply(spec); err != nil {
			logger.Errorf("Agent", "apply config failed: %v", err)
		}
	}
	return state.get()
}

func lookupXray() (string, bool) {
	for _, p := range []string{
		"/opt/nebula-agent/xray/xray",
		"/usr/local/bin/xray",
		"/usr/bin/xray",
		filepath.Join(os.Getenv("HOME"), ".local/bin/xray"),
	} {
		if _, err := os.Stat(p); err == nil {
			return p, true
		}
	}
	return "", false
}
