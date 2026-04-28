package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/poouo/NebulaPanel/agent/internal/logger"
)

// NodeSpec is the panel-side node descriptor sent to the agent.
type NodeSpec struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	Transport   string `json:"transport"`
	TLS         int    `json:"tls"`
	TLSSNI      string `json:"tls_sni"`
	UUID        string `json:"uuid"`
	AlterID     int    `json:"alter_id"`
	ExtraConfig string `json:"extra_config"`
}

// AuditRule is a domain block rule.
type AuditRule struct {
	Domain string `json:"domain"`
}

// Spec is the full configuration the panel sends back via the heartbeat
// response. It contains both nodes and audit rules.
type Spec struct {
	Nodes       []NodeSpec  `json:"nodes"`
	Audit       bool        `json:"audit_enabled"`
	AuditRules  []AuditRule `json:"audit_rules"`
	XrayVersion string      `json:"xray_version"`
}

// Manager owns the running xray child process and reconciles its config.
type Manager struct {
	mu       sync.Mutex
	bin      string
	workDir  string
	cfgPath  string
	curHash  string
	cmd      *exec.Cmd
	cancelFn context.CancelFunc
}

// NewManager creates a new Manager. Caller may call Apply(spec) to (re)load.
func NewManager(bin, workDir string) *Manager {
	return &Manager{
		bin:     bin,
		workDir: workDir,
		cfgPath: filepath.Join(workDir, "xray.json"),
	}
}

// Stop terminates a running child process if any.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stopLocked()
}

