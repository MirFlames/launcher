package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// JDKInfo — информация о JDK из backend API
type JDKInfo struct {
	Version        string `json:"version"`
	RelativePath   string `json:"relative_path"`
	JavaExecutable string `json:"java_executable"`
	Mandatory      bool   `json:"mandatory"`
	DownloadURL    string `json:"download_url,omitempty"` // Альтернативный URL (если Adoptium недоступен)
}

func getJDKExePath(launcherDir string, info *JDKInfo) string {
	relPath := filepath.FromSlash(strings.ReplaceAll(info.RelativePath, "\\", string(filepath.Separator)))
	return filepath.Join(launcherDir, relPath, filepath.FromSlash(strings.ReplaceAll(info.JavaExecutable, "\\", string(filepath.Separator))))
}

func checkJDKExists(info *JDKInfo, launcherDir string) (javaExe string, exists bool) {
	javaExe = getJDKExePath(launcherDir, info)
	_, err := os.Stat(javaExe)
	return javaExe, err == nil
}

func downloadAndExtractJDK(info *JDKInfo, launcherDir string, onProgress func(stage, status string, progress float64)) (javaExe string, err error) {
	relPath := filepath.FromSlash(strings.ReplaceAll(info.RelativePath, "\\", string(filepath.Separator)))
	targetDir := filepath.Join(launcherDir, relPath)
	javaExe = getJDKExePath(launcherDir, info)

	if onProgress != nil {
		onProgress("Подготовка JDK", "Скачивание JDK...", 0)
	}
	zipPath, err := downloadJDK(info.Version, info.DownloadURL, func(p float64) {
		if onProgress != nil {
			onProgress("Подготовка JDK", fmt.Sprintf("Скачивание JDK... %.0f%%", p*100), p*0.5)
		}
	})
	if err != nil {
		return "", fmt.Errorf("скачивание JDK: %w", err)
	}
	defer os.Remove(zipPath)

	if onProgress != nil {
		onProgress("Подготовка JDK", "Установка JDK...", 0.5)
	}
	if err := extractJDK(zipPath, targetDir, func(p float64) {
		if onProgress != nil {
			onProgress("Подготовка JDK", fmt.Sprintf("Установка JDK... %.0f%%", 50+p*50), 0.5+p*0.4)
		}
	}); err != nil {
		return "", fmt.Errorf("установка JDK: %w", err)
	}
	if onProgress != nil {
		onProgress("Подготовка JDK", "JDK установлен", 1)
	}
	return javaExe, nil
}

// EnsureJDK проверяет наличие JDK в папке лаунчера и при необходимости скачивает.
// JAVA_HOME и системный PATH не используются — только папка рядом с exe.
func EnsureJDK(launcherDir string, onProgress func(stage, status string, progress float64)) (javaExe string, err error) {
	info, err := fetchJDKInfo()
	if err != nil {
		return "", fmt.Errorf("получение информации о JDK: %w", err)
	}
	if info == nil {
		return "", fmt.Errorf("не удалось получить информацию о JDK с сервера")
	}

	javaExe, exists := checkJDKExists(info, launcherDir)
	if exists {
		return javaExe, nil
	}
	if !info.Mandatory {
		return "", fmt.Errorf("JDK не найден по пути %s. Установите JDK вручную", javaExe)
	}

	javaExe, err = downloadAndExtractJDK(info, launcherDir, onProgress)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(javaExe); err != nil {
		return "", fmt.Errorf("java.exe не найден после установки: %s", javaExe)
	}
	return javaExe, nil
}

func fetchJDKInfo() (*JDKInfo, error) {
	cfg, err := LoadConfig()
	if err != nil || cfg == nil || cfg.ApiBaseUrl == "" {
		return nil, fmt.Errorf("конфиг не загружен")
	}
	base := strings.TrimSuffix(cfg.ApiBaseUrl, "/")
	url := base + "/api/jdk/info"

	resp, err := getWithRetry(url, httpTimeoutShort)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var info JDKInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	if info.RelativePath == "" || info.JavaExecutable == "" {
		return nil, fmt.Errorf("неполный ответ API")
	}
	return &info, nil
}

func downloadJDK(version, customURL string, onProgress func(float64)) (string, error) {
	var url string
	if customURL != "" {
		url = strings.TrimSpace(customURL)
	} else {
		versionNum := strings.TrimPrefix(strings.TrimSpace(version), "jdk-")
		major := versionNum
		if idx := strings.Index(versionNum, "."); idx > 0 {
			major = versionNum[:idx]
		}

		osName := "windows"
		arch := "x64"
		switch runtime.GOOS {
		case "darwin":
			osName = "mac"
			if runtime.GOARCH == "arm64" {
				arch = "aarch64"
			} else {
				arch = "x64"
			}
		case "linux":
			osName = "linux"
			if runtime.GOARCH == "arm64" || runtime.GOARCH == "arm" {
				arch = "aarch64"
			} else {
				arch = "x64"
			}
		}

		// Adoptium API — бесплатный OpenJDK
		url = fmt.Sprintf("https://api.adoptium.net/v3/binary/latest/%s/ga/%s/%s/jdk/hotspot/normal/eclipse?project=jdk&archive_type=zip",
			major, osName, arch)
	}

	resp, err := getWithRetry(url, httpTimeoutJDK)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	tmpDir, err := os.MkdirTemp("", "launcher-jdk-*")
	if err != nil {
		return "", err
	}
	zipPath := filepath.Join(tmpDir, "jdk.zip")
	total := resp.ContentLength
	if total <= 0 {
		total = 0
	}
	if err := streamToFileWithProgress(resp.Body, zipPath, total, onProgress); err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	return zipPath, nil
}

func extractJDK(zipPath, targetDir string, onProgress func(float64)) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	// Найти корневую папку в архиве (jdk-21.x.x/ или eclipse.jdk-21.x.x/)
	var entries []*zip.File
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		entries = append(entries, f)
	}
	total := len(entries)
	if total == 0 {
		return fmt.Errorf("архив пуст")
	}

	for i, f := range entries {
		name := filepath.FromSlash(f.Name)
		// Убрать корневую папку (первый сегмент пути)
		parts := strings.SplitN(name, string(filepath.Separator), 2)
		if len(parts) < 2 {
			continue
		}
		relPath := parts[1]
		destPath := filepath.Join(targetDir, relPath)

		if f.FileInfo().IsDir() {
			os.MkdirAll(destPath, dirMode)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), dirMode); err != nil {
			return err
		}
		out, err := os.Create(destPath)
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
		if onProgress != nil {
			onProgress(float64(i+1) / float64(total))
		}
	}
	return nil
}
