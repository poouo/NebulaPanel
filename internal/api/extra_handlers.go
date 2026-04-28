package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/poouo/NebulaPanel/internal/auth"
	"github.com/poouo/NebulaPanel/internal/db"
	"github.com/poouo/NebulaPanel/internal/logger"
)

// handleLoginChallenge issues a one-time challenge for the new password
// challenge/response flow. The client computes:
//
//	clientHash = SHA-256(plaintext_password)         // never sent in cleartext
//	response   = HMAC-SHA256(clientHash, challenge)  // sent to /api/login
//
// The plaintext password never leaves the browser and the database keeps
// only bcrypt(clientHash) plus an opaque verifier reference for the HMAC.
func handleLoginChallenge(w http.ResponseWriter, r *http.Request) {
	c := auth.IssueChallenge()
	jsonOK(w, map[string]string{"challenge": c})
}

// handleUpdateAgentMeta updates remark / entry_ip / name fields for an Agent.
// These fields are user-editable in the panel; entry_ip overrides the
// auto-detected outbound IP reported by the agent and is what end-users connect
// to.
func handleUpdateAgentMeta(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name    *string `json:"name"`
		Remark  *string `json:"remark"`
		EntryIP *string `json:"entry_ip"`
	}
	if err := parseJSON(r, &req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	sets := []string{}
	args := []interface{}{}
	if req.Name != nil {
		sets = append(sets, "name=?")
		args = append(args, *req.Name)
	}
	if req.Remark != nil {
		sets = append(sets, "remark=?")
		args = append(args, *req.Remark)
	}
	if req.EntryIP != nil {
		sets = append(sets, "entry_ip=?")
		args = append(args, *req.EntryIP)
	}
	if len(sets) == 0 {
		jsonError(w, "nothing to update", http.StatusBadRequest)
		return
	}
	sets = append(sets, "updated_at=CURRENT_TIMESTAMP")
	args = append(args, id)
	query := "UPDATE agents SET " + joinComma(sets) + " WHERE id=?"
	if _, err := db.DB.Exec(query, args...); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Infof("Admin", "Updated agent meta id=%s", id)
	jsonOK(w, "updated")
}

// handleAgentBootstrap is called once by the installer: it exchanges the
// agent token for the shared comm_key needed to encrypt further heartbeats.
// This avoids putting the comm_key into the public settings endpoint.
func handleAgentBootstrap(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := parseJSON(r, &req); err != nil || strings.TrimSpace(req.Token) == "" {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	var id int
	var name string
	err := db.DB.QueryRow("SELECT id, name FROM agents WHERE token=?", req.Token).Scan(&id, &name)
	if err != nil {
		logger.Warnf("Agent", "Bootstrap with unknown token from %s", r.RemoteAddr)
		jsonError(w, "invalid token", http.StatusForbidden)
		return
	}
	commKey := getSetting("comm_key", "")
	if commKey == "" {
		jsonError(w, "panel comm_key not configured", http.StatusInternalServerError)
		return
	}
	logger.Infof("Agent", "Bootstrap ok for agent id=%d name=%s", id, name)
	jsonOK(w, map[string]interface{}{
		"id":       id,
		"name":     name,
		"comm_key": commKey,
	})
}

// handleRestartAgent marks the agent to restart on its next heartbeat.
// The agent consumes this flag and exits with status 0 so systemd restarts it.
func handleRestartAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, err := db.DB.Exec("UPDATE agents SET restart_pending=1, report_fast_until=? WHERE id=?",
		time.Now().UTC().Add(30*time.Second).Format("2006-01-02 15:04:05"), id); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Infof("Admin", "Queued restart for agent id=%s", id)
	jsonOK(w, "restart queued")
}