func (m *Manager) stopLocked() {
	if m.cancelFn != nil {
		m.cancelFn()
		m.cancelFn = nil
	}
	if m.cmd != nil && m.cmd.Process != nil {
		_ = m.cmd.Process.Signal(syscall.SIGTERM)
		done := make(chan struct{})
		go func() { _ = m.cmd.Wait(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = m.cmd.Process.Kill()
		}
		m.cmd = nil
	}
}

// Apply renders config from spec and (re)starts xray when content changes.
func (m *Manager) Apply(spec Spec) error {
	cfg := buildXrayConfig(spec)
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	sum := sha256.Sum256(raw)
	hash := hex.EncodeToString(sum[:])

	m.mu.Lock()
	defer m.mu.Unlock()

	if hash == m.curHash && m.cmd != nil {
		// nothing to do
		return nil
	}

	if err := os.MkdirAll(m.workDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(m.cfgPath, raw, 0o600); err != nil {
		return err
	}

	m.stopLocked()

	if _, err := os.Stat(m.bin); err != nil {
		logger.Warnf("Core", "xray binary not found at %s, will not start kernel; config saved to %s", m.bin, m.cfgPath)
		m.curHash = hash
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFn = cancel
	m.cmd = exec.CommandContext(ctx, m.bin, "run", "-c", m.cfgPath)
	m.cmd.Stdout = newPrefixWriter("xray")
	m.cmd.Stderr = newPrefixWriter("xray")
	if err := m.cmd.Start(); err != nil {
		m.cmd = nil
		return fmt.Errorf("start xray: %w", err)
	}
	m.curHash = hash
	logger.Infof("Core", "xray started pid=%d nodes=%d audit=%v rules=%d",
		m.cmd.Process.Pid, len(spec.Nodes), spec.Audit, len(spec.AuditRules))

	go func(c *exec.Cmd) {
		_ = c.Wait()
		logger.Warnf("Core", "xray process exited, will be re-applied on next sync")
		m.mu.Lock()
		if m.cmd == c {
			m.cmd = nil
			m.curHash = ""
		}
		m.mu.Unlock()
	}(m.cmd)
	return nil
}

// XrayVersion returns the version of the bundled xray (best-effort).
func (m *Manager) XrayVersion() string {
	if _, err := os.Stat(m.bin); err != nil {
		return ""
	}
	out, err := exec.Command(m.bin, "version").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Xray ") {
			return line
		}
	}
	return strings.TrimSpace(string(out))
}

// CurrentConfigPath returns the path of the rendered xray config.
func (m *Manager) CurrentConfigPath() string { return m.cfgPath }

// ───────────────────────── render xray config ─────────────────────────

type xrayInbound struct {
	Tag      string                 `json:"tag"`
	Listen   string                 `json:"listen,omitempty"`
	Port     int                    `json:"port"`
	Protocol string                 `json:"protocol"`
	Settings map[string]interface{} `json:"settings"`
	Stream   map[string]interface{} `json:"streamSettings,omitempty"`
}

type xrayOutbound struct {
	Tag      string                 `json:"tag"`
	Protocol string                 `json:"protocol"`
	Settings map[string]interface{} `json:"settings,omitempty"`
}

type xrayRoutingRule struct {
	Type        string   `json:"type"`
	Domain      []string `json:"domain,omitempty"`
	OutboundTag string   `json:"outboundTag"`
}

type xrayConfig struct {
	Log struct {
		Loglevel string `json:"loglevel"`
	} `json:"log"`
	Inbounds  []xrayInbound  `json:"inbounds"`
	Outbounds []xrayOutbound `json:"outbounds"`
	Routing   struct {
		DomainStrategy string            `json:"domainStrategy"`
		Rules          []xrayRoutingRule `json:"rules"`
	} `json:"routing"`
}

func buildXrayConfig(spec Spec) xrayConfig {
	cfg := xrayConfig{}
	cfg.Log.Loglevel = "warning"

	for _, n := range spec.Nodes {
		ib := renderInbound(n)
		if ib.Protocol != "" {
			cfg.Inbounds = append(cfg.Inbounds, ib)
		}
	}

	cfg.Outbounds = []xrayOutbound{
		{Tag: "direct", Protocol: "freedom"},
		{Tag: "block", Protocol: "blackhole"},
	}
	cfg.Routing.DomainStrategy = "AsIs"
	if spec.Audit && len(spec.AuditRules) > 0 {
		var doms []string
		for _, r := range spec.AuditRules {
			d := strings.TrimSpace(r.Domain)
			if d == "" {
				continue
			}
			doms = append(doms, d)
		}
		if len(doms) > 0 {
			cfg.Routing.Rules = append(cfg.Routing.Rules, xrayRoutingRule{
				Type: "field", Domain: doms, OutboundTag: "block",
			})
		}
	}
	return cfg
}

func renderInbound(n NodeSpec) xrayInbound {
	ib := xrayInbound{
		Tag:    fmt.Sprintf("node-%d", n.ID),
		Listen: "0.0.0.0",
		Port:   n.Port,
	}
	switch strings.ToLower(n.Protocol) {
	case "vmess":
		ib.Protocol = "vmess"
		ib.Settings = map[string]interface{}{
			"clients": []map[string]interface{}{
				{"id": n.UUID, "alterId": n.AlterID},
			},
		}
	case "vless":
		ib.Protocol = "vless"
		ib.Settings = map[string]interface{}{
			"clients":    []map[string]interface{}{{"id": n.UUID}},
			"decryption": "none",
		}
	case "trojan":
		ib.Protocol = "trojan"
		ib.Settings = map[string]interface{}{
			"clients": []map[string]interface{}{{"password": n.UUID}},
		}
	case "ss", "shadowsocks":
		method := "aes-256-gcm"
		password := n.UUID
		if n.ExtraConfig != "" {
			var extra map[string]string
			if json.Unmarshal([]byte(n.ExtraConfig), &extra) == nil {
				if v := extra["method"]; v != "" {
					method = v
				}
				if v := extra["password"]; v != "" {
					password = v
				}
			}
		}
		ib.Protocol = "shadowsocks"
		ib.Settings = map[string]interface{}{
			"method":   method,
			"password": password,
			"network":  "tcp,udp",
		}
	default:
		// unsupported by xray inbound here (e.g. hysteria2 needs hysteria core)
		return xrayInbound{}
	}

	stream := map[string]interface{}{
		"network": strings.ToLower(strings.TrimSpace(orDefault(n.Transport, "tcp"))),
	}
	if n.TLS == 1 {
		stream["security"] = "tls"
		tls := map[string]interface{}{}
		if n.TLSSNI != "" {
			tls["serverName"] = n.TLSSNI
		}
		stream["tlsSettings"] = tls
	}
	ib.Stream = stream
	return ib
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// prefixWriter implements io.Writer to forward kernel stdout/stderr to logger.
type prefixWriter struct{ tag string }

func newPrefixWriter(tag string) *prefixWriter { return &prefixWriter{tag: tag} }

func (p *prefixWriter) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	for _, line := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		if line == "" {
			continue
		}
		logger.Infof(p.tag, "%s", line)
	}
	return len(b), nil
}

// ErrNotConfigured is returned when no nodes are configured.
var ErrNotConfigured = errors.New("no nodes configured")
