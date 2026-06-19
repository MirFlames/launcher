//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// ApplyLauncherUpdate скачивает новую версию лаунчера и обновляет текущий бинарник на месте.
// Использует sh-скрипт для атомарной замены без потери ярлыков/ссылок на исполняемый файл.
func (a *App) ApplyLauncherUpdate() error {
	logInfo("update", "начало ApplyLauncherUpdate (unix), текущая версия: %s", LauncherVersion)
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
	newPath := filepath.Join(dir, "launcher.new")
	logInfo("update", "unix update: currentExe=%s newPath=%s", currentExe, newPath)

	exePath, err := downloadUpdateBinary(manifest)
	if err != nil {
		logError("update", "ошибка downloadUpdateBinary: %v", err)
		return err
	}
	if err := copyFile(exePath, newPath); err != nil {
		logError("update", "ошибка copyFile(%s → %s): %v", exePath, newPath, err)
		return fmt.Errorf("копирование обновления: %w", err)
	}
	_ = os.Chmod(newPath, 0755)
	logInfo("update", "обновление скопировано в %s", newPath)

	scriptPath := filepath.Join(os.TempDir(), "launcher-update-"+manifest.Version+".sh")
	scriptContents := "#!/bin/sh\nsleep 1\ncp \"$LAUNCHER_NEW\" \"$LAUNCHER_OLD\"\nchmod +x \"$LAUNCHER_OLD\"\nrm -f \"$LAUNCHER_NEW\"\n\"$LAUNCHER_OLD\" &\n"
	if err := os.WriteFile(scriptPath, []byte(scriptContents), 0700); err != nil {
		logError("update", "ошибка записи sh-скрипта %s: %v", scriptPath, err)
		return fmt.Errorf("не удалось создать скрипт обновления: %w", err)
	}
	logInfo("update", "sh-скрипт обновления записан: %s", scriptPath)

	cmd := exec.Command("sh", scriptPath) // #nosec G204 — управляемые пути
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = append(os.Environ(),
		"LAUNCHER_OLD="+currentExe,
		"LAUNCHER_NEW="+newPath,
	)
	if err := cmd.Start(); err != nil {
		logError("update", "ошибка запуска sh-скрипта обновления: %v", err)
		return fmt.Errorf("не удалось запустить процесс обновления: %w", err)
	}
	logInfo("update", "процесс обновления запущен в фоне, текущий процесс завершится вручную")
	return nil
}
