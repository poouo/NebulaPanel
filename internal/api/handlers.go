package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/poouo/NebulaPanel/internal/auth"
	"github.com/poouo/NebulaPanel/internal/captcha"
	"github.com/poouo/NebulaPanel/internal/crypto"
	"github.com/poouo/NebulaPanel/internal/db"
	"github.com/poouo/NebulaPanel/internal/logger"
	"github.com/poouo/NebulaPanel/internal/subscription"
)

// ── 通用工具 ──

func jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"code": 0, "data": data})
}

func jsonError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{"code": -1, "message": msg})
}

func parseJSON(r *http.Request, v interface{}) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

func getCommKey() string {
	var key string
	db.DB.QueryRow("SELECT value FROM settings WHERE key='comm_key'").Scan(&key)
	return key
}

func getSetting(key, fallback string) string {
	var val string
	err := db.DB.QueryRow("SELECT value FROM settings WHERE key=?", key).Scan(&val)
	if err != nil || val == "" {
		return fallback
	}
	return val
}

// ── 静态页面 ──

func handleIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/static/index.html")
}

// ── 验证码 ──

func handleCaptcha(w http.ResponseWriter, r *http.Request) {
	id, svg := captcha.Generate()
	jsonOK(w, map[string]string{"captcha_id": id, "captcha_svg": svg})
}

func handleNeedCaptcha(w http.ResponseWriter, r *http.Request) {
	ip := getClientIP(r)
	jsonOK(w, map[string]bool{"need": captcha.NeedCaptcha(ip)})
}

// ── 注册 ──

func handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username   string `json:"username"`
		Password   string `json:"password"`    // legacy plaintext
		ClientHash string `json:"client_hash"` // sha256(password) hex (preferred)
		CaptchaID  string `json:"captcha_id"`
		Captcha    string `json:"captcha"`
	}
	if err := parseJSON(r, &req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	// 检查是否开放注册
	if getSetting("allow_register", "true") != "true" {
		jsonError(w, "registration is disabled", http.StatusForbidden)
		return
	}

	// 验证码必填
	if !captcha.Verify(req.CaptchaID, req.Captcha) {
		logger.Warnf("Auth", "Register captcha failed: %s from %s", req.Username, getClientIP(r))
		jsonError(w, "captcha verification failed", http.StatusBadRequest)
		return
	}

	if len(req.Username) < 3 {
		jsonError(w, "username min 3 chars", http.StatusBadRequest)
		return
	}

	// Pick the credential to store. Prefer client-side SHA-256 hash so we never
	// touch the plaintext password on the server side.
	ch := strings.TrimSpace(req.ClientHash)
	if ch == "" {
		if len(req.Password) < 6 {
			jsonError(w, "password min 6 chars", http.StatusBadRequest)
			return
		}
		ch = auth.SHA256Hex(req.Password)
	}
	hash, _ := auth.HashClientHash(ch)
	subToken := auth.GenerateSubToken()

	res, err := db.DB.Exec(
		`INSERT INTO users (username, password, role, sub_token) VALUES (?, ?, 'user', ?)`,
		req.Username, hash, subToken)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			jsonError(w, "username already exists", http.StatusConflict)
			return
		}
		jsonError(w, "create user failed", http.StatusInternalServerError)
		return
	}

	newID, _ := res.LastInsertId()
	_ = auth.SaveVerifier(int(newID), ch)

	logger.Infof("Auth", "User registered: %s from %s", req.Username, getClientIP(r))
	jsonOK(w, map[string]string{"message": "registered successfully"})
}

// ── 登录 ──

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username   string `json:"username"`
		Password   string `json:"password"`     // legacy plaintext (fallback only)
		ClientHash string `json:"client_hash"`  // sha256(password) hex (preferred)
		Challenge  string `json:"challenge"`    // server-issued nonce (for HMAC mode)
		Response   string `json:"response"`     // hmac-sha256(client_hash, challenge)
		CaptchaID  string `json:"captcha_id"`
		Captcha    string `json:"captcha"`
	}
	if err := parseJSON(r, &req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	ip := getClientIP(r)

	// 如果该 IP 需要验证码，则校验
	if captcha.NeedCaptcha(ip) {
		if !captcha.Verify(req.CaptchaID, req.Captcha) {
			logger.Warnf("Auth", "Login captcha failed: %s from %s", req.Username, ip)
			jsonError(w, "captcha verification failed", http.StatusBadRequest)
			return
		}
	}

	var id int
	var hash, role string
	var enabled int
	err := db.DB.QueryRow(
		"SELECT id, password, role, enabled FROM users WHERE username = ?",
		req.Username).Scan(&id, &hash, &role, &enabled)
	if err != nil {
		captcha.RecordFail(ip)
		logger.Warnf("Auth", "Login failed (user not found): %s from %s", req.Username, ip)
		jsonError(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	ok := false
	// Mode A: HMAC challenge/response over sha256(password)
	if req.Challenge != "" && req.Response != "" {
		if auth.ConsumeChallenge(req.Challenge) {
			stored := auth.LoadVerifier(id)
			if stored != "" {
				if auth.VerifyHMAC(stored, req.Challenge, req.Response) {
					ok = true
				}
			}
		}
	}
	// Mode B: client sent sha256(password) hex; verify against bcrypt hash
	if !ok && req.ClientHash != "" {
		if auth.CheckClientHash(req.ClientHash, hash) {
			ok = true
			// upgrade verifier so future logins can use Mode A
			_ = auth.SaveVerifier(id, req.ClientHash)
		}
	}
	// Mode C (legacy): plaintext password directly bcrypt-checked.
	if !ok && req.Password != "" {
		if auth.CheckPassword(req.Password, hash) {
			ok = true
			// migrate to client-hash chain transparently
			ch := auth.SHA256Hex(req.Password)
			if newHash, herr := auth.HashClientHash(ch); herr == nil {
				db.DB.Exec("UPDATE users SET password=?, updated_at=CURRENT_TIMESTAMP WHERE id=?", newHash, id)
				_ = auth.SaveVerifier(id, ch)
			}
		}
	}

	if !ok {
		captcha.RecordFail(ip)
		logger.Warnf("Auth", "Login failed (wrong password): %s from %s", req.Username, ip)
		need := captcha.NeedCaptcha(ip)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"code": -1, "message": "invalid credentials", "need_captcha": need,
		})
		return
	}

	if enabled != 1 {
		jsonError(w, "account disabled", http.StatusForbidden)
		return
	}

	captcha.ClearFail(ip)
	token, _ := auth.GenerateToken(id, req.Username, role)
	logger.Infof("Auth", "User logged in: %s from %s", req.Username, ip)
	jsonOK(w, map[string]interface{}{"token": token, "role": role, "username": req.Username})
}