// handleAgentFastMode enables the "fast heartbeat" window so that the
// selected agent(s) report every few seconds while the admin is actively
// viewing the Agents page. When `id` is omitted, fast mode is applied to
// all agents.
func handleAgentFastMode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID       int `json:"id"`
		Duration int `json:"duration"`
		Seconds  int `json:"seconds"`
	}
	_ = parseJSON(r, &req)
	if req.Duration <= 0 {
		req.Duration = req.Seconds
	}
	if req.Duration <= 0 || req.Duration > 300 {
		req.Duration = 90
	}
	until := time.Now().UTC().Add(time.Duration(req.Duration) * time.Second).Format("2006-01-02 15:04:05")
	if req.ID > 0 {
		db.DB.Exec("UPDATE agents SET report_fast_until=? WHERE id=?", until, req.ID)
	} else {
		db.DB.Exec("UPDATE agents SET report_fast_until=?", until)
	}
	jsonOK(w, map[string]interface{}{"until": until, "duration": req.Duration})
}

// handleRotateAgentToken regenerates the Agent registration token.
// Useful when the old token leaks or when you want to reset install scripts.
func handleRotateAgentToken(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	token := generateAgentToken()
	if _, err := db.DB.Exec("UPDATE agents SET token=? WHERE id=?", token, id); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Infof("Admin", "Rotated token for agent id=%s", id)
	jsonOK(w, map[string]string{"token": token})
}

// handleGetAgentInstallScript returns the token-based one-click installer
// for a specific agent. The token is embedded so the installer can register
// directly with the panel without needing the shared COMM_KEY in the URL.
func handleGetAgentInstallScript(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		jsonError(w, "invalid id", http.StatusBadRequest)
		return
	}
	var name, token string
	if err := db.DB.QueryRow("SELECT name, COALESCE(token,'') FROM agents WHERE id=?", id).Scan(&name, &token); err != nil {
		jsonError(w, "agent not found", http.StatusNotFound)
		return
	}
	if token == "" {
		token = generateAgentToken()
		db.DB.Exec("UPDATE agents SET token=? WHERE id=?", token, id)
	}
	panelURL := strings.TrimRight(getPanelURL(r), "/")
	oneLiner := fmt.Sprintf("bash <(curl -fsSL %s/static/agent/install.sh) install %s %s", panelURL, panelURL, token)
	uninstall := fmt.Sprintf("bash <(curl -fsSL %s/static/agent/install.sh) uninstall", panelURL)
	jsonOK(w, map[string]interface{}{
		"id":            id,
		"name":          name,
		"token":         token,
		"panel_url":     panelURL,
		"install_cmd":   oneLiner,
		"uninstall_cmd": uninstall,
	})
}

// generateAgentToken creates a 32-char random hex token used for agent
// registration.
func generateAgentToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}

// getPanelURL resolves the panel's public base URL. Prefers the admin-
// configured `panel_host` setting; otherwise derives it from the request.
func getPanelURL(r *http.Request) string {
	host := getSetting("panel_host", "")
	if host == "" {
		host = r.Host
	}
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	if strings.Contains(host, "://") {
		return host
	}
	return scheme + "://" + host
}

// ── 审计规则 ──

func handleListAuditRules(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query("SELECT id, domain, COALESCE(remark,''), enabled, created_at FROM audit_rules ORDER BY id ASC")
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var list []map[string]interface{}
	for rows.Next() {
		var id, enabled int
		var domain, remark, createdAt string
		rows.Scan(&id, &domain, &remark, &enabled, &createdAt)
		list = append(list, map[string]interface{}{
			"id": id, "domain": domain, "remark": remark,
			"enabled": enabled, "created_at": createdAt,
		})
	}
	if list == nil {
		list = []map[string]interface{}{}
	}
	jsonOK(w, list)
}

func handleCreateAuditRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Domain  string `json:"domain"`
		Remark  string `json:"remark"`
		Enabled int    `json:"enabled"`
	}
	if err := parseJSON(r, &req); err != nil || req.Domain == "" {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Enabled == 0 {
		req.Enabled = 1
	}
	res, err := db.DB.Exec("INSERT INTO audit_rules (domain, remark, enabled) VALUES (?, ?, ?)",
		req.Domain, req.Remark, req.Enabled)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()
	logger.Infof("Admin", "Created audit rule: %s (id=%d)", req.Domain, id)
	jsonOK(w, map[string]interface{}{"id": id})
}

func handleUpdateAuditRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req map[string]interface{}
	if err := parseJSON(r, &req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	allow := map[string]bool{"domain": true, "remark": true, "enabled": true}
	sets := []string{}
	args := []interface{}{}
	for k, v := range req {
		if allow[k] {
			sets = append(sets, k+"=?")
			args = append(args, v)
		}
	}
	if len(sets) == 0 {
		jsonError(w, "nothing to update", http.StatusBadRequest)
		return
	}
	args = append(args, id)
	query := "UPDATE audit_rules SET " + joinComma(sets) + " WHERE id=?"
	db.DB.Exec(query, args...)
	jsonOK(w, "updated")
}

func handleDeleteAuditRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	db.DB.Exec("DELETE FROM audit_rules WHERE id=?", id)
	logger.Infof("Admin", "Deleted audit rule id=%s", id)
	jsonOK(w, "deleted")
}

// ── helpers ──

func joinComma(parts []string) string {
	return strings.Join(parts, ",")
}

// buildAgentSpec returns the JSON the agent expects in heartbeat reply.
// Includes the list of nodes assigned to this agent, the active audit
// rules, the requested next heartbeat interval (fast vs. slow mode) and
// an optional restart flag. The caller encrypts this body.
func buildAgentSpec(agentID int) []byte {
	type nodeOut struct {
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
	type ruleOut struct {
		Domain string `json:"domain"`
	}
	type spec struct {
		Status        string    `json:"status"`
		Nodes         []nodeOut `json:"nodes"`
		Audit         bool      `json:"audit_enabled"`
		AuditRules    []ruleOut `json:"audit_rules"`
		NextHeartbeat int       `json:"next_heartbeat"`
		Restart       bool      `json:"restart"`
	}

	out := spec{Status: "ok", Nodes: []nodeOut{}, AuditRules: []ruleOut{}, NextHeartbeat: 15}

	// Determine heartbeat cadence: fast if report_fast_until > now (UTC).
	var fastUntil *string
	_ = db.DB.QueryRow("SELECT report_fast_until FROM agents WHERE id=?", agentID).Scan(&fastUntil)
	if fastUntil != nil && *fastUntil != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", *fastUntil); err == nil && t.After(time.Now().UTC()) {
			out.NextHeartbeat = 3
		}
	}

	// Consume pending restart flag (one-shot).
	var pending int
	_ = db.DB.QueryRow("SELECT restart_pending FROM agents WHERE id=?", agentID).Scan(&pending)
	if pending == 1 {
		out.Restart = true
		db.DB.Exec("UPDATE agents SET restart_pending=0 WHERE id=?", agentID)
	}

	rows, err := db.DB.Query(`SELECT id, name, address, port, protocol, transport, tls,
				COALESCE(tls_sni,''), COALESCE(uuid,''), alter_id, COALESCE(extra_config,'')
			 FROM nodes WHERE enabled=1 AND agent_id=?
			 ORDER BY sort_order ASC, id ASC`, agentID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var n nodeOut
			rows.Scan(&n.ID, &n.Name, &n.Address, &n.Port, &n.Protocol, &n.Transport,
				&n.TLS, &n.TLSSNI, &n.UUID, &n.AlterID, &n.ExtraConfig)
			out.Nodes = append(out.Nodes, n)
		}
	}

	out.Audit = getSetting("audit_enabled", "false") == "true"
	if out.Audit {
		ar, err := db.DB.Query("SELECT domain FROM audit_rules WHERE enabled=1")
		if err == nil {
			defer ar.Close()
			for ar.Next() {
				var d string
				ar.Scan(&d)
				out.AuditRules = append(out.AuditRules, ruleOut{Domain: d})
			}
		}
	}
	b, _ := json.Marshal(out)
	return b
}
