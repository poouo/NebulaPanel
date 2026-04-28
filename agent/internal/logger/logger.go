package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

const retentionDays = 30

var (
	stdLogger  *log.Logger
	logDir     string
	maxAgeDays = retentionDays
)

// Init prepares rotating file logger. logs are kept for 30 days.
func Init(dir string) error {
	if dir == "" {
		dir = "/var/log/nebula-agent"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	logDir = dir

	rot := &lumberjack.Logger{
		Filename:   filepath.Join(dir, "agent.log"),
		MaxSize:    20, // MB per file
		MaxBackups: 100,
		MaxAge:     retentionDays, // days
		Compress:   true,
		LocalTime:  true,
	}

	mw := io.MultiWriter(os.Stdout, rot)
	stdLogger = log.New(mw, "", 0)

	go startCleaner()
	Infof("Logger", "log dir=%s retention=%d days", dir, retentionDays)
	return nil
}

// Infof / Warnf / Errorf write structured lines: TS LEVEL [module] message
func write(level, module, msg string) {
	if stdLogger == nil {
		log.Printf("[%s] [%s] %s", level, module, msg)
		return
	}
	ts := time.Now().Format("2006-01-02 15:04:05")
	stdLogger.Printf("%s %s [%s] %s", ts, strings.ToUpper(level), module, msg)
}

func Infof(module, format string, a ...interface{}) {
	write("info", module, fmt.Sprintf(format, a...))
}

func Warnf(module, format string, a ...interface{}) {
	write("warn", module, fmt.Sprintf(format, a...))
}

func Errorf(module, format string, a ...interface{}) {
	write("error", module, fmt.Sprintf(format, a...))
}

// startCleaner removes rotated log files older than retentionDays as a safety
// net (lumberjack already enforces MaxAge but old leftover files might remain).
func startCleaner() {
	t := time.NewTicker(6 * time.Hour)
	defer t.Stop()
	for {
		cleanup()
		<-t.C
	}
}

func cleanup() {
	if logDir == "" {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fi, err := e.Info()
		if err != nil {
			continue
		}
		if fi.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(logDir, e.Name()))
		}
	}
}
