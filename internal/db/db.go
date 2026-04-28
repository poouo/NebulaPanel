package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
		log.Println("[DB] Database initialized successfully")
	})
}

func migrate() {
	tables := []string{
		// ── 用户表：含流量限制、速率限制、到期时间、流量重置周期 ──
		`CREATE TABLE IF NOT EXISTS users (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			username        TEXT UNIQUE NOT NULL,
			password        TEXT NOT NULL,
			role            TEXT NOT NULL DEFAULT 'user',

			-- 流量（单位 Bytes）
			traffic_up      INTEGER NOT NULL DEFAULT 0,
			traffic_down    INTEGER NOT NULL DEFAULT 0,
			traffic_limit   INTEGER NOT NULL DEFAULT 0,

			-- 速率限制（单位 Mbps，0=不限）
			speed_limit     INTEGER NOT NULL DEFAULT 0,

			-- 到期 & 重置
			expire_at       TEXT,
			reset_day       INTEGER NOT NULL DEFAULT 0,
			last_reset_at   TEXT,

			-- 订阅
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
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		// ── Agent 从机表 ──
		`CREATE TABLE IF NOT EXISTS agents (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			name            TEXT NOT NULL,
			host            TEXT NOT NULL,
			port            INTEGER NOT NULL DEFAULT 9527,
			status          TEXT NOT NULL DEFAULT 'offline',
			version         TEXT,
			cpu_usage       REAL    DEFAULT 0,
			mem_usage       REAL    DEFAULT 0,
			net_in          INTEGER DEFAULT 0,
			net_out         INTEGER DEFAULT 0,
			uptime          INTEGER DEFAULT 0,
			last_heartbeat  DATETIME,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
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

		// ── 流量记录（按小时聚合，用于趋势图） ──
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

		// ── 索引 ──
		`CREATE INDEX IF NOT EXISTS idx_logs_created_at ON logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_users_sub_token ON users(sub_token)`,
		`CREATE INDEX IF NOT EXISTS idx_traffic_logs_user ON traffic_logs(user_id, record_at)`,
		`CREATE INDEX IF NOT EXISTS idx_traffic_logs_time ON traffic_logs(record_at)`,
	}
	for _, t := range tables {
		if _, err := DB.Exec(t); err != nil {
			log.Fatalf("Failed to migrate: %v\nSQL: %s", err, t)
		}
	}
	log.Println("[DB] Migration completed")
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
	tableNames := []string{"users", "nodes", "agents", "sub_templates", "user_nodes", "settings", "traffic_logs", "logs"}
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

	orderedTables := []string{"settings", "users", "nodes", "sub_templates", "agents", "user_nodes", "traffic_logs", "logs"}
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