// ── 当前用户 ──

func handleMe(w http.ResponseWriter, r *http.Request) {
	uid, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	var username, role, subToken string
	var trafficUp, trafficDown, trafficLimit int64
	var speedLimit int
	var enabled int
	var expireAt, createdAt *string

	err := db.DB.QueryRow(
		`SELECT username, role, traffic_up, traffic_down, traffic_limit,
				speed_limit, expire_at, sub_token, enabled, created_at
		 FROM users WHERE id = ?`, uid).Scan(
		&username, &role, &trafficUp, &trafficDown, &trafficLimit,
		&speedLimit, &expireAt, &subToken, &enabled, &createdAt)
	if err != nil {
		jsonError(w, "user not found", http.StatusNotFound)
		return
	}

	jsonOK(w, map[string]interface{}{
		"id":            uid,
		"username":      username,
		"role":          role,
		"traffic_up":    trafficUp,
		"traffic_down":  trafficDown,
		"traffic_limit": trafficLimit,
		"traffic_used":  trafficUp + trafficDown,
		"speed_limit":   speedLimit,
		"expire_at":     expireAt,
		"sub_token":     subToken,
		"enabled":       enabled,
		"created_at":    createdAt,
	})
}

func handleChangePassword(w http.ResponseWriter, r *http.Request) {
	uid, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	var req struct {
		OldPassword   string `json:"old_password"`
		NewPassword   string `json:"new_password"`
		OldClientHash string `json:"old_client_hash"`
		NewClientHash string `json:"new_client_hash"`
	}
	if err := parseJSON(r, &req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	oldCH := strings.TrimSpace(req.OldClientHash)
	if oldCH == "" && req.OldPassword != "" {
		oldCH = auth.SHA256Hex(req.OldPassword)
	}
	newCH := strings.TrimSpace(req.NewClientHash)
	if newCH == "" {
		if len(req.NewPassword) < 6 {
			jsonError(w, "password min 6 chars", http.StatusBadRequest)
			return
		}
		newCH = auth.SHA256Hex(req.NewPassword)
	}

	var hash string
	db.DB.QueryRow("SELECT password FROM users WHERE id = ?", uid).Scan(&hash)
	if !auth.CheckClientHash(oldCH, hash) && !auth.CheckPassword(req.OldPassword, hash) {
		jsonError(w, "old password incorrect", http.StatusBadRequest)
		return
	}

	newHash, _ := auth.HashClientHash(newCH)
	db.DB.Exec("UPDATE users SET password = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", newHash, uid)
	_ = auth.SaveVerifier(uid, newCH)
	logger.Infof("Auth", "User %d changed password", uid)
	jsonOK(w, "password changed")
}

// ── 用户管理 (admin) ──

func handleListUsers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query(
		`SELECT id, username, role, traffic_up, traffic_down, traffic_limit,
				speed_limit, expire_at, sub_token, enabled, reset_day, created_at, updated_at
		 FROM users ORDER BY id ASC`)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []map[string]interface{}
	for rows.Next() {
		var id, speedLimit, enabled, resetDay int
		var trafficUp, trafficDown, trafficLimit int64
		var username, role string
		var subToken, expireAt, createdAt, updatedAt *string
		rows.Scan(&id, &username, &role, &trafficUp, &trafficDown, &trafficLimit,
			&speedLimit, &expireAt, &subToken, &enabled, &resetDay, &createdAt, &updatedAt)
		users = append(users, map[string]interface{}{
			"id": id, "username": username, "role": role,
			"traffic_up": trafficUp, "traffic_down": trafficDown,
			"traffic_limit": trafficLimit, "traffic_used": trafficUp + trafficDown,
			"speed_limit": speedLimit, "expire_at": expireAt,
			"sub_token": subToken, "enabled": enabled,
			"reset_day": resetDay,
			"created_at": createdAt, "updated_at": updatedAt,
		})
	}
	if users == nil {
		users = []map[string]interface{}{}
	}
	jsonOK(w, users)
}

func handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username     string `json:"username"`
		Password     string `json:"password"`
		Role         string `json:"role"`
		TrafficLimit int64  `json:"traffic_limit"`
		SpeedLimit   int    `json:"speed_limit"`
		ExpireAt     string `json:"expire_at"`
		ResetDay     int    `json:"reset_day"`
	}
	if err := parseJSON(r, &req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Role == "" {
		req.Role = "user"
	}
	// Always store bcrypt(SHA-256(plain)) so panel never persists plaintext.
	ch := auth.SHA256Hex(req.Password)
	hash, _ := auth.HashClientHash(ch)
	subToken := auth.GenerateSubToken()

	var expireAt interface{} = nil
	if req.ExpireAt != "" {
		expireAt = req.ExpireAt
	}

	result, err := db.DB.Exec(
		`INSERT INTO users (username, password, role, traffic_limit, speed_limit, expire_at, reset_day, sub_token)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		req.Username, hash, req.Role, req.TrafficLimit, req.SpeedLimit, expireAt, req.ResetDay, subToken)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			jsonError(w, "username already exists", http.StatusConflict)
			return
		}
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, _ := result.LastInsertId()
	_ = auth.SaveVerifier(int(id), ch)
	logger.Infof("Admin", "Created user: %s (id=%d)", req.Username, id)
	jsonOK(w, map[string]interface{}{"id": id})
}

func handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Username     *string `json:"username"`
		Password     *string `json:"password"`
		Role         *string `json:"role"`
		TrafficLimit *int64  `json:"traffic_limit"`
		SpeedLimit   *int    `json:"speed_limit"`
		ExpireAt     *string `json:"expire_at"`
		ResetDay     *int    `json:"reset_day"`
		Enabled      *int    `json:"enabled"`
	}
	if err := parseJSON(r, &req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	sets := []string{}
	args := []interface{}{}
	if req.Username != nil {
		sets = append(sets, "username=?")
		args = append(args, *req.Username)
	}
	if req.Password != nil && *req.Password != "" {
		ch := auth.SHA256Hex(*req.Password)
		hash, _ := auth.HashClientHash(ch)
		sets = append(sets, "password=?")
		args = append(args, hash)
		if uid, _ := strconv.Atoi(id); uid > 0 {
			_ = auth.SaveVerifier(uid, ch)
		}
	}
	if req.Role != nil {
		sets = append(sets, "role=?")
		args = append(args, *req.Role)
	}
	if req.TrafficLimit != nil {
		sets = append(sets, "traffic_limit=?")
		args = append(args, *req.TrafficLimit)
	}
	if req.SpeedLimit != nil {
		sets = append(sets, "speed_limit=?")
		args = append(args, *req.SpeedLimit)
	}
	if req.ExpireAt != nil {
		sets = append(sets, "expire_at=?")
		args = append(args, *req.ExpireAt)
	}
	if req.ResetDay != nil {
		sets = append(sets, "reset_day=?")
		args = append(args, *req.ResetDay)
	}
	if req.Enabled != nil {
		sets = append(sets, "enabled=?")
		args = append(args, *req.Enabled)
	}

	if len(sets) == 0 {
		jsonError(w, "nothing to update", http.StatusBadRequest)
		return
	}
	sets = append(sets, "updated_at=CURRENT_TIMESTAMP")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE users SET %s WHERE id=?", strings.Join(sets, ","))
	db.DB.Exec(query, args...)
	logger.Infof("Admin", "Updated user id=%s", id)
	jsonOK(w, "updated")
}

func handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	db.DB.Exec("DELETE FROM user_nodes WHERE user_id=?", id)
	db.DB.Exec("DELETE FROM traffic_logs WHERE user_id=?", id)
	db.DB.Exec("DELETE FROM users WHERE id=?", id)
	logger.Infof("Admin", "Deleted user id=%s", id)
	jsonOK(w, "deleted")
}

func handleResetTraffic(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	db.DB.Exec("UPDATE users SET traffic_up=0, traffic_down=0, last_reset_at=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP WHERE id=?", id)
	logger.Infof("Admin", "Reset traffic for user id=%s", id)
	jsonOK(w, "traffic reset")
}

// ── 节点管理 (admin only) ──

func handleListNodes(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query(
		`SELECT id, name, address, port, protocol, transport, tls,
				COALESCE(tls_sni,''), COALESCE(uuid,''), alter_id,
				COALESCE(extra_config,''), enabled, sort_order,
				COALESCE(agent_id,0), created_at, updated_at
		 FROM nodes ORDER BY sort_order ASC, id ASC`)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var nodes []map[string]interface{}
	for rows.Next() {
		var id, port, tls, alterID, enabled, sortOrder, agentID int
		var name, address, protocol, transport, tlsSNI, uuid, extraConfig string
		var createdAt, updatedAt *string
		rows.Scan(&id, &name, &address, &port, &protocol, &transport, &tls,
			&tlsSNI, &uuid, &alterID, &extraConfig, &enabled, &sortOrder,
			&agentID, &createdAt, &updatedAt)
		nodes = append(nodes, map[string]interface{}{
			"id": id, "name": name, "address": address, "port": port,
			"protocol": protocol, "transport": transport, "tls": tls,
			"tls_sni": tlsSNI, "uuid": uuid, "alter_id": alterID,
			"extra_config": extraConfig, "enabled": enabled,
			"sort_order": sortOrder, "agent_id": agentID,
			"created_at": createdAt, "updated_at": updatedAt,
		})
	}
	if nodes == nil {
		nodes = []map[string]interface{}{}
	}
	jsonOK(w, nodes)
}

func handleCreateNode(w http.ResponseWriter, r *http.Request) {
	var req struct {
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
		SortOrder   int    `json:"sort_order"`
		AgentID     int    `json:"agent_id"`
	}
	if err := parseJSON(r, &req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Port == 0 {
		req.Port = 443
	}
	if req.Protocol == "" {
		req.Protocol = "vmess"
	}
	if req.Transport == "" {
		req.Transport = "tcp"
	}

	result, err := db.DB.Exec(
		`INSERT INTO nodes (name, address, port, protocol, transport, tls, tls_sni, uuid, alter_id, extra_config, sort_order, agent_id)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.Name, req.Address, req.Port, req.Protocol, req.Transport,
		req.TLS, req.TLSSNI, req.UUID, req.AlterID, req.ExtraConfig, req.SortOrder, req.AgentID)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, _ := result.LastInsertId()
	logger.Infof("Admin", "Created node: %s (id=%d)", req.Name, id)
	jsonOK(w, map[string]interface{}{"id": id})
}

func handleUpdateNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req map[string]interface{}
	if err := parseJSON(r, &req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	allowed := map[string]bool{
		"name": true, "address": true, "port": true, "protocol": true,
		"transport": true, "tls": true, "tls_sni": true, "uuid": true,
		"alter_id": true, "extra_config": true, "enabled": true, "sort_order": true,
		"agent_id": true,
	}

	sets := []string{}
	args := []interface{}{}
	for k, v := range req {
		if allowed[k] {
			sets = append(sets, k+"=?")
			args = append(args, v)
		}
	}
	if len(sets) == 0 {
		jsonError(w, "nothing to update", http.StatusBadRequest)
		return
	}
	sets = append(sets, "updated_at=CURRENT_TIMESTAMP")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE nodes SET %s WHERE id=?", strings.Join(sets, ","))
	db.DB.Exec(query, args...)
	logger.Infof("Admin", "Updated node id=%s", id)
	jsonOK(w, "updated")
}

func handleDeleteNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	db.DB.Exec("DELETE FROM user_nodes WHERE node_id=?", id)
	db.DB.Exec("DELETE FROM nodes WHERE id=?", id)
	logger.Infof("Admin", "Deleted node id=%s", id)
	jsonOK(w, "deleted")
}

func handleToggleNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var enabled int
	db.DB.QueryRow("SELECT enabled FROM nodes WHERE id=?", id).Scan(&enabled)
	newVal := 1
	if enabled == 1 {
		newVal = 0
	}
	db.DB.Exec("UPDATE nodes SET enabled=?, updated_at=CURRENT_TIMESTAMP WHERE id=?", newVal, id)
	state := "enabled"
	if newVal == 0 {
		state = "disabled"
	}
	logger.Infof("Admin", "Toggled node id=%s -> %s", id, state)
	jsonOK(w, map[string]interface{}{"enabled": newVal})
}

// ── Agent 管理 (admin) ──

func handleListAgents(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query(
		`SELECT id, name, host, port, status, COALESCE(version,''),
				cpu_usage, mem_usage, net_in, net_out, uptime,
				COALESCE(remark,''), COALESCE(entry_ip,''),
				last_heartbeat, created_at
		 FROM agents ORDER BY id ASC`)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var agents []map[string]interface{}
	for rows.Next() {
		var id, port int
		var netIn, netOut int64
		var uptime int
		var cpuUsage, memUsage float64
		var name, host, status, version, remark, entryIP string
		var lastHB, createdAt *string
		rows.Scan(&id, &name, &host, &port, &status, &version,
			&cpuUsage, &memUsage, &netIn, &netOut, &uptime,
			&remark, &entryIP, &lastHB, &createdAt)
		displayIP := entryIP
		if displayIP == "" {
			displayIP = host
		}
		agents = append(agents, map[string]interface{}{
			"id": id, "name": name, "host": host, "port": port,
			"status": status, "version": version,
			"cpu_usage": cpuUsage, "mem_usage": memUsage,
			"net_in": netIn, "net_out": netOut, "uptime": uptime,
			"remark": remark, "entry_ip": entryIP, "display_ip": displayIP,
			"last_heartbeat": lastHB, "created_at": createdAt,
		})
	}
	if agents == nil {
		agents = []map[string]interface{}{}
	}
	jsonOK(w, agents)
}

func handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Host string `json:"host"`
		Port int    `json:"port"`
	}
	if err := parseJSON(r, &req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Port == 0 {
		req.Port = 9527
	}
	result, err := db.DB.Exec("INSERT INTO agents (name, host, port) VALUES (?, ?, ?)",
		req.Name, req.Host, req.Port)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, _ := result.LastInsertId()
	logger.Infof("Admin", "Created agent: %s (id=%d)", req.Name, id)
	jsonOK(w, map[string]interface{}{"id": id})
}

func handleUpdateAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name *string `json:"name"`
		Host *string `json:"host"`
		Port *int    `json:"port"`
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
	if req.Host != nil {
		sets = append(sets, "host=?")
		args = append(args, *req.Host)
	}
	if req.Port != nil {
		sets = append(sets, "port=?")
		args = append(args, *req.Port)
	}
	if len(sets) == 0 {
		jsonError(w, "nothing to update", http.StatusBadRequest)
		return
	}
	sets = append(sets, "updated_at=CURRENT_TIMESTAMP")
	args = append(args, id)
	db.DB.Exec(fmt.Sprintf("UPDATE agents SET %s WHERE id=?", strings.Join(sets, ",")), args...)
	logger.Infof("Admin", "Updated agent id=%s", id)
	jsonOK(w, "updated")
}

func handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	db.DB.Exec("DELETE FROM agents WHERE id=?", id)
	logger.Infof("Admin", "Deleted agent id=%s", id)
	jsonOK(w, "deleted")
}

func handleGetInstallScript(w http.ResponseWriter, r *http.Request) {
	panelHost := getSetting("panel_host", r.Host)
	commKey := getCommKey()
	script := fmt.Sprintf(`#!/bin/bash
# NebulaPanel Agent Install Script
# Auto-generated - Panel: %s

PANEL_URL="http://%s"
COMM_KEY="%s"
AGENT_PORT=9527
REPO_URL="https://raw.githubusercontent.com/poouo/NebulaPanel/main/agent/nebula-agent.sh"
INSTALL_DIR="/opt/nebula-agent"

install_agent() {
    echo "==> Installing NebulaPanel Agent..."
    mkdir -p "$INSTALL_DIR"

    # Try GitHub first, fallback to panel
    echo "==> Downloading agent script..."
    if ! curl -sL --connect-timeout 10 "$REPO_URL" -o "$INSTALL_DIR/nebula-agent.sh" 2>/dev/null; then
        echo "==> GitHub timeout, downloading from panel..."
        curl -sL "$PANEL_URL/static/agent/nebula-agent.sh" -o "$INSTALL_DIR/nebula-agent.sh"
    fi
    chmod +x "$INSTALL_DIR/nebula-agent.sh"

    # Write config
    cat > "$INSTALL_DIR/agent.conf" <<CONF
PANEL_URL=$PANEL_URL
COMM_KEY=$COMM_KEY
AGENT_PORT=$AGENT_PORT
CONF

    # Create systemd service
    cat > /etc/systemd/system/nebula-agent.service <<EOF
[Unit]
Description=NebulaPanel Agent
After=network.target

[Service]
Type=simple
ExecStart=/bin/bash $INSTALL_DIR/nebula-agent.sh
Restart=always
RestartSec=5
EnvironmentFile=$INSTALL_DIR/agent.conf

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable nebula-agent
    systemctl start nebula-agent
    echo "==> NebulaPanel Agent installed and started!"
    echo "==> Config: $INSTALL_DIR/agent.conf"
}

uninstall_agent() {
    echo "==> Uninstalling NebulaPanel Agent..."
    systemctl stop nebula-agent 2>/dev/null
    systemctl disable nebula-agent 2>/dev/null
    rm -f /etc/systemd/system/nebula-agent.service
    systemctl daemon-reload
    rm -rf "$INSTALL_DIR"
    echo "==> NebulaPanel Agent uninstalled!"
}

case "${1:-install}" in
    install)  install_agent ;;
    uninstall) uninstall_agent ;;
    *) echo "Usage: $0 {install|uninstall}" ;;
esac`, panelHost, panelHost, commKey)

	jsonOK(w, map[string]string{"script": script})
}

// ── Agent 加密通信 ──

func handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	commKey := getCommKey()
	if commKey == "" {
		jsonError(w, "comm key not configured", http.StatusInternalServerError)
		return
	}

	var envelope struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	plaintext, err := crypto.Decrypt(envelope.Data, commKey)
	if err != nil {
		logger.Warnf("Agent", "Heartbeat decrypt failed from %s: %v", getClientIP(r), err)
		jsonError(w, "decrypt failed", http.StatusForbidden)
		return
	}

	var hb struct {
		Hostname string  `json:"hostname"`
		Host     string  `json:"host"`
		CPU      float64 `json:"cpu"`
		Mem      float64 `json:"mem"`
		MemTotal int64   `json:"mem_total"`
		NetIn    int64   `json:"net_in"`
		NetOut   int64   `json:"net_out"`
		Uptime   int     `json:"uptime"`
		Version  string  `json:"version"`
		OS       string  `json:"os"`
		Arch     string  `json:"arch"`
	}
	if err := json.Unmarshal(plaintext, &hb); err != nil {
		jsonError(w, "invalid heartbeat data", http.StatusBadRequest)
		return
	}
	// Use client IP if host not provided
	if hb.Host == "" {
		hb.Host = getClientIP(r)
	}
	if hb.Hostname == "" {
		hb.Hostname = hb.Host
	}
	// Auto-register: check if agent exists, if not create it
	var agentID int
	err = db.DB.QueryRow("SELECT id FROM agents WHERE host=?", hb.Host).Scan(&agentID)
	if err != nil {
		// Agent not found, auto-register
		result, insertErr := db.DB.Exec(
			`INSERT INTO agents (name, host, port, status, version, cpu_usage, mem_usage, net_in, net_out, uptime, last_heartbeat)
			 VALUES (?, ?, 9527, 'online', ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
			hb.Hostname, hb.Host, hb.Version, hb.CPU, hb.Mem, hb.NetIn, hb.NetOut, hb.Uptime)
		if insertErr == nil {
			newID, _ := result.LastInsertId()
			logger.Infof("Agent", "Auto-registered new agent: %s (%s) id=%d", hb.Hostname, hb.Host, newID)
		} else {
			logger.Warnf("Agent", "Failed to auto-register agent %s: %v", hb.Host, insertErr)
		}
	} else {
		// Agent exists, update
		db.DB.Exec(
			`UPDATE agents SET name=?, status='online', cpu_usage=?, mem_usage=?, net_in=?, net_out=?,
			 uptime=?, version=?, last_heartbeat=CURRENT_TIMESTAMP, updated_at=CURRENT_TIMESTAMP
			 WHERE host=?`,
			hb.Hostname, hb.CPU, hb.Mem, hb.NetIn, hb.NetOut, hb.Uptime, hb.Version, hb.Host)
	}
	// Resolve agent id (existing or just inserted)
	if agentID == 0 {
		db.DB.QueryRow("SELECT id FROM agents WHERE host=?", hb.Host).Scan(&agentID)
	}
	respBody := buildAgentSpec(agentID)
	respData, _ := crypto.Encrypt(respBody, commKey)
	jsonOK(w, map[string]string{"data": respData})
}

func handleAgentTraffic(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	commKey := getCommKey()
	if commKey == "" {
		jsonError(w, "comm key not configured", http.StatusInternalServerError)
		return
	}

	var envelope struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}

	plaintext, err := crypto.Decrypt(envelope.Data, commKey)
	if err != nil {
		logger.Warnf("Agent", "Traffic decrypt failed from %s: %v", getClientIP(r), err)
		jsonError(w, "decrypt failed", http.StatusForbidden)
		return
	}

	var report struct {
		Users []struct {
			UserID int   `json:"user_id"`
			Up     int64 `json:"up"`
			Down   int64 `json:"down"`
		} `json:"users"`
	}
	if err := json.Unmarshal(plaintext, &report); err != nil {
		jsonError(w, "invalid traffic data", http.StatusBadRequest)
		return
	}

	hourKey := time.Now().Format("2006-01-02 15:00")
	for _, u := range report.Users {
		// 更新用户总流量
		db.DB.Exec("UPDATE users SET traffic_up=traffic_up+?, traffic_down=traffic_down+?, updated_at=CURRENT_TIMESTAMP WHERE id=?",
			u.Up, u.Down, u.UserID)

		// 写入小时级流量日志
		var existID int
		err := db.DB.QueryRow("SELECT id FROM traffic_logs WHERE user_id=? AND record_at=?", u.UserID, hourKey).Scan(&existID)
		if err != nil {
			db.DB.Exec("INSERT INTO traffic_logs (user_id, traffic_up, traffic_down, record_at) VALUES (?, ?, ?, ?)",
				u.UserID, u.Up, u.Down, hourKey)
		} else {
			db.DB.Exec("UPDATE traffic_logs SET traffic_up=traffic_up+?, traffic_down=traffic_down+? WHERE id=?",
				u.Up, u.Down, existID)
		}
	}

	respData, _ := crypto.Encrypt([]byte(`{"status":"ok"}`), commKey)
	jsonOK(w, map[string]string{"data": respData})
}

// ── 订阅模板 ──

func handleListTemplates(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query("SELECT id, name, content, format, is_default, created_at FROM sub_templates ORDER BY id ASC")
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var templates []map[string]interface{}
	for rows.Next() {
		var id, isDefault int
		var name, content, format string
		var createdAt *string
		rows.Scan(&id, &name, &content, &format, &isDefault, &createdAt)
		templates = append(templates, map[string]interface{}{
			"id": id, "name": name, "content": content, "format": format,
			"is_default": isDefault, "created_at": createdAt,
		})
	}
	if templates == nil {
		templates = []map[string]interface{}{}
	}
	jsonOK(w, templates)
}

func handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string `json:"name"`
		Content   string `json:"content"`
		Format    string `json:"format"`
		IsDefault int    `json:"is_default"`
	}
	if err := parseJSON(r, &req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.IsDefault == 1 {
		db.DB.Exec("UPDATE sub_templates SET is_default=0 WHERE format=?", req.Format)
	}
	result, err := db.DB.Exec("INSERT INTO sub_templates (name, content, format, is_default) VALUES (?, ?, ?, ?)",
		req.Name, req.Content, req.Format, req.IsDefault)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	id, _ := result.LastInsertId()
	logger.Infof("Admin", "Created template: %s (id=%d)", req.Name, id)
	jsonOK(w, map[string]interface{}{"id": id})
}

func handleUpdateTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		Name      *string `json:"name"`
		Content   *string `json:"content"`
		Format    *string `json:"format"`
		IsDefault *int    `json:"is_default"`
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
	if req.Content != nil {
		sets = append(sets, "content=?")
		args = append(args, *req.Content)
	}
	if req.Format != nil {
		sets = append(sets, "format=?")
		args = append(args, *req.Format)
	}
	if req.IsDefault != nil {
		if *req.IsDefault == 1 {
			var fmt string
			db.DB.QueryRow("SELECT format FROM sub_templates WHERE id=?", id).Scan(&fmt)
			db.DB.Exec("UPDATE sub_templates SET is_default=0 WHERE format=?", fmt)
		}
		sets = append(sets, "is_default=?")
		args = append(args, *req.IsDefault)
	}
	if len(sets) == 0 {
		jsonError(w, "nothing to update", http.StatusBadRequest)
		return
	}
	sets = append(sets, "updated_at=CURRENT_TIMESTAMP")
	args = append(args, id)
	db.DB.Exec(fmt.Sprintf("UPDATE sub_templates SET %s WHERE id=?", strings.Join(sets, ",")), args...)
	jsonOK(w, "updated")
}

func handleDeleteTemplate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	db.DB.Exec("DELETE FROM sub_templates WHERE id=?", id)
	logger.Infof("Admin", "Deleted template id=%s", id)
	jsonOK(w, "deleted")
}

// ── 用户-节点分配 ──

func handleGetUserNodes(w http.ResponseWriter, r *http.Request) {
	uid := r.PathValue("id")
	rows, err := db.DB.Query("SELECT node_id FROM user_nodes WHERE user_id=?", uid)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var nodeIDs []int
	for rows.Next() {
		var nid int
		rows.Scan(&nid)
		nodeIDs = append(nodeIDs, nid)
	}
	if nodeIDs == nil {
		nodeIDs = []int{}
	}
	jsonOK(w, nodeIDs)
}

func handleSetUserNodes(w http.ResponseWriter, r *http.Request) {
	uid := r.PathValue("id")
	var req struct {
		NodeIDs []int `json:"node_ids"`
	}
	if err := parseJSON(r, &req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	db.DB.Exec("DELETE FROM user_nodes WHERE user_id=?", uid)
	for _, nid := range req.NodeIDs {
		db.DB.Exec("INSERT INTO user_nodes (user_id, node_id) VALUES (?, ?)", uid, nid)
	}
	logger.Infof("Admin", "Set nodes for user id=%s: %v", uid, req.NodeIDs)
	jsonOK(w, "updated")
}

// ── 流量统计 ──

func handleTrafficStats(w http.ResponseWriter, r *http.Request) {
	uid, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	role := r.Header.Get("X-Role")

	// 管理员可查看指定用户
	if role == "admin" {
		if quid := r.URL.Query().Get("user_id"); quid != "" {
			uid, _ = strconv.Atoi(quid)
		}
	}

	var trafficUp, trafficDown, trafficLimit int64
	var speedLimit int
	var expireAt *string
	db.DB.QueryRow("SELECT traffic_up, traffic_down, traffic_limit, speed_limit, expire_at FROM users WHERE id=?", uid).
		Scan(&trafficUp, &trafficDown, &trafficLimit, &speedLimit, &expireAt)

	jsonOK(w, map[string]interface{}{
		"traffic_up":    trafficUp,
		"traffic_down":  trafficDown,
		"traffic_used":  trafficUp + trafficDown,
		"traffic_limit": trafficLimit,
		"speed_limit":   speedLimit,
		"expire_at":     expireAt,
	})
}

func handleTrafficChart(w http.ResponseWriter, r *http.Request) {
	uid, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	role := r.Header.Get("X-Role")

	if role == "admin" {
		if quid := r.URL.Query().Get("user_id"); quid != "" {
			uid, _ = strconv.Atoi(quid)
		}
	}

	// 日期参数，默认今天
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	// 查询该天 24 小时的数据
	rows, err := db.DB.Query(
		`SELECT record_at, traffic_up, traffic_down FROM traffic_logs
		 WHERE user_id = ? AND record_at >= ? AND record_at < ?
		 ORDER BY record_at ASC`,
		uid, dateStr+" 00:00", dateStr+" 23:59")
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	dataMap := make(map[string][2]int64)
	for rows.Next() {
		var recordAt string
		var up, down int64
		rows.Scan(&recordAt, &up, &down)
		dataMap[recordAt] = [2]int64{up, down}
	}

	// 填充 24 小时
	var chart []map[string]interface{}
	for h := 0; h < 24; h++ {
		hourKey := fmt.Sprintf("%s %02d:00", dateStr, h)
		d := dataMap[hourKey]
		chart = append(chart, map[string]interface{}{
			"hour":    fmt.Sprintf("%02d:00", h),
			"time":    hourKey,
			"up":      d[0],
			"down":    d[1],
			"total":   d[0] + d[1],
		})
	}

	// 同时返回可选日期列表（有数据的日期，最近30天）
	dateRows, _ := db.DB.Query(
		`SELECT DISTINCT substr(record_at, 1, 10) as d FROM traffic_logs
		 WHERE user_id = ? ORDER BY d DESC LIMIT 30`, uid)
	var dates []string
	if dateRows != nil {
		defer dateRows.Close()
		for dateRows.Next() {
			var d string
			dateRows.Scan(&d)
			dates = append(dates, d)
		}
	}

	jsonOK(w, map[string]interface{}{
		"date":    dateStr,
		"chart":   chart,
		"dates":   dates,
	})
}

// ── 订阅接口 ──

func handleSubscription(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "clash"
	}

	userID, active, err := subscription.GetUserBySubToken(token)
	if err != nil {
		http.Error(w, "invalid subscription", http.StatusNotFound)
		return
	}
	if !active {
		http.Error(w, "subscription expired or disabled", http.StatusForbidden)
		return
	}

	content, err := subscription.GenerateForUser(userID, format)
	if err != nil {
		http.Error(w, "failed to generate subscription", http.StatusInternalServerError)
		return
	}

	switch format {
	case "clash", "mihomo":
		w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=clash.yaml")
	case "surge":
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	default:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	// 订阅信息头
	var trafficUp, trafficDown, trafficLimit int64
	var expireAt *string
	db.DB.QueryRow("SELECT traffic_up, traffic_down, traffic_limit, expire_at FROM users WHERE id=?", userID).
		Scan(&trafficUp, &trafficDown, &trafficLimit, &expireAt)
	w.Header().Set("Subscription-Userinfo",
		fmt.Sprintf("upload=%d; download=%d; total=%d", trafficUp, trafficDown, trafficLimit))
	if expireAt != nil && *expireAt != "" {
		t, err := time.Parse("2006-01-02 15:04:05", *expireAt)
		if err == nil {
			w.Header().Set("Subscription-Expire", fmt.Sprintf("%d", t.Unix()))
		}
	}

	w.Write([]byte(content))
}

// ── 系统设置 ──

func handleGetSettings(w http.ResponseWriter, r *http.Request) {
	rows, err := db.DB.Query("SELECT key, value FROM settings")
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		settings[k] = v
	}
	jsonOK(w, settings)
}

func handleGetPublicSettings(w http.ResponseWriter, r *http.Request) {
	allowReg := getSetting("allow_register", "true")
	siteName := getSetting("site_name", "NebulaPanel")
	jsonOK(w, map[string]string{
		"allow_register": allowReg,
		"site_name":      siteName,
	})
}

func handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := parseJSON(r, &req); err != nil {
		jsonError(w, "invalid request", http.StatusBadRequest)
		return
	}
	for k, v := range req {
		db.DB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", k, v)
	}
	logger.Info("Admin", "Settings updated")
	jsonOK(w, "updated")
}

// ── 导入导出 ──

func handleExport(w http.ResponseWriter, r *http.Request) {
	data, err := db.ExportAll()
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=nebula_backup_%s.json", time.Now().Format("20060102_150405")))
	json.NewEncoder(w).Encode(data)
}

func handleImport(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		jsonError(w, "read body failed", http.StatusBadRequest)
		return
	}
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		jsonError(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if err := db.ImportAll(data); err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Info("Admin", "Data imported successfully")
	jsonOK(w, "imported")
}

// ── 日志 ──

func handleGetLogs(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	level := r.URL.Query().Get("level")
	module := r.URL.Query().Get("module")

	logs, total, err := logger.GetLogs(page, pageSize, level, module)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if logs == nil {
		logs = []map[string]interface{}{}
	}
	jsonOK(w, map[string]interface{}{
		"logs":  logs,
		"total": total,
		"page":  page,
	})
}

// ── 仪表盘 ──

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	uid, _ := strconv.Atoi(r.Header.Get("X-User-ID"))
	role := r.Header.Get("X-Role")

	result := make(map[string]interface{})

	// 用户自己的信息
	var trafficUp, trafficDown, trafficLimit int64
	var speedLimit, enabled int
	var expireAt *string
	db.DB.QueryRow("SELECT traffic_up, traffic_down, traffic_limit, speed_limit, expire_at, enabled FROM users WHERE id=?", uid).
		Scan(&trafficUp, &trafficDown, &trafficLimit, &speedLimit, &expireAt, &enabled)

	result["user"] = map[string]interface{}{
		"traffic_up":    trafficUp,
		"traffic_down":  trafficDown,
		"traffic_used":  trafficUp + trafficDown,
		"traffic_limit": trafficLimit,
		"speed_limit":   speedLimit,
		"expire_at":     expireAt,
		"enabled":       enabled,
	}

	// 今日流量趋势（按小时）
	today := time.Now().Format("2006-01-02")
	rows, _ := db.DB.Query(
		`SELECT record_at, traffic_up, traffic_down FROM traffic_logs
		 WHERE user_id = ? AND record_at >= ? AND record_at < ?
		 ORDER BY record_at ASC`,
		uid, today+" 00:00", today+" 23:59")
	dataMap := make(map[string][2]int64)
	if rows != nil {
		for rows.Next() {
			var recordAt string
			var up, down int64
			rows.Scan(&recordAt, &up, &down)
			dataMap[recordAt] = [2]int64{up, down}
		}
		rows.Close()
	}
	var todayChart []map[string]interface{}
	for h := 0; h < 24; h++ {
		hourKey := fmt.Sprintf("%s %02d:00", today, h)
		d := dataMap[hourKey]
		todayChart = append(todayChart, map[string]interface{}{
			"hour":  fmt.Sprintf("%02d:00", h),
			"time":  hourKey,
			"up":    d[0],
			"down":  d[1],
			"total": d[0] + d[1],
		})
	}
	result["today_chart"] = todayChart

	// 管理员额外信息
	if role == "admin" {
		var totalUsers, totalNodes, totalAgents, onlineAgents int
		db.DB.QueryRow("SELECT COUNT(*) FROM users").Scan(&totalUsers)
		db.DB.QueryRow("SELECT COUNT(*) FROM nodes").Scan(&totalNodes)
		db.DB.QueryRow("SELECT COUNT(*) FROM agents").Scan(&totalAgents)
		db.DB.QueryRow("SELECT COUNT(*) FROM agents WHERE status='online'").Scan(&onlineAgents)

		var totalTrafficUp, totalTrafficDown int64
		db.DB.QueryRow("SELECT COALESCE(SUM(traffic_up),0), COALESCE(SUM(traffic_down),0) FROM users").
			Scan(&totalTrafficUp, &totalTrafficDown)

		result["admin"] = map[string]interface{}{
			"total_users":        totalUsers,
			"total_nodes":        totalNodes,
			"total_agents":       totalAgents,
			"online_agents":      onlineAgents,
			"total_traffic_up":   totalTrafficUp,
			"total_traffic_down": totalTrafficDown,
		}
	}

	jsonOK(w, result)
}
