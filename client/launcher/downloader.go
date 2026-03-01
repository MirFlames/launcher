package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
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

const (
	assetsBaseURL  = "https://resources.download.minecraft.net/"
	connectTimeout = 30 * time.Second
	readTimeout    = 2 * time.Minute
)

// ServerVersion — ответ GET /api/version
type ServerVersion struct {
	MinecraftVersion string       `json:"minecraft_version"`
	ModsHash         string       `json:"mods_hash"`
	ClientFiles      []ServerFile `json:"client_files"`
	Mods             []ServerFile `json:"mods"`
	ServerHost       string       `json:"server_host"`
	ServerPort       string       `json:"server_port"`
}

// ServerFile — файл из манифеста (мод или client_file)
type ServerFile struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Hash string `json:"hash"`
}

func getCurrentOS() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "osx"
	default:
		return "linux"
	}
}

func libraryApplies(lib *ModpackLibrary, currentOS string) bool {
	if lib == nil || lib.Rules == nil || len(lib.Rules) == 0 {
		return true
	}
	for _, r := range lib.Rules {
		if r.OS == nil || r.OS.Name == "" {
			continue
		}
		osMatch := strings.EqualFold(r.OS.Name, currentOS)
		if strings.EqualFold(r.Action, "allow") && osMatch {
			return true
		}
		if strings.EqualFold(r.Action, "disallow") && osMatch {
			return false
		}
	}
	return false
}

func getLibraryPath(gameDir string, lib *ModpackLibrary) string {
	if lib == nil || lib.Artifact == nil || lib.Artifact.Path == "" {
		return ""
	}
	path := lib.Artifact.Path
	if !strings.HasPrefix(path, "libraries/") {
		path = "libraries/" + path
	}
	return filepath.Join(gameDir, filepath.FromSlash(path))
}

