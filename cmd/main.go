package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/poouo/NebulaPanel/internal/api"
	"github.com/poouo/NebulaPanel/internal/auth"
	"github.com/poouo/NebulaPanel/internal/captcha"
	"github.com/poouo/NebulaPanel/internal/crypto"
	"github.com/poouo/NebulaPanel/internal/db"
	"github.com/poouo/NebulaPanel/internal/logger"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("=== NebulaPanel Starting ===")

	// 配置
	dbPath := getEnv("DB_PATH", "/data/nebula.db")
	listenAddr := getEnv("LISTEN", ":3001")
	jwtSecret := getEnv("JWT_SECRET", "")
	adminUser := getEnv("ADMIN_USER", "admin")
	adminPass := getEnv("ADMIN_PASS", "admin123")

	// 初始化数据库
	db.Init(dbPath)
	defer db.Close()

	// 初始化 JWT
	auth.Init(jwtSecret)

	// 初始化通信密钥（如果不存在则自动生成）
	initCommKey()

	// 创建默认管理员
	ensureAdmin(adminUser, adminPass)
	// 创建默认订阅模板
	ensureDefaultTemplates()

	// 启动日志清理器（保留30天）
	logger.StartLogCleaner()

	// 启动验证码失败记录清理器
	captcha.StartFailCleaner()

	// 启动流量重置检查器
	startTrafficResetChecker()

	// 启动 Agent 离线检测
	startAgentOfflineChecker()

	// 启动流量日志清理（保留30天）
	startTrafficLogCleaner()

	// 路由
	router := api.NewRouter()

	log.Printf("NebulaPanel listening on %s", listenAddr)
	logger.Info("System", fmt.Sprintf("NebulaPanel started on %s", listenAddr))

	if err := http.ListenAndServe(listenAddr, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func initCommKey() {
	var existing string
	db.DB.QueryRow("SELECT value FROM settings WHERE key='comm_key'").Scan(&existing)
	if existing == "" {
		key, err := crypto.GenerateKey()
		if err != nil {
			log.Fatalf("Failed to generate comm key: %v", err)
		}
		db.DB.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('comm_key', ?)", key)
		log.Printf("[Init] Generated communication key: %s", key[:16]+"...")
		logger.Info("System", "Communication key generated")
	} else {
		log.Println("[Init] Communication key loaded")
	}

	// 默认设置
	defaults := map[string]string{
		"site_name":      "NebulaPanel",
		"allow_register": "true",
		"panel_host":     "",
	}
	for k, v := range defaults {
		var val string
		err := db.DB.QueryRow("SELECT value FROM settings WHERE key=?", k).Scan(&val)
		if err != nil {
			db.DB.Exec("INSERT INTO settings (key, value) VALUES (?, ?)", k, v)
		}
	}
}

func ensureAdmin(username, password string) {
	var count int
	db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE role='admin'").Scan(&count)
	if count == 0 {
		hash, _ := auth.HashPassword(password)
		subToken := auth.GenerateSubToken()
		db.DB.Exec(
			`INSERT INTO users (username, password, role, sub_token) VALUES (?, ?, 'admin', ?)`,
			username, hash, subToken)
		log.Printf("[Init] Default admin created: %s / %s", username, password)
		logger.Infof("System", "Default admin created: %s", username)
	}
}

// 每小时检查一次流量重置
func startTrafficResetChecker() {
	go func() {
		for {
			day := time.Now().Day()
			rows, err := db.DB.Query(
				"SELECT id FROM users WHERE reset_day = ? AND (last_reset_at IS NULL OR last_reset_at < ?)",
				day, time.Now().Format("2006-01-02"))
			if err == nil {
				for rows.Next() {
					var uid int
					rows.Scan(&uid)
					db.DB.Exec("UPDATE users SET traffic_up=0, traffic_down=0, last_reset_at=CURRENT_TIMESTAMP WHERE id=?", uid)
					logger.Infof("System", "Auto-reset traffic for user id=%d (reset_day=%d)", uid, day)
				}
				rows.Close()
			}
			time.Sleep(1 * time.Hour)
		}
	}()
}

// 每分钟检查 Agent 是否离线（超过3分钟无心跳）
func startAgentOfflineChecker() {
	go func() {
		for {
			cutoff := time.Now().Add(-3 * time.Minute).Format("2006-01-02 15:04:05")
			db.DB.Exec("UPDATE agents SET status='offline' WHERE status='online' AND last_heartbeat < ?", cutoff)
			time.Sleep(1 * time.Minute)
		}
	}()
}

// 每天清理超过30天的流量日志
func startTrafficLogCleaner() {
	go func() {
		for {
			cutoff := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
			result, err := db.DB.Exec("DELETE FROM traffic_logs WHERE record_at < ?", cutoff)
			if err == nil {
				affected, _ := result.RowsAffected()
				if affected > 0 {
					logger.Infof("System", "Cleaned %d old traffic log entries", affected)
				}
			}
			time.Sleep(24 * time.Hour)
		}
	}()
}

// 创建默认订阅模板（如果不存在）
func ensureDefaultTemplates() {
	var count int
	db.DB.QueryRow("SELECT COUNT(*) FROM sub_templates").Scan(&count)
	if count > 0 {
		return
	}

	clashTemplate := "# NebulaPanel - Clash/Mihomo Default Template\n" +
		"# Variables: {{PROXIES}} = proxy list, {{PROXY_NAMES}} = proxy name list\n\n" +
		"mixed-port: 7890\n" +
		"allow-lan: false\n" +
		"bind-address: '*'\n" +
		"mode: rule\n" +
		"log-level: info\n" +
		"unified-delay: true\n" +
		"external-controller: 127.0.0.1:9090\n\n" +
		"dns:\n" +
		"  enable: true\n" +
		"  listen: 0.0.0.0:53\n" +
		"  enhanced-mode: fake-ip\n" +
		"  fake-ip-range: 198.18.0.1/16\n" +
		"  nameserver:\n" +
		"    - 223.5.5.5\n" +
		"    - 119.29.29.29\n" +
		"  fallback:\n" +
		"    - 8.8.8.8\n" +
		"    - 1.1.1.1\n" +
		"  fallback-filter:\n" +
		"    geoip: true\n" +
		"    geoip-code: CN\n\n" +
		"proxies:\n" +
		"{{PROXIES}}\n\n" +
		"proxy-groups:\n" +
		"  - name: Proxy\n" +
		"    type: select\n" +
		"    proxies:\n" +
		"      - Auto\n" +
		"      - DIRECT\n" +
		"{{PROXY_NAMES}}\n\n" +
		"  - name: Auto\n" +
		"    type: url-test\n" +
		"    url: http://www.gstatic.com/generate_204\n" +
		"    interval: 300\n" +
		"    tolerance: 50\n" +
		"    proxies:\n" +
		"{{PROXY_NAMES}}\n\n" +
		"rules:\n" +
		"  - GEOIP,LAN,DIRECT\n" +
		"  - GEOIP,CN,DIRECT\n" +
		"  - MATCH,Proxy\n"

	db.DB.Exec(
		"INSERT INTO sub_templates (name, content, format, is_default) VALUES (?, ?, 'clash', 1)",
		"Default Clash Template", clashTemplate)

	log.Println("[Init] Default Clash subscription template created")
	logger.Info("System", "Default Clash subscription template created")
}
