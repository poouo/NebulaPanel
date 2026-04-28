package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

var (
	DB   *sql.DB
	once sync.Once
)

func Init(dbPath string) {
	once.Do(func() {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create db directory: %v", err)
		}
		var err error
		DB, err = sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}
		DB.SetMaxOpenConns(1)
		if err = DB.Ping(); err != nil {
			log.Fatalf("Failed to ping database: %v", err)
		}
		migrate()
		alter()
		createIndexes()
		log.Println("[DB] Database initialized successfully")
	})
}

func migrate() {
	tables := []string{
		// ── 用户表 ──
		`CREATE TABLE IF NOT EXISTS users (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			username        TEXT UNIQUE NOT NULL,
			password        TEXT NOT NULL,
			role            TEXT NOT NULL DEFAULT 'user',
			traffic_up      INTEGER NOT NULL DEFAULT 0,
			traffic_down    INTEGER NOT NULL DEFAULT 0,
			traffic_limit   INTEGER NOT NULL DEFAULT 0,
			speed_limit     INTEGER NOT NULL DEFAULT 0,
			expire_at       TEXT,
			reset_day       INTEGER NOT NULL DEFAULT 0,
			last_reset_at   TEXT,
			sub_token       TEXT UNIQUE,
			enabled         INTEGER NOT NULL DEFAULT 1,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// ── 节点表 ──
		`CREATE TABLE IF NOT EXISTS nodes (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			name            TEXT NOT NULL,
			address         TEXT NOT NULL,
			port            INTEGER NOT NULL DEFAULT 443,
			protocol        TEXT NOT NULL DEFAULT 'vmess',
			transport       TEXT NOT NULL DEFAULT 'tcp',
			tls             INTEGER NOT NULL DEFAULT 0,
			tls_sni         TEXT,
			uuid            TEXT,
			alter_id        INTEGER NOT NULL DEFAULT 0,
			extra_config    TEXT,
			enabled         INTEGER NOT NULL DEFAULT 1,
			sort_order      INTEGER NOT NULL DEFAULT 0,
			agent_id        INTEGER NOT NULL DEFAULT 0,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// ── Agent 表 ──
		`CREATE TABLE IF NOT EXISTS agents (
			id                 INTEGER PRIMARY KEY AUTOINCREMENT,
			name               TEXT NOT NULL,
			host               TEXT NOT NULL,
			port               INTEGER NOT NULL DEFAULT 9527,
			status             TEXT NOT NULL DEFAULT 'offline',
			token              TEXT DEFAULT '',
			version            TEXT,
			xray_version       TEXT DEFAULT '',
			cpu_usage          REAL    DEFAULT 0,
			cpu_cores          INTEGER NOT NULL DEFAULT 0,
			cpu_model          TEXT    DEFAULT '',
			mem_usage          REAL    DEFAULT 0,
			mem_total          INTEGER NOT NULL DEFAULT 0,
			disk_total         INTEGER NOT NULL DEFAULT 0,
			disk_used          INTEGER NOT NULL DEFAULT 0,
			disk_usage         REAL NOT NULL DEFAULT 0,
			os_info            TEXT    DEFAULT '',
			kernel             TEXT    DEFAULT '',
			arch               TEXT    DEFAULT '',
			load_avg           TEXT    DEFAULT '',
			net_in             INTEGER DEFAULT 0,
			net_out            INTEGER DEFAULT 0,
			net_in_speed       INTEGER NOT NULL DEFAULT 0,
			net_out_speed      INTEGER NOT NULL DEFAULT 0,
			uptime             INTEGER DEFAULT 0,
			remark             TEXT    DEFAULT '',
			entry_ip           TEXT    DEFAULT '',
			restart_pending    INTEGER NOT NULL DEFAULT 0,
			report_fast_until  DATETIME,
			last_heartbeat     DATETIME,
			created_at         DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at         DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// ── 订阅模板表 ──
		`CREATE TABLE IF NOT EXISTS sub_templates (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			name            TEXT NOT NULL,
			content         TEXT NOT NULL,
			format          TEXT NOT NULL DEFAULT 'clash',
			is_default      INTEGER NOT NULL DEFAULT 0,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// ── 用户-节点关联 ──
		`CREATE TABLE IF NOT EXISTS user_nodes (
			user_id  INTEGER NOT NULL,
			node_id  INTEGER NOT NULL,
			PRIMARY KEY (user_id, node_id),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (node_id) REFERENCES nodes(id) ON DELETE CASCADE
		)`,

		// ── 流量记录 ──
		`CREATE TABLE IF NOT EXISTS traffic_logs (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id     INTEGER NOT NULL,
			node_id     INTEGER NOT NULL DEFAULT 0,
			traffic_up  INTEGER NOT NULL DEFAULT 0,
			traffic_down INTEGER NOT NULL DEFAULT 0,
			record_at   TEXT NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,

		// ── 系统设置 ──
		`CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,

		// ── 操作日志 ──
		`CREATE TABLE IF NOT EXISTS logs (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			level      TEXT NOT NULL DEFAULT 'info',
			module     TEXT NOT NULL,
			message    TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// ── 审计规则表（默认关闭，下发到 Agent 阻断指定网址） ──
		`CREATE TABLE IF NOT EXISTS audit_rules (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			domain      TEXT NOT NULL,
			remark      TEXT DEFAULT '',
			enabled     INTEGER NOT NULL DEFAULT 1,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

	}
	for _, t := range tables {
		if _, err := DB.Exec(t); err != nil {
			log.Fatalf("Failed to migrate: %v\nSQL: %s", err, t)
		}
	}
	log.Println("[DB] Migration completed")
}

// createIndexes runs after migrate() + alter(), so that columns added to old
// databases by ALTER TABLE are already present when we reference them.
// Any single failure here is logged as a warning rather than being fatal,
// because indexes are purely for performance and we never want a stale
// index definition to block the panel from starting up.
func createIndexes() {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_logs_created_at  ON logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_users_sub_token  ON users(sub_token)`,
		`CREATE INDEX IF NOT EXISTS idx_traffic_logs_user ON traffic_logs(user_id, record_at)`,
		`CREATE INDEX IF NOT EXISTS idx_traffic_logs_time ON traffic_logs(record_at)`,
		`CREATE INDEX IF NOT EXISTS idx_nodes_agent       ON nodes(agent_id)`,
	}
	for _, q := range indexes {
		if _, err := DB.Exec(q); err != nil {
			log.Printf("[DB] index warning: %v (sql=%s)", err, q)
		}
	}
}

// alter performs idempotent ALTER TABLE for columns added in newer versions.
func alter() {
	type col struct {
		Table, Name, DDL string
	}
	cols := []col{
		{"agents", "remark", "ALTER TABLE agents ADD COLUMN remark TEXT DEFAULT ''"},
		{"agents", "entry_ip", "ALTER TABLE agents ADD COLUMN entry_ip TEXT DEFAULT ''"},
		{"nodes", "agent_id", "ALTER TABLE nodes ADD COLUMN agent_id INTEGER NOT NULL DEFAULT 0"},
		// v2.1: agent token-based auto registration & richer host metrics
		{"agents", "token", "ALTER TABLE agents ADD COLUMN token TEXT DEFAULT ''"},
		{"agents", "cpu_cores", "ALTER TABLE agents ADD COLUMN cpu_cores INTEGER NOT NULL DEFAULT 0"},
		{"agents", "cpu_model", "ALTER TABLE agents ADD COLUMN cpu_model TEXT DEFAULT ''"},
		{"agents", "mem_total", "ALTER TABLE agents ADD COLUMN mem_total INTEGER NOT NULL DEFAULT 0"},
		{"agents", "disk_total", "ALTER TABLE agents ADD COLUMN disk_total INTEGER NOT NULL DEFAULT 0"},
		{"agents", "disk_used", "ALTER TABLE agents ADD COLUMN disk_used INTEGER NOT NULL DEFAULT 0"},
		{"agents", "disk_usage", "ALTER TABLE agents ADD COLUMN disk_usage REAL NOT NULL DEFAULT 0"},
		{"agents", "os_info", "ALTER TABLE agents ADD COLUMN os_info TEXT DEFAULT ''"},
		{"agents", "kernel", "ALTER TABLE agents ADD COLUMN kernel TEXT DEFAULT ''"},
		{"agents", "arch", "ALTER TABLE agents ADD COLUMN arch TEXT DEFAULT ''"},
		{"agents", "xray_version", "ALTER TABLE agents ADD COLUMN xray_version TEXT DEFAULT ''"},
		{"agents", "load_avg", "ALTER TABLE agents ADD COLUMN load_avg TEXT DEFAULT ''"},
		{"agents", "net_in_speed", "ALTER TABLE agents ADD COLUMN net_in_speed INTEGER NOT NULL DEFAULT 0"},
		{"agents", "net_out_speed", "ALTER TABLE agents ADD COLUMN net_out_speed INTEGER NOT NULL DEFAULT 0"},
		{"agents", "restart_pending", "ALTER TABLE agents ADD COLUMN restart_pending INTEGER NOT NULL DEFAULT 0"},
		{"agents", "report_fast_until", "ALTER TABLE agents ADD COLUMN report_fast_until DATETIME"},
	}
	for _, c := range cols {
		if !columnExists(c.Table, c.Name) {
			if _, err := DB.Exec(c.DDL); err != nil && !strings.Contains(err.Error(), "duplicate") {
				log.Printf("[DB] alter %s.%s warning: %v", c.Table, c.Name, err)
			}
		}
	}

	// default settings
	defaults := map[string]string{
		"audit_enabled": "false",
		"github_url":    "https://github.com/poouo/NebulaPanel",
	}
	for k, v := range defaults {
		var val string
		if err := DB.QueryRow("SELECT value FROM settings WHERE key=?", k).Scan(&val); err != nil {
			DB.Exec("INSERT INTO settings (key, value) VALUES (?, ?)", k, v)
		}
	}
}

func columnExists(table, col string) bool {
	rows, err := DB.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err == nil {
			if strings.EqualFold(name, col) {
				return true
			}
		}
	}
	return false
}

func Close() {
	if DB != nil {
		DB.Close()
	}
}

// ────────────────────────────────────────
// 导出 / 导入
// ────────────────────────────────────────

func ExportAll() (map[string]interface{}, error) {
	result := make(map[string]interface{})
	tableNames := []string{"users", "nodes", "agents", "sub_templates", "user_nodes", "settings", "traffic_logs", "logs", "audit_rules"}
	for _, table := range tableNames {
		rows, err := DB.Query(fmt.Sprintf("SELECT * FROM %s", table))
		if err != nil {
			return nil, fmt.Errorf("export table %s: %w", table, err)
		}
		cols, _ := rows.Columns()
		var records []map[string]interface{}
		for rows.Next() {
			values := make([]interface{}, len(cols))
			valuePtrs := make([]interface{}, len(cols))
			for i := range values {
				valuePtrs[i] = &values[i]
			}
			rows.Scan(valuePtrs...)
			record := make(map[string]interface{})
			for i, col := range cols {
				val := values[i]
				if b, ok := val.([]byte); ok {
					record[col] = string(b)
				} else {
					record[col] = val
				}
			}
			records = append(records, record)
		}
		rows.Close()
		result[table] = records
	}
	return result, nil
}

func ImportAll(data map[string]interface{}) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	orderedTables := []string{"settings", "users", "nodes", "sub_templates", "agents", "user_nodes", "traffic_logs", "logs", "audit_rules"}
	for _, table := range orderedTables {
		tx.Exec(fmt.Sprintf("DELETE FROM %s", table))
	}

	for _, table := range orderedTables {
		records, ok := data[table]
		if !ok {
			continue
		}
		recordList, ok := records.([]interface{})
		if !ok {
			continue
		}
		for _, r := range recordList {
			record, ok := r.(map[string]interface{})
			if !ok {
				continue
			}
			cols := make([]string, 0, len(record))
			vals := make([]interface{}, 0, len(record))
			placeholders := make([]string, 0, len(record))
			for k, v := range record {
				cols = append(cols, k)
				vals = append(vals, v)
				placeholders = append(placeholders, "?")
			}
			query := fmt.Sprintf("INSERT OR REPLACE INTO %s (%s) VALUES (%s)",
				table, joinStrings(cols, ","), joinStrings(placeholders, ","))
			if _, err := tx.Exec(query, vals...); err != nil {
				log.Printf("[DB] Import warning for table %s: %v", table, err)
			}
		}
	}
	return tx.Commit()
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
