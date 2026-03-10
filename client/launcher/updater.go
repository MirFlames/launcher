package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// LauncherUpdateManifest описывает формат JSON-манифеста обновления,
// лежащего в релизах GitHub (launcher-update.json).
type LauncherUpdateManifest struct {
	Version     string `json:"version"`
	Mandatory   bool   `json:"mandatory"`
	Changelog   string `json:"changelog"`
	DownloadURL string `json:"download_url"`
	Size        int64  `json:"size"`
	SHA256      string `json:"sha256"`
	PublishedAt string `json:"published_at,omitempty"`
}

// LauncherUpdateInfo — урезанная версия манифеста для фронтенда.
type LauncherUpdateInfo struct {
	Version        string `json:"version"`
	Mandatory      bool   `json:"mandatory"`
	Changelog      string `json:"changelog"`
	DownloadURL    string `json:"download_url"`
	Size           int64  `json:"size"`
	SHA256         string `json:"sha256"`
	CurrentVersion string `json:"current_version"`
}

// lastUpdateManifest кэширует последний успешно загруженный и проверенный манифест.
var lastUpdateManifest *LauncherUpdateManifest

// fetchUpdateManifest загружает и проверяет подпись манифеста обновления.
// Если автообновление не сконфигурировано, возвращает (nil, nil).
func fetchUpdateManifest() (*LauncherUpdateManifest, error) {
	manifestURL := strings.TrimSpace(buildUpdateManifestURL)
	sigURL := strings.TrimSpace(buildUpdateSignatureURL)
	pubHex := strings.TrimSpace(buildUpdatePublicKeyHex)
	if manifestURL == "" || sigURL == "" || pubHex == "" {
		// Механизм автообновления не настроен.
		return nil, nil
	}

	if _, err := url.Parse(manifestURL); err != nil {
		return nil, fmt.Errorf("некорректный URL манифеста: %w", err)
	}
	if _, err := url.Parse(sigURL); err != nil {
		return nil, fmt.Errorf("некорректный URL подписи: %w", err)
	}

	// Загружаем JSON манифеста.
	resp, err := getWithRetry(manifestURL, httpTimeoutShort)
	if err != nil {
		return nil, fmt.Errorf("загрузка манифеста: %w", err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("чтение манифеста: %w", err)
	}

	// Загружаем подпись (hex-строка Ed25519(data)).
	sResp, err := getWithRetry(sigURL, httpTimeoutShort)
	if err != nil {
		return nil, fmt.Errorf("загрузка подписи манифеста: %w", err)
	}
	defer sResp.Body.Close()
	sigRaw, err := io.ReadAll(sResp.Body)
	if err != nil {
		return nil, fmt.Errorf("чтение подписи манифеста: %w", err)
	}
	sigStr := strings.TrimSpace(string(sigRaw))
	sigBytes, err := hex.DecodeString(sigStr)
	if err != nil {
		return nil, fmt.Errorf("подпись манифеста: некорректный hex: %w", err)
	}

	pubBytes, err := hex.DecodeString(pubHex)
	if err != nil {
		return nil, fmt.Errorf("публичный ключ: некорректный hex: %w", err)
	}
	if len(pubBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("публичный ключ: ожидается %d байт, получено %d", ed25519.PublicKeySize, len(pubBytes))
	}

	if !ed25519.Verify(ed25519.PublicKey(pubBytes), data, sigBytes) {
		return nil, fmt.Errorf("подпись манифеста недействительна")
	}

	var manifest LauncherUpdateManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("разбор манифеста: %w", err)
	}
	if manifest.Version == "" || manifest.DownloadURL == "" || manifest.SHA256 == "" {
		return nil, fmt.Errorf("манифест обновления неполон")
	}
	lastUpdateManifest = &manifest
	return &manifest, nil
}

// compareVersions сравнивает версии вида "1.0.2".
// Возвращает -1 если a<b, 0 если a==b, 1 если a>b.
func compareVersions(a, b string) int {
	parse := func(s string) []int {
		parts := strings.Split(strings.TrimSpace(s), ".")
		out := make([]int, len(parts))
		for i, p := range parts {
			n := 0
			if p != "" {
				_, _ = fmt.Sscanf(p, "%d", &n)
			}
			out[i] = n
		}
		return out
	}
	pa := parse(a)
	pb := parse(b)
	n := len(pa)
	if len(pb) > n {
		n = len(pb)
	}
	for i := 0; i < n; i++ {
		va, vb := 0, 0
		if i < len(pa) {
			va = pa[i]
		}
		if i < len(pb) {
			vb = pb[i]
		}
		if va < vb {
			return -1
		}
		if va > vb {
			return 1
		}
	}
	return 0
}

