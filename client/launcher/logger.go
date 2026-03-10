package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	appLogger   *log.Logger
	logFile     *os.File
	logInitOnce sync.Once
)

// initLogger инициализирует файловый логгер с ротацией.
// Логи пишутся в %APPDATA%/FamMCLauncher/logs/latest.log (Windows)
// или в ~/.config/FamMCLauncher/logs/latest.log на других платформах.
func initLogger() {
	logInitOnce.Do(func() {
		dir, err := getLogsDir()
		if err != nil {
			return
		}
		if err := os.MkdirAll(dir, dirMode); err != nil {
			return
		}

		latest := filepath.Join(dir, "latest.log")
		// Ротация предыдущего latest → YYYY-MM-DD-N.log
		if _, err := os.Stat(latest); err == nil {
			date := time.Now().Format("2006-01-02")
			maxN := 0
			entries, err := os.ReadDir(dir)
			if err == nil {
				prefix := date + "-"
				for _, e := range entries {
					if e.IsDir() {
						continue
					}
					name := e.Name()
					if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".log") {
						continue
					}
					middle := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".log")
					if n, err := strconv.Atoi(middle); err == nil && n > maxN {
						maxN = n
					}
				}
			}
			rotated := filepath.Join(dir, fmt.Sprintf("%s-%d.log", date, maxN+1))
			_ = os.Rename(latest, rotated)
		}

		f, err := os.OpenFile(latest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, configFileMode)
		if err != nil {
			return
		}
		logFile = f
		appLogger = log.New(f, "", log.LstdFlags|log.Lmicroseconds)
		logInfo("logger", "логирование инициализировано, файл: %s", latest)
	})
}

// getLogsDir возвращает директорию для логов.
func getLogsDir() (string, error) {
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "FamMCLauncher", "logs"), nil
		}
	}
	// fallback: рядом с конфигом
	base, err := getAppConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "logs"), nil
}

func logInfo(category, format string, args ...any) {
	logPrintf("INFO", category, format, args...)
}

func logError(category, format string, args ...any) {
	logPrintf("ERROR", category, format, args...)
}

func logDebug(category, format string, args ...any) {
	logPrintf("DEBUG", category, format, args...)
}

func logPrintf(level, category, format string, args ...any) {
	msg := fmt.Sprintf("[%s] [%s] %s", level, category, fmt.Sprintf(format, args...))
	if appLogger != nil {
		appLogger.Println(msg)
	} else {
		log.Println(msg)
	}
}

