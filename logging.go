package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func logPath() string {
	return filepath.Join(configDir, "hop.log")
}

func clearLog() error {
	if _, err := os.Stat(logPath()); os.IsNotExist(err) {
		return nil
	}
	return os.Truncate(logPath(), 0)
}

func rotateLogIfNewDay() {
	info, err := os.Stat(logPath())
	if err != nil {
		return // belum ada log sama sekali
	}
	today := time.Now().Format("2006-01-02")
	lastWritten := info.ModTime().Format("2006-01-02")
	if lastWritten != today {
		os.Truncate(logPath(), 0)
	}
}

func logEvent(alias string, format string, args ...interface{}) {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return
	}
	f, err := os.OpenFile(logPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	ts := time.Now().Format("2006-01-02 15:04:05")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(f, "[%s] [%s] %s\n", ts, alias, msg)
}