// downloadUpdateBinary скачивает бинарник обновления в temp-файл и проверяет SHA256.
func downloadUpdateBinary(manifest *LauncherUpdateManifest) (string, error) {
	if manifest == nil {
		return "", fmt.Errorf("манифест обновления не задан")
	}
	tmpDir := os.TempDir()
	base := "launcher-update-" + strings.ReplaceAll(manifest.Version, " ", "_")
	if runtime.GOOS == "windows" {
		base += ".exe"
	}
	tmpPath := filepath.Join(tmpDir, base)

	if err := downloadFile(manifest.DownloadURL, tmpPath, manifest.Size, nil); err != nil {
		return "", fmt.Errorf("загрузка обновления: %w", err)
	}
	if !verifySHA256(tmpPath, manifest.SHA256) {
		os.Remove(tmpPath)
		return "", fmt.Errorf("контрольная сумма загруженного обновления не совпадает")
	}
	// На *nix помечаем как исполняемый, чтобы можно было запустить напрямую.
	if runtime.GOOS != "windows" {
		_ = os.Chmod(tmpPath, 0755)
	}
	return tmpPath, nil
}

// CheckLauncherUpdate проверяет наличие новой версии лаунчера.
// Возвращает nil, nil если автообновление отключено или обновление не требуется.
func (a *App) CheckLauncherUpdate() (*LauncherUpdateInfo, error) {
	manifest, err := fetchUpdateManifest()
	if err != nil {
		return nil, err
	}
	if manifest == nil {
		// автообновление не сконфигурировано
		return nil, nil
	}
	if compareVersions(manifest.Version, LauncherVersion) <= 0 {
		// обновление не требуется
		return nil, nil
	}
	info := &LauncherUpdateInfo{
		Version:        manifest.Version,
		Mandatory:      manifest.Mandatory,
		Changelog:      manifest.Changelog,
		DownloadURL:    manifest.DownloadURL,
		Size:           manifest.Size,
		SHA256:         manifest.SHA256,
		CurrentVersion: LauncherVersion,
	}
	return info, nil
}

// ApplyLauncherUpdate скачивает новую версию лаунчера и обновляет текущий бинарник.
// На Windows выполняется обновление "на месте", чтобы ярлыки, указывающие на launcher.exe,
// продолжали открывать обновлённую версию.
func (a *App) ApplyLauncherUpdate() error {
	manifest := lastUpdateManifest
	if manifest == nil {
		m, err := fetchUpdateManifest()
		if err != nil {
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

	// Специальная логика для Windows: обновление "на месте" текущего launcher.exe,
	// чтобы ярлыки продолжали работать.
	if runtime.GOOS == "windows" {
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

		// Скачиваем новый бинарник рядом с текущим.
		tmpManifest := *manifest
		tmpManifest.DownloadURL = manifest.DownloadURL
		if err := downloadFile(tmpManifest.DownloadURL, newPath, tmpManifest.Size, nil); err != nil {
			return fmt.Errorf("загрузка обновления: %w", err)
		}
		if !verifySHA256(newPath, manifest.SHA256) {
			_ = os.Remove(newPath)
			return fmt.Errorf("контрольная сумма загруженного обновления не совпадает")
		}

		// Запускаем вспомогательный cmd, который после задержки заменит бинарник и перезапустит лаунчер.
		// Используем небольшую паузу, чтобы успеть корректно завершить текущий процесс.
		script := fmt.Sprintf(
			`ping 127.0.0.1 -n 3 >NUL && copy /Y "%s" "%s" >NUL && del "%s" && start "" "%s"`,
			newPath, currentExe, newPath, currentExe,
		)
		cmd := exec.Command("cmd.exe", "/C", script) // #nosec G204 — управляемые пути
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("не удалось запустить процесс обновления: %w", err)
		}
		// Дальнейшее завершение и рестарт лаунчера обрабатываются внешним процессом.
		return nil
	}

	// Для других платформ оставляем прежнее поведение: запускаем загруженный бинарник.
	path, err := downloadUpdateBinary(manifest)
	if err != nil {
		return err
	}
	cmd := exec.Command(path) // #nosec G204 — путь контролируется подписанным манифестом
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("не удалось запустить установщик обновления: %w", err)
	}
	return nil
}

