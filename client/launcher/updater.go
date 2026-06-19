package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// LauncherUpdateManifest описывает формат JSON-манифеста обновления,
// лежащего в релизах GitHub (launcher-update.json).
type LauncherUpdateManifest struct {
	Version             string `json:"version"`
	Mandatory           bool   `json:"mandatory"`
	MinMandatoryVersion string `json:"min_mandatory_version,omitempty"`
	Changelog           string `json:"changelog"`
	DownloadURL         string `json:"download_url"`
	Size                int64  `json:"size"`
	SHA256              string `json:"sha256"`
	PublishedAt         string `json:"published_at,omitempty"`
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
	logInfo("update", "получен манифест обновления: version=%s mandatory=%v url=%s size=%d", manifest.Version, manifest.Mandatory, manifest.DownloadURL, manifest.Size)
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

// downloadUpdateBinary скачивает обновление (exe или zip) в temp-файл, проверяет SHA256
// и при необходимости распаковывает zip до исполняемого файла. Возвращает путь к exe.
func downloadUpdateBinary(manifest *LauncherUpdateManifest) (string, error) {
	if manifest == nil {
		return "", fmt.Errorf("манифест обновления не задан")
	}
	tmpDir := os.TempDir()
	base := "launcher-update-" + strings.ReplaceAll(manifest.Version, " ", "_")
	downloadPath := filepath.Join(tmpDir, base)

	// Добавляем расширение для удобства (если есть в URL).
	if u, err := url.Parse(manifest.DownloadURL); err == nil {
		if ext := filepath.Ext(u.Path); ext != "" {
			downloadPath += ext
		}
	}

	if err := downloadFile(manifest.DownloadURL, downloadPath, manifest.Size, nil); err != nil {
		return "", fmt.Errorf("загрузка обновления: %w", err)
	}
	if !verifySHA256(downloadPath, manifest.SHA256) {
		_ = os.Remove(downloadPath)
		return "", fmt.Errorf("контрольная сумма загруженного обновления не совпадает")
	}

	ext := strings.ToLower(filepath.Ext(downloadPath))

	// ZIP-архив (Windows и macOS используют zip).
	if ext == ".zip" {
		r, err := zip.OpenReader(downloadPath)
		if err != nil {
			return "", fmt.Errorf("распаковка обновления: %w", err)
		}
		defer r.Close()

		// На Windows ищем .exe, на других платформах — первый файл без расширения или первый файл вообще.
		var target *zip.File
		for _, f := range r.File {
			if f.FileInfo().IsDir() {
				continue
			}
			name := strings.ToLower(f.Name)
			if runtime.GOOS == "windows" {
				if strings.HasSuffix(name, ".exe") {
					target = f
					break
				}
			} else {
				// Берём бинарник (файл без .exe расширения), или первый попавшийся файл как запасной вариант.
				if !strings.HasSuffix(name, ".exe") && !strings.HasSuffix(name, ".json") && !strings.HasSuffix(name, ".sig") {
					target = f
					break
				}
			}
		}
		if target == nil && len(r.File) > 0 {
			// Запасной вариант: берём первый файл в архиве.
			for _, f := range r.File {
				if !f.FileInfo().IsDir() {
					target = f
					break
				}
			}
		}
		if target == nil {
			return "", fmt.Errorf("в архиве обновления не найден исполняемый файл")
		}

		rc, err := target.Open()
		if err != nil {
			return "", fmt.Errorf("чтение бинарника из архива: %w", err)
		}
		defer rc.Close()

		binExt := ""
		if runtime.GOOS == "windows" {
			binExt = ".exe"
		}
		exePath := filepath.Join(tmpDir, base+binExt)
		out, err := os.Create(exePath)
		if err != nil {
			return "", fmt.Errorf("создание временного бинарника: %w", err)
		}
		if _, err := io.Copy(out, rc); err != nil {
			out.Close()
			return "", fmt.Errorf("распаковка бинарника: %w", err)
		}
		out.Close()
		_ = os.Remove(downloadPath)
		if runtime.GOOS != "windows" {
			_ = os.Chmod(exePath, 0755)
		}
		logInfo("update", "обновление распаковано из zip: %s → %s", downloadPath, exePath)
		return exePath, nil
	}

	// TAR.GZ-архив (Linux).
	if strings.HasSuffix(strings.ToLower(downloadPath), ".tar.gz") || ext == ".gz" {
		f, err := os.Open(downloadPath)
		if err != nil {
			return "", fmt.Errorf("открытие tar.gz: %w", err)
		}
		defer f.Close()

		gz, err := gzip.NewReader(f)
		if err != nil {
			return "", fmt.Errorf("gzip: %w", err)
		}
		defer gz.Close()

		tr := tar.NewReader(gz)
		var binPath string
		for {
			hdr, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", fmt.Errorf("чтение tar: %w", err)
			}
			if hdr.Typeflag == tar.TypeReg && !strings.HasSuffix(hdr.Name, "/") {
				binPath = filepath.Join(tmpDir, base)
				out, err := os.Create(binPath)
				if err != nil {
					return "", fmt.Errorf("создание бинарника из tar: %w", err)
				}
				if _, err := io.Copy(out, tr); err != nil { // #nosec G110 — trusted signed manifest
					out.Close()
					return "", fmt.Errorf("распаковка бинарника из tar: %w", err)
				}
				out.Close()
				break
			}
		}
		_ = os.Remove(downloadPath)
		if binPath == "" {
			return "", fmt.Errorf("в tar.gz архиве не найден исполняемый файл")
		}
		_ = os.Chmod(binPath, 0755)
		logInfo("update", "обновление распаковано из tar.gz: %s → %s", downloadPath, binPath)
		return binPath, nil
	}

	// Не архив — считаем, что это готовый бинарник.
	if runtime.GOOS != "windows" {
		_ = os.Chmod(downloadPath, 0755)
	}
	logInfo("update", "обновление скачано как бинарник: %s", downloadPath)
	return downloadPath, nil
}

// CheckLauncherUpdate проверяет наличие новой версии лаунчера.
// Возвращает nil, nil если автообновление отключено или обновление не требуется.
func (a *App) CheckLauncherUpdate() (*LauncherUpdateInfo, error) {
	manifest, err := fetchUpdateManifest()
	if err != nil {
		logError("update", "ошибка получения манифеста: %v", err)
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
	// Если существует минимальная обязательная версия, и текущая ниже её — обновление считаем обязательным,
	// даже если конкретный релиз помечен как опциональный.
	effectiveMandatory := manifest.Mandatory
	if manifest.MinMandatoryVersion != "" && compareVersions(LauncherVersion, manifest.MinMandatoryVersion) < 0 {
		effectiveMandatory = true
	}
	info := &LauncherUpdateInfo{
		Version:        manifest.Version,
		Mandatory:      effectiveMandatory,
		Changelog:      manifest.Changelog,
		DownloadURL:    manifest.DownloadURL,
		Size:           manifest.Size,
		SHA256:         manifest.SHA256,
		CurrentVersion: LauncherVersion,
	}
	logInfo("update", "доступно обновление: current=%s latest=%s mandatory=%v", LauncherVersion, info.Version, info.Mandatory)
	return info, nil
}


// copyFile копирует файл src в dst, перезаписывая, если существует.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), dirMode); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
