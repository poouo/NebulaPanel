package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds agent runtime config loaded from KEY=VALUE file.
type Config struct {
	PanelURL          string
	CommKey           string
	AgentToken        string
	HeartbeatInterval int
	TrafficInterval   int
	LogDir            string
	WorkDir           string
	XrayBin           string
	NodeName          string
}

// LoadFile parses a simple KEY=VALUE file (shell-compatible) and applies defaults.
func LoadFile(path string) (*Config, error) {
	c := &Config{
		HeartbeatInterval: 30,
		TrafficInterval:   60,
		LogDir:            "/var/log/nebula-agent",
		WorkDir:           "/opt/nebula-agent",
		XrayBin:           "/opt/nebula-agent/xray",
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 0 {
			continue
		}
		k := strings.TrimSpace(line[:eq])
		v := strings.TrimSpace(line[eq+1:])
		v = strings.Trim(v, `"'`)
		switch k {
		case "PANEL_URL":
			c.PanelURL = v
		case "COMM_KEY":
			c.CommKey = v
		case "AGENT_TOKEN", "TOKEN":
			c.AgentToken = v
		case "HEARTBEAT_INTERVAL":
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				c.HeartbeatInterval = n
			}
		case "TRAFFIC_INTERVAL":
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				c.TrafficInterval = n
			}
		case "LOG_DIR":
			c.LogDir = v
		case "WORK_DIR":
			c.WorkDir = v
		case "XRAY_BIN":
			c.XrayBin = v
		case "NODE_NAME":
			c.NodeName = v
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if c.PanelURL == "" {
		return nil, fmt.Errorf("PANEL_URL is required in %s", path)
	}
	if c.CommKey == "" && c.AgentToken == "" {
		return nil, fmt.Errorf("either COMM_KEY or AGENT_TOKEN is required in %s", path)
	}
	return c, nil
}
