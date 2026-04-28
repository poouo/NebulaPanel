package logger

import (
	"fmt"
	"log"
	"time"

	"github.com/poouo/NebulaPanel/internal/db"
)

const (
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
	RetentionDays = 30
)

func Log(level, module, message string) {
	log.Printf("[%s] [%s] %s", level, module, message)
	if db.DB != nil {
		db.DB.Exec("INSERT INTO logs (level, module, message) VALUES (?, ?, ?)",
			level, module, message)
	}
}

func Info(module, message string) {
	Log(LevelInfo, module, message)
}

func Warn(module, message string) {
	Log(LevelWarn, module, message)
}

func Error(module, msg string) {
	Log(LevelError, module, msg)
}

func Infof(module, format string, args ...interface{}) {
	Info(module, fmt.Sprintf(format, args...))
}

func Warnf(module, format string, args ...interface{}) {
	Warn(module, fmt.Sprintf(format, args...))
}

func Errorf(module, format string, args ...interface{}) {
	Error(module, fmt.Sprintf(format, args...))
}

// CleanOldLogs removes logs older than RetentionDays
func CleanOldLogs() {
	if db.DB == nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -RetentionDays).Format("2006-01-02 15:04:05")
	result, err := db.DB.Exec("DELETE FROM logs WHERE created_at < ?", cutoff)
	if err != nil {
		log.Printf("[Logger] Failed to clean old logs: %v", err)
		return
	}
	affected, _ := result.RowsAffected()
	if affected > 0 {
		log.Printf("[Logger] Cleaned %d old log entries", affected)
	}
}

// StartLogCleaner starts a background goroutine to clean old logs daily
func StartLogCleaner() {
	go func() {
		for {
			CleanOldLogs()
			time.Sleep(24 * time.Hour)
		}
	}()
	log.Println("[Logger] Log cleaner started (retention: 30 days)")
}

// GetLogs retrieves logs with pagination
func GetLogs(page, pageSize int, level, module string) ([]map[string]interface{}, int, error) {
	countQuery := "SELECT COUNT(*) FROM logs WHERE 1=1"
	query := "SELECT id, level, module, message, created_at FROM logs WHERE 1=1"
	args := []interface{}{}

	if level != "" {
		countQuery += " AND level = ?"
		query += " AND level = ?"
		args = append(args, level)
	}
	if module != "" {
		countQuery += " AND module = ?"
		query += " AND module = ?"
		args = append(args, module)
	}

	var total int
	db.DB.QueryRow(countQuery, args...).Scan(&total)

	query += " ORDER BY id DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []map[string]interface{}
	for rows.Next() {
		var id int
		var lvl, mod, msg, createdAt string
		rows.Scan(&id, &lvl, &mod, &msg, &createdAt)
		logs = append(logs, map[string]interface{}{
			"id":         id,
			"level":      lvl,
			"module":     mod,
			"message":    msg,
			"created_at": createdAt,
		})
	}
	return logs, total, nil
}
