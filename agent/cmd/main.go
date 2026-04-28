package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/poouo/NebulaPanel/agent/internal/client"
	"github.com/poouo/NebulaPanel/agent/internal/config"
	"github.com/poouo/NebulaPanel/agent/internal/core"
	"github.com/poouo/NebulaPanel/agent/internal/logger"
	"github.com/poouo/NebulaPanel/agent/internal/sysinfo"
)

const Version = "2.0.0"

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
	logger.Infof("Agent", "starting nebula-agent v%s panel=%s heartbeat=%ds",
		Version, cfg.PanelURL, cfg.HeartbeatInterval)

	// auto-locate xray if needed
	if _, err := os.Stat(cfg.XrayBin); err != nil {
		if alt, ok := lookupXray(); ok {
			cfg.XrayBin = alt
			logger.Infof("Agent", "located xray at %s", alt)
		} else {
			logger.Warnf("Agent", "xray binary not found; node generation disabled until installed at %s", cfg.XrayBin)
		}
	}

	cli := client.New(cfg.PanelURL, cfg.CommKey)
	mgr := core.NewManager(cfg.XrayBin, cfg.WorkDir)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	hbTicker := time.NewTicker(time.Duration(cfg.HeartbeatInterval) * time.Second)
	defer hbTicker.Stop()

	// initial heartbeat right away
	doHeartbeat(cli, mgr, cfg)

	for {
		select {
		case <-hbTicker.C:
			doHeartbeat(cli, mgr, cfg)
		case <-stop:
			logger.Infof("Agent", "shutdown signal received")
			mgr.Stop()
			return
		}
	}
}

func doHeartbeat(cli *client.Client, mgr *core.Manager, cfg *config.Config) {
	stats := sysinfo.Collect(Version)
	if cfg.NodeName != "" {
		stats.Hostname = cfg.NodeName
	}
	resp, err := cli.PostEncrypted("/api/agent/heartbeat", stats)
	if err != nil {
		logger.Errorf("Agent", "heartbeat failed: %v", err)
		return
	}
	logger.Infof("Agent", "heartbeat ok host=%s cpu=%.1f%% mem=%.1f%% net=%d/%d",
		stats.Host, stats.CPU, stats.Mem, stats.NetIn, stats.NetOut)

	// Optional config payload in heartbeat response
	if len(resp) == 0 {
		return
	}
	var spec core.Spec
	if err := json.Unmarshal(resp, &spec); err != nil {
		// Server may simply return {"status":"ok"} → ignore.
		return
	}
	if len(spec.Nodes) == 0 && len(spec.AuditRules) == 0 {
		return
	}
	if err := mgr.Apply(spec); err != nil {
		logger.Errorf("Agent", "apply config failed: %v", err)
	}
}

func lookupXray() (string, bool) {
	for _, p := range []string{
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
