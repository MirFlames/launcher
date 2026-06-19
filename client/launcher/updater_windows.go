//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// ApplyLauncherUpdate скачивает новую версию лаунчера и обновляет текущий бинарник на месте.
// Использует bat-скрипт, чтобы ярлыки продолжали указывать на launcher.exe.
func (a *App) ApplyLauncherUpdate() error {
	logInfo("update", "начало ApplyLauncherUpdate (windows), текущая версия: %s", LauncherVersion)
	manifest := lastUpdateManifest
	if manifest == nil {
		m, err := fetchUpdateManifest()
		if err != nil {
			logError("update", "ошибка fetchUpdateManifest в ApplyLauncherUpdate: %v", err)
			return err
		}
		if m == nil {
			return fmt.Errorf("обновление не найдено")
		}
		manifest = m
	}
	if compareVersions(manifest.Version, LauncherVersion) <= 0 {
		return fmt.Errorf("обновление не требуется")
	}

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("не удалось определить путь к текущему exe: %w", err)
	}
	currentExe, err = filepath.Abs(currentExe)
	if err != nil {
		return fmt.Errorf("не удалось нормализовать путь к exe: %w", err)
	}

	dir := filepath.Dir(currentExe)
	newPath := filepath.Join(dir, "launcher.new.exe")
	logInfo("update", "windows update: currentExe=%s newPath=%s", currentExe, newPath)

	exePath, err := downloadUpdateBinary(manifest)
	if err != nil {
		logError("update", "ошибка downloadUpdateBinary: %v", err)
		return err
	}
	if err := copyFile(exePath, newPath); err != nil {
		logError("update", "ошибка copyFile(%s → %s): %v", exePath, newPath, err)
		return fmt.Errorf("копирование обновления: %w", err)
	}
	logInfo("update", "обновление скопировано в %s", newPath)

	scriptPath := filepath.Join(os.TempDir(), "launcher-update-"+manifest.Version+".bat")
	scriptContents := "@echo off\r\nsetlocal ENABLEDELAYEDEXPANSION\r\n:loop\r\nping 127.0.0.1 -n 2 >NUL\r\ncopy /Y \"%LAUNCHER_NEW%\" \"%LAUNCHER_OLD%\" >NUL\r\nif errorlevel 1 goto loop\r\ndel \"%LAUNCHER_NEW%\"\r\nstart \"\" \"%LAUNCHER_OLD%\"\r\nendlocal\r\n"
	if err := os.WriteFile(scriptPath, []byte(scriptContents), 0600); err != nil {
		logError("update", "ошибка записи bat-скрипта %s: %v", scriptPath, err)
		return fmt.Errorf("не удалось создать скрипт обновления: %w", err)
	}
	logInfo("update", "bat-скрипт обновления записан: %s", scriptPath)

	cmd := exec.Command("cmd.exe", "/C", scriptPath) // #nosec G204 — управляемые пути
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Env = append(os.Environ(),
		"LAUNCHER_OLD="+currentExe,
		"LAUNCHER_NEW="+newPath,
	)
	if err := cmd.Start(); err != nil {
		logError("update", "ошибка запуска bat-скрипта обновления: %v", err)
		return fmt.Errorf("не удалось запустить процесс обновления: %w", err)
	}
	logInfo("update", "процесс обновления запущен в фоне, текущий процесс завершится вручную")
	return nil
}