func downloadFile(urlStr, destPath string, expectedSize int64, onProgress func(float64)) error {
	client := &http.Client{Timeout: connectTimeout + readTimeout}
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "MinecraftLauncher/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	total := resp.ContentLength
	if total <= 0 {
		total = expectedSize
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	var written int64
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			nn, wErr := f.Write(buf[:n])
			written += int64(nn)
			if wErr != nil {
				return wErr
			}
			if onProgress != nil && total > 0 {
				onProgress(float64(written) / float64(total))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	if onProgress != nil {
		onProgress(1)
	}
	return nil
}

// EnsureLibraries скачивает отсутствующие библиотеки из modpack
func EnsureLibraries(gameDir string, modpack *ModpackConfig, onProgress func(stage, status string, progress float64)) error {
	currentOS := getCurrentOS()
	var toDownload []*ModpackLibrary
	for i := range modpack.Libraries {
		lib := &modpack.Libraries[i]
		if !libraryApplies(lib, currentOS) {
			continue
		}
		if lib.Artifact == nil || lib.Artifact.URL == "" {
			continue
		}
		dest := getLibraryPath(gameDir, lib)
		if dest == "" {
			continue
		}
		if _, err := os.Stat(dest); err == nil {
			continue
		}
		toDownload = append(toDownload, lib)
	}

	total := len(toDownload)
	for i, lib := range toDownload {
		dest := getLibraryPath(gameDir, lib)
		status := fmt.Sprintf("Библиотека %s (%d/%d)", lib.Name, i+1, total)
		if onProgress != nil {
			onProgress("Загрузка библиотек", status, float64(i)/float64(total))
		}
		if err := downloadFile(lib.Artifact.URL, dest, lib.Artifact.Size, func(p float64) {
			if onProgress != nil {
				onProgress("Загрузка библиотек", status, (float64(i)+p)/float64(total))
			}
		}); err != nil {
			return fmt.Errorf("библиотека %s: %w", lib.Name, err)
		}
	}
	if onProgress != nil && total > 0 {
		onProgress("Загрузка библиотек", "Библиотеки загружены", 1)
	}
	return nil
}

// EnsureAssetIndex скачивает индекс ассетов
func EnsureAssetIndex(gameDir string, modpack *ModpackConfig) error {
	if modpack.AssetIndex == nil || modpack.AssetIndex.URL == "" {
		return nil
	}
	id := modpack.AssetIndex.ID
	if id == "" {
		id = modpack.Assets
	}
	if id == "" {
		id = "29"
	}
	indexPath := filepath.Join(gameDir, "assets", "indexes", id+".json")
	if _, err := os.Stat(indexPath); err == nil {
		return nil
	}
	return downloadFile(modpack.AssetIndex.URL, indexPath, modpack.AssetIndex.Size, nil)
}

// assetIndexObjects — структура для парсинга assets index
type assetIndexObjects map[string]struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

// EnsureAssets скачивает отсутствующие ассеты
func EnsureAssets(gameDir string, modpack *ModpackConfig, onProgress func(stage, status string, progress float64)) error {
	id := modpack.Assets
	if id == "" && modpack.AssetIndex != nil {
		id = modpack.AssetIndex.ID
	}
	if id == "" {
		id = "29"
	}
	indexPath := filepath.Join(gameDir, "assets", "indexes", id+".json")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil // индекс может отсутствовать
	}

	var root struct {
		Objects assetIndexObjects `json:"objects"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil
	}
	if root.Objects == nil {
		return nil
	}

	assetsDir := filepath.Join(gameDir, "assets", "objects")
	var toDownload []struct {
		hash string
		size int64
	}
	for _, obj := range root.Objects {
		if len(obj.Hash) < 2 {
			continue
		}
		dest := filepath.Join(assetsDir, obj.Hash[:2], obj.Hash)
		if _, err := os.Stat(dest); err == nil {
			continue
		}
		toDownload = append(toDownload, struct {
			hash string
			size int64
		}{obj.Hash, obj.Size})
	}

	total := len(toDownload)
	for i, entry := range toDownload {
		url := assetsBaseURL + entry.hash[:2] + "/" + entry.hash
		dest := filepath.Join(assetsDir, entry.hash[:2], entry.hash)
		status := fmt.Sprintf("Ассеты %d/%d", i+1, total)
		if onProgress != nil {
			onProgress("Загрузка ассетов", status, float64(i)/float64(total))
		}
		if err := downloadFile(url, dest, entry.size, func(p float64) {
			if onProgress != nil {
				onProgress("Загрузка ассетов", status, (float64(i)+p)/float64(total))
			}
		}); err != nil {
			os.Remove(dest)
			// не падаем — ассеты не критичны
		}
	}
	if onProgress != nil && total > 0 {
		onProgress("Загрузка ассетов", "Ассеты загружены", 1)
	}
	return nil
}

// FetchServerVersion получает /api/version
func FetchServerVersion() (*ServerVersion, error) {
	cfg, err := LoadConfig()
	if err != nil || cfg == nil || cfg.ApiBaseUrl == "" {
		return nil, fmt.Errorf("конфиг не загружен")
	}
	base := strings.TrimSuffix(cfg.ApiBaseUrl, "/")
	url := base + "/api/version"

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	var v ServerVersion
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, err
	}
	return &v, nil
}

func verifySHA256(path, expectedHex string) bool {
	if expectedHex == "" {
		return true
	}
	expectedHex = strings.TrimPrefix(strings.ToLower(expectedHex), "sha256:")
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	h := sha256.Sum256(data)
	actual := hex.EncodeToString(h[:])
	return actual == expectedHex
}

// EnsureMods скачивает моды из /api/version и удаляет моды, которых нет на сервере.
// Возвращает (downloaded, error): downloaded=true, если был скачан хотя бы один мод.
func EnsureMods(gameDir string, version *ServerVersion, onProgress func(stage, status string, progress float64)) (bool, error) {
	if version == nil {
		return false, nil
	}

	modsDir := filepath.Join(gameDir, "mods")
	if err := os.MkdirAll(modsDir, 0755); err != nil {
		return false, err
	}

	// Моды, которых нет на сервере — удаляем у клиента
	expectedMods := make(map[string]bool)
	for _, m := range version.Mods {
		expectedMods[m.Name] = true
	}
	entries, err := os.ReadDir(modsDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if !expectedMods[e.Name()] {
				path := filepath.Join(modsDir, e.Name())
				os.Remove(path)
			}
		}
	}

	if len(version.Mods) == 0 {
		return false, nil
	}

	var toDownload []ServerFile
	for _, m := range version.Mods {
		if m.URL == "" {
			continue
		}
		dest := filepath.Join(modsDir, m.Name)
		if _, err := os.Stat(dest); err == nil {
			if m.Hash != "" && !verifySHA256(dest, m.Hash) {
				os.Remove(dest)
				toDownload = append(toDownload, m)
			}
			continue
		}
		toDownload = append(toDownload, m)
	}

	total := len(toDownload)
	for i, m := range toDownload {
		dest := filepath.Join(modsDir, m.Name)
		status := fmt.Sprintf("Мод %s (%d/%d)", m.Name, i+1, total)
		if onProgress != nil {
			onProgress("Загрузка модов", status, float64(i)/float64(total))
		}
		if err := downloadFile(m.URL, dest, 0, func(p float64) {
			if onProgress != nil {
				onProgress("Загрузка модов", status, (float64(i)+p)/float64(total))
			}
		}); err != nil {
			return false, fmt.Errorf("мод %s: %w", m.Name, err)
		}
		if m.Hash != "" && !verifySHA256(dest, m.Hash) {
			os.Remove(dest)
			return false, fmt.Errorf("хеш мода не совпадает: %s", m.Name)
		}
	}
	if onProgress != nil && total > 0 {
		onProgress("Загрузка модов", "Моды загружены", 1)
	}
	return len(toDownload) > 0, nil
}

// EnsureClientFiles скачивает settings-файлы (options.txt и т.п.) из version.ClientFiles
// в gameDir. Вызывается только при SyncClientSettings=true и докачке модов.
// Пропускает client_files из versions/ (JAR) — они обрабатываются EnsureClientJar.
func EnsureClientFiles(gameDir string, version *ServerVersion, onProgress func(stage, status string, progress float64)) error {
	if version == nil || len(version.ClientFiles) == 0 {
		return nil
	}

	var toDownload []ServerFile
	for _, f := range version.ClientFiles {
		if f.URL == "" {
			continue
		}
		if strings.Contains(f.URL, "/versions/") {
			continue
		}
		dest := filepath.Join(gameDir, f.Name)
		if _, err := os.Stat(dest); err == nil {
			if f.Hash != "" && !verifySHA256(dest, f.Hash) {
				os.Remove(dest)
				toDownload = append(toDownload, f)
			}
			continue
		}
		toDownload = append(toDownload, f)
	}

	total := len(toDownload)
	for i, f := range toDownload {
		dest := filepath.Join(gameDir, f.Name)
		status := fmt.Sprintf("Настройки %s (%d/%d)", f.Name, i+1, total)
		if onProgress != nil {
			onProgress("Загрузка настроек", status, float64(i)/float64(total))
		}
		if err := downloadFile(f.URL, dest, 0, func(p float64) {
			if onProgress != nil {
				onProgress("Загрузка настроек", status, (float64(i)+p)/float64(total))
			}
		}); err != nil {
			return fmt.Errorf("client_file %s: %w", f.Name, err)
		}
		if f.Hash != "" && !verifySHA256(dest, f.Hash) {
			os.Remove(dest)
			return fmt.Errorf("хеш файла не совпадает: %s", f.Name)
		}
	}
	if onProgress != nil && total > 0 {
		onProgress("Загрузка настроек", "Настройки загружены", 1)
	}
	return nil
}

// EnsureClientJar скачивает client.jar из modpack
func EnsureClientJar(gameDir string, modpack *ModpackConfig, onProgress func(stage, status string, progress float64)) error {
	if modpack.Downloads == nil || modpack.Downloads.Client == nil || modpack.Downloads.Client.URL == "" {
		return nil
	}

	versionID := modpack.ID
	if versionID == "" {
		versionID = "modpack"
	}
	clientJar := filepath.Join(gameDir, "versions", versionID, versionID+".jar")
	if _, err := os.Stat(clientJar); err == nil {
		return nil
	}
	altJar := filepath.Join(gameDir, "versions", versionID, "client.jar")
	if _, err := os.Stat(altJar); err == nil {
		return nil
	}

	if onProgress != nil {
		onProgress("Загрузка client.jar", "Скачивание client.jar...", 0)
	}
	if err := downloadFile(modpack.Downloads.Client.URL, clientJar, modpack.Downloads.Client.Size, func(p float64) {
		if onProgress != nil {
			onProgress("Загрузка client.jar", fmt.Sprintf("Скачивание client.jar... %.0f%%", p*100), p)
		}
	}); err != nil {
		return fmt.Errorf("client.jar: %w", err)
	}
	if onProgress != nil {
		onProgress("Загрузка client.jar", "client.jar загружен", 1)
	}
	return nil
}

func isNativeLibrary(lib *ModpackLibrary) bool {
	return lib != nil && strings.Contains(lib.Name, ":natives-")
}

// ExtractNatives извлекает нативные библиотеки из JAR в папку natives
func ExtractNatives(gameDir string, modpack *ModpackConfig) error {
	currentOS := getCurrentOS()
	nativesDir := filepath.Join(gameDir, "natives")
	if err := os.MkdirAll(nativesDir, 0755); err != nil {
		return err
	}

	for i := range modpack.Libraries {
		lib := &modpack.Libraries[i]
		if !libraryApplies(lib, currentOS) || !isNativeLibrary(lib) {
			continue
		}
		jarPath := getLibraryPath(gameDir, lib)
		if jarPath == "" {
			continue
		}
		if _, err := os.Stat(jarPath); err != nil {
			continue
		}
		if err := extractNativesFromJar(jarPath, nativesDir); err != nil {
			// не критично
		}
	}
	return nil
}

func extractNativesFromJar(jarPath, targetDir string) error {
	r, err := zip.OpenReader(jarPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		name := strings.ToLower(filepath.Base(f.Name))
		if !strings.HasSuffix(name, ".dll") && !strings.HasSuffix(name, ".so") &&
			!strings.HasSuffix(name, ".dylib") && !strings.HasSuffix(name, ".jnilib") {
			continue
		}
		dest := filepath.Join(targetDir, filepath.Base(f.Name))
		rc, err := f.Open()
		if err != nil {
			continue
		}
		out, err := os.Create(dest)
		if err != nil {
			rc.Close()
			continue
		}
		io.Copy(out, rc)
		rc.Close()
		out.Close()
	}
	return nil
}
