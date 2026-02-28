package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// JDKInfo — информация о JDK из backend API
type JDKInfo struct {
	Version        string `json:"version"`
	RelativePath   string `json:"relative_path"`
	JavaExecutable string `json:"java_executable"`
	Mandatory      bool   `json:"mandatory"`
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

	relPath := filepath.FromSlash(strings.ReplaceAll(info.RelativePath, "\\", string(filepath.Separator)))
	javaExePath := filepath.Join(launcherDir, relPath, filepath.FromSlash(strings.ReplaceAll(info.JavaExecutable, "\\", string(filepath.Separator))))

	if _, err := os.Stat(javaExePath); err == nil {
		return javaExePath, nil
	}

	if !info.Mandatory {
		return "", fmt.Errorf("JDK не найден по пути %s. Установите JDK вручную", javaExePath)
	}

	if onProgress != nil {
		onProgress("Подготовка JDK", "Скачивание JDK...", 0)
	}

	targetDir := filepath.Join(launcherDir, relPath)
	zipPath, err := downloadJDK(info.Version, func(p float64) {
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

	if _, err := os.Stat(javaExePath); err != nil {
		return "", fmt.Errorf("java.exe не найден после установки: %s", javaExePath)
	}
	return javaExePath, nil
}

func fetchJDKInfo() (*JDKInfo, error) {
	cfg, err := LoadConfig()
	if err != nil || cfg == nil || cfg.ApiBaseUrl == "" {
		return nil, fmt.Errorf("конфиг не загружен")
	}
	base := strings.TrimSuffix(cfg.ApiBaseUrl, "/")
	url := base + "/api/jdk/info"

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var info JDKInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	if info.RelativePath == "" || info.JavaExecutable == "" {
		return nil, fmt.Errorf("неполный ответ API")
	}
	return &info, nil
}

func downloadJDK(version string, onProgress func(float64)) (string, error) {
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
	url := fmt.Sprintf("https://api.adoptium.net/v3/binary/latest/%s/ga/%s/%s/jdk/hotspot/normal/eclipse?project=jdk&archive_type=zip",
		major, osName, arch)

	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	tmpDir, err := os.MkdirTemp("", "launcher-jdk-*")
	if err != nil {
		return "", err
	}
	zipPath := filepath.Join(tmpDir, "jdk.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	defer f.Close()

	total := resp.ContentLength
	var written int64
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			nn, wErr := f.Write(buf[:n])
			written += int64(nn)
			if wErr != nil {
				return "", wErr
			}
			if onProgress != nil && total > 0 {
				onProgress(float64(written) / float64(total))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	if onProgress != nil {
		onProgress(1)
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
			os.MkdirAll(destPath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
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
