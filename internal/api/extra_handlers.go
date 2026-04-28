package api

import (
	"encoding/json"
	"net/http"

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
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ","
		}
		out += p
	}
	return out
}

// agentSpec returns the encrypted JSON the agent expects in heartbeat reply.
// Includes the list of nodes assigned to this agent and the active audit
// rules. Unencrypted form is returned; the caller wraps it with crypto.Encrypt.
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
		Status      string    `json:"status"`
		Nodes       []nodeOut `json:"nodes"`
		Audit       bool      `json:"audit_enabled"`
		AuditRules  []ruleOut `json:"audit_rules"`
	}

	out := spec{Status: "ok", Nodes: []nodeOut{}, AuditRules: []ruleOut{}}

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
