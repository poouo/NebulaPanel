package api

import (
	"fmt"
	"net/http"

	"github.com/poouo/NebulaPanel/internal/auth"
)

func NewRouter() http.Handler {
	mux := http.NewServeMux()

	// ── 公开接口 ──
	mux.HandleFunc("POST /api/login", handleLogin)
	mux.HandleFunc("POST /api/register", handleRegister)
	mux.HandleFunc("GET /api/captcha", handleCaptcha)
	mux.HandleFunc("GET /api/login/need-captcha", handleNeedCaptcha)
	mux.HandleFunc("GET /api/login/challenge", handleLoginChallenge)
	mux.HandleFunc("GET /api/sub/{token}", handleSubscription)
	mux.HandleFunc("GET /api/settings/public", handleGetPublicSettings)

	// ── Agent 加密通信接口 ──
	mux.HandleFunc("POST /api/agent/bootstrap", handleAgentBootstrap)
	mux.HandleFunc("POST /api/agent/heartbeat", handleAgentHeartbeat)
	mux.HandleFunc("POST /api/agent/traffic", handleAgentTraffic)

	// ── 需要认证的接口 ──
	mux.HandleFunc("GET /api/me", authWrap(handleMe))
	mux.HandleFunc("PUT /api/me/password", authWrap(handleChangePassword))

	// 用户管理 (admin)
	mux.HandleFunc("GET /api/users", adminWrap(handleListUsers))
	mux.HandleFunc("POST /api/users", adminWrap(handleCreateUser))
	mux.HandleFunc("PUT /api/users/{id}", adminWrap(handleUpdateUser))
	mux.HandleFunc("DELETE /api/users/{id}", adminWrap(handleDeleteUser))
	mux.HandleFunc("POST /api/users/{id}/reset-traffic", adminWrap(handleResetTraffic))

	// 节点管理 (admin only)
	mux.HandleFunc("GET /api/nodes", adminWrap(handleListNodes))
	mux.HandleFunc("POST /api/nodes", adminWrap(handleCreateNode))
	mux.HandleFunc("PUT /api/nodes/{id}", adminWrap(handleUpdateNode))
	mux.HandleFunc("DELETE /api/nodes/{id}", adminWrap(handleDeleteNode))
	mux.HandleFunc("PUT /api/nodes/{id}/toggle", adminWrap(handleToggleNode))

	// Agent 管理 (admin)
	mux.HandleFunc("GET /api/agents", adminWrap(handleListAgents))
	mux.HandleFunc("POST /api/agents", adminWrap(handleCreateAgent))
	mux.HandleFunc("PUT /api/agents/{id}", adminWrap(handleUpdateAgentMeta))
	mux.HandleFunc("DELETE /api/agents/{id}", adminWrap(handleDeleteAgent))
	mux.HandleFunc("POST /api/agents/{id}/restart", adminWrap(handleRestartAgent))
	mux.HandleFunc("POST /api/agents/{id}/rotate-token", adminWrap(handleRotateAgentToken))
	mux.HandleFunc("POST /api/agents/fast-mode", adminWrap(handleAgentFastMode))
	mux.HandleFunc("GET /api/agents/install-script", adminWrap(handleGetInstallScript))
	mux.HandleFunc("GET /api/agents/{id}/install-script", adminWrap(handleGetAgentInstallScript))

	// 审计规则 (admin)
	mux.HandleFunc("GET /api/audit/rules", adminWrap(handleListAuditRules))
	mux.HandleFunc("POST /api/audit/rules", adminWrap(handleCreateAuditRule))
	mux.HandleFunc("PUT /api/audit/rules/{id}", adminWrap(handleUpdateAuditRule))
	mux.HandleFunc("DELETE /api/audit/rules/{id}", adminWrap(handleDeleteAuditRule))

	// 订阅模板 (admin)
	mux.HandleFunc("GET /api/templates", adminWrap(handleListTemplates))
	mux.HandleFunc("POST /api/templates", adminWrap(handleCreateTemplate))
	mux.HandleFunc("PUT /api/templates/{id}", adminWrap(handleUpdateTemplate))
	mux.HandleFunc("DELETE /api/templates/{id}", adminWrap(handleDeleteTemplate))

	// 用户-节点分配 (admin)
	mux.HandleFunc("GET /api/users/{id}/nodes", adminWrap(handleGetUserNodes))
	mux.HandleFunc("PUT /api/users/{id}/nodes", adminWrap(handleSetUserNodes))

	// 流量统计
	mux.HandleFunc("GET /api/traffic/stats", authWrap(handleTrafficStats))
	mux.HandleFunc("GET /api/traffic/chart", authWrap(handleTrafficChart))

	// 系统设置 (admin)
	mux.HandleFunc("GET /api/settings", adminWrap(handleGetSettings))
	mux.HandleFunc("PUT /api/settings", adminWrap(handleUpdateSettings))

	// 导入导出 (admin)
	mux.HandleFunc("GET /api/export", adminWrap(handleExport))
	mux.HandleFunc("POST /api/import", adminWrap(handleImport))

	// 日志 (admin)
	mux.HandleFunc("GET /api/logs", adminWrap(handleGetLogs))

	// 仪表盘
	mux.HandleFunc("GET /api/dashboard", authWrap(handleDashboard))

	// 静态文件
	fs := http.FileServer(http.Dir("web/static"))
	mux.Handle("GET /static/", http.StripPrefix("/static/", fs))
	mux.HandleFunc("GET /", handleIndex)

	return corsMiddleware(mux)
}

// ── 中间件 ──

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func authWrap(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := auth.ExtractToken(r)
		if token == "" {
			jsonError(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		claims, err := auth.ParseToken(token)
		if err != nil {
			jsonError(w, "invalid token", http.StatusUnauthorized)
			return
		}
		r.Header.Set("X-User-ID", fmt.Sprintf("%d", claims.UserID))
		r.Header.Set("X-Username", claims.Username)
		r.Header.Set("X-Role", claims.Role)
		fn(w, r)
	}
}

func adminWrap(fn http.HandlerFunc) http.HandlerFunc {
	return authWrap(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Role") != "admin" {
			jsonError(w, "forbidden", http.StatusForbidden)
			return
		}
		fn(w, r)
	})
}
