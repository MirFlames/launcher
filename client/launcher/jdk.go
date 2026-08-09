package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
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

// normalizeJDKInfo приводит ответ API к текущей ОС.
// Бэкенд отдаёт всем клиентам одинаковый Windows-ориентированный JSON
// ("java_executable": "bin\\java.exe"), поэтому нормализуем на стороне лаунчера.
func normalizeJDKInfo(info *JDKInfo) {
	if info == nil || runtime.GOOS == "windows" {
		return
	}
	exe := strings.TrimSpace(info.JavaExecutable)
	if strings.EqualFold(filepath.Ext(exe), ".exe") {
		info.JavaExecutable = exe[:len(exe)-len(".exe")]
	}
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
	archivePath, err := downloadJDK(info.Version, info.DownloadURL, func(p float64) {
		if onProgress != nil {
			onProgress("Подготовка JDK", fmt.Sprintf("Скачивание JDK... %.0f%%", p*100), p*0.5)
		}
	})
	if err != nil {
		return "", fmt.Errorf("скачивание JDK: %w", err)
	}
	defer os.Remove(archivePath)

	if onProgress != nil {
		onProgress("Подготовка JDK", "Установка JDK...", 0.5)
	}

	// Распаковываем в соседнюю временную папку и подменяем целевую только целиком:
	// прерванная установка не должна оставить полу-JDK, который потом сойдёт за готовый.
	stageDir := fmt.Sprintf("%s.tmp-%d", targetDir, os.Getpid())
	_ = os.RemoveAll(stageDir)
	defer os.RemoveAll(stageDir)
	if err := os.MkdirAll(stageDir, dirMode); err != nil {
		return "", fmt.Errorf("установка JDK: %w", err)
	}

	if err := extractJDKArchive(archivePath, stageDir, func(p float64) {
		if onProgress != nil {
			onProgress("Подготовка JDK", fmt.Sprintf("Установка JDK... %.0f%%", 50+p*50), 0.5+p*0.4)
		}
	}); err != nil {
		return "", fmt.Errorf("установка JDK: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(targetDir), dirMode); err != nil {
		return "", fmt.Errorf("установка JDK: %w", err)
	}
	_ = os.RemoveAll(targetDir)
	if err := os.Rename(stageDir, targetDir); err != nil {
		return "", fmt.Errorf("перенос JDK в %s: %w", targetDir, err)
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
	normalizeJDKInfo(info)

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
		return "", fmt.Errorf("исполняемый файл Java не найден после установки: %s", javaExe)
	}
	// Страховка: если архив не донёс биты доступа, java всё равно должна быть исполняемой.
	if runtime.GOOS != "windows" {
		_ = os.Chmod(javaExe, 0755)
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
	// Adoptium публикует zip только под Windows; под macOS и Linux — tar.gz
	// (для macOS доступны лишь .pkg и .tar.gz, zip не существует).
	archiveType := "tar.gz"
	if runtime.GOOS == "windows" {
		archiveType = "zip"
	}

	var url string
	// download_url с бэкенда по построению указывает на сборку под Windows
	// (API отдаёт один статичный ответ всем клиентам), поэтому на других ОС его игнорируем.
	if customURL != "" && runtime.GOOS == "windows" {
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
		url = fmt.Sprintf("https://api.adoptium.net/v3/binary/latest/%s/ga/%s/%s/jdk/hotspot/normal/eclipse?project=jdk&archive_type=%s",
			major, osName, arch, archiveType)
	}

	logInfo("jdk", "скачивание JDK: os=%s arch=%s archive_type=%s url=%s", runtime.GOOS, runtime.GOARCH, archiveType, url)

	resp, err := getWithRetry(url, httpTimeoutJDK)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	tmpDir, err := os.MkdirTemp("", "launcher-jdk-*")
	if err != nil {
		return "", err
	}
	archivePath := filepath.Join(tmpDir, "jdk."+archiveType)
	total := resp.ContentLength
	if total <= 0 {
		total = 0
	}
	if err := streamToFileWithProgress(resp.Body, archivePath, total, onProgress); err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	return archivePath, nil
}

// extractJDKArchive распаковывает архив JDK, определяя формат по сигнатуре первых байт,
// а не по расширению: Adoptium умеет отдать tar.gz на запрос zip, и полагаться на имя нельзя.
func extractJDKArchive(archivePath, targetDir string, onProgress func(float64)) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	head := make([]byte, 4)
	n, _ := io.ReadFull(f, head)
	f.Close()
	head = head[:n]

	switch {
	case bytes.HasPrefix(head, []byte("PK\x03\x04")):
		return extractJDKZip(archivePath, targetDir, onProgress)
	case bytes.HasPrefix(head, []byte{0x1f, 0x8b}):
		return extractJDKTarGz(archivePath, targetDir, onProgress)
	default:
		var size int64
		if st, statErr := os.Stat(archivePath); statErr == nil {
			size = st.Size()
		}
		return fmt.Errorf("неизвестный формат архива JDK: ожидался zip или tar.gz, получено %d байт с сигнатурой % x", size, head)
	}
}

// normalizeArchiveName приводит имя записи архива к слэшам и убирает "./" и ведущий слэш.
func normalizeArchiveName(name string) string {
	n := strings.ReplaceAll(name, "\\", "/")
	n = path.Clean(n)
	if n == "." || n == "/" {
		return ""
	}
	return strings.TrimPrefix(n, "/")
}

// jdkStripPrefix определяет префикс, срезаемый со всех записей архива.
// Обычно это единственный корневой каталог (jdk-21.0.5+11/), но у сборок macOS
// внутри лежит бандл — тогда берём только JAVA_HOME (jdk-21.0.5+11/Contents/Home/),
// а служебные Contents/Info.plist и Contents/MacOS отбрасываем.
// Итоговая раскладка на всех ОС одинаковая: <targetDir>/bin/java[.exe].
func jdkStripPrefix(names []string) string {
	root := ""
	for _, raw := range names {
		// Запись самого корневого каталога ("jdk-21/") — нормальная часть архива
		// и, в отличие от файла в корне, вычислению префикса не мешает.
		isDir := strings.HasSuffix(raw, "/") || strings.HasSuffix(raw, `\`)
		n := normalizeArchiveName(raw)
		if n == "" {
			continue
		}
		first := n
		if idx := strings.Index(n, "/"); idx > 0 {
			first = n[:idx]
		} else if !isDir {
			// Есть файл в корне архива — общего каталога нет, ничего не срезаем.
			return ""
		}
		if root == "" {
			root = first
		} else if first != root {
			return ""
		}
	}
	if root == "" {
		return ""
	}
	prefix := root + "/"
	if home := prefix + "Contents/Home/"; hasPrefixEntry(names, home) {
		return home
	}
	return prefix
}

func hasPrefixEntry(names []string, prefix string) bool {
	for _, raw := range names {
		if strings.HasPrefix(normalizeArchiveName(raw), prefix) {
			return true
		}
	}
	return false
}

// relArchivePath срезает общий префикс. Возвращает "" для записей вне префикса —
// они пропускаются (служебные файлы бандла macOS).
func relArchivePath(name, prefix string) string {
	n := normalizeArchiveName(name)
	if n == "" {
		return ""
	}
	if prefix != "" {
		if !strings.HasPrefix(n, prefix) {
			return ""
		}
		n = n[len(prefix):]
	}
	return strings.Trim(n, "/")
}

// jdkDestPath собирает путь назначения и защищает от выхода за пределы targetDir (zip-slip).
func jdkDestPath(targetDir, relPath string) (string, error) {
	destPath := filepath.Join(targetDir, filepath.FromSlash(relPath))
	base := filepath.Clean(targetDir) + string(filepath.Separator)
	if !strings.HasPrefix(filepath.Clean(destPath)+string(filepath.Separator), base) {
		return "", fmt.Errorf("запись архива ведёт за пределы папки установки: %s", relPath)
	}
	return destPath, nil
}

// jdkWriteFile записывает обычный файл, сохраняя биты доступа.
// Без exec-бита распакованная java на macOS и Linux просто не запустится.
func jdkWriteFile(targetDir, relPath string, mode os.FileMode, src io.Reader) error {
	destPath, err := jdkDestPath(targetDir, relPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destPath), dirMode); err != nil {
		return err
	}
	perm := mode.Perm()
	if perm == 0 {
		perm = 0644
	}
	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, src); err != nil { // #nosec G110 — архив с api.adoptium.net
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	// OpenFile учитывает umask, поэтому режим выставляем явно.
	return os.Chmod(destPath, perm)
}

// jdkWriteSymlink создаёт символьную ссылку. На Windows это требует привилегий,
// поэтому там неудача не считается фатальной — JDK под Windows симлинков не содержит.
func jdkWriteSymlink(targetDir, relPath, linkname string) error {
	destPath, err := jdkDestPath(targetDir, relPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destPath), dirMode); err != nil {
		return err
	}
	_ = os.Remove(destPath)
	if err := os.Symlink(linkname, destPath); err != nil {
		if runtime.GOOS == "windows" {
			logInfo("jdk", "симлинк пропущен (%s → %s): %v", relPath, linkname, err)
			return nil
		}
		return err
	}
	return nil
}

func extractJDKZip(archivePath, targetDir string, onProgress func(float64)) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()

	names := make([]string, 0, len(r.File))
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	prefix := jdkStripPrefix(names)

	var entries []*zip.File
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if relArchivePath(f.Name, prefix) == "" {
			continue
		}
		entries = append(entries, f)
	}
	total := len(entries)
	if total == 0 {
		return fmt.Errorf("архив пуст")
	}

	for i, f := range entries {
		relPath := relArchivePath(f.Name, prefix)
		mode := f.Mode()

		rc, err := f.Open()
		if err != nil {
			return err
		}
		if mode&os.ModeSymlink != 0 {
			link, readErr := io.ReadAll(io.LimitReader(rc, 4096))
			rc.Close()
			if readErr != nil {
				return readErr
			}
			if err := jdkWriteSymlink(targetDir, relPath, string(link)); err != nil {
				return err
			}
		} else {
			err = jdkWriteFile(targetDir, relPath, mode, rc)
			rc.Close()
			if err != nil {
				return err
			}
		}
		if onProgress != nil {
			onProgress(float64(i+1) / float64(total))
		}
	}
	return nil
}

// tarGzNames читает только заголовки архива — нужен отдельный проход,
// чтобы вычислить срезаемый корень до начала распаковки (tar не даёт произвольного доступа).
func tarGzNames(archivePath string) ([]string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()

	var names []string
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		names = append(names, hdr.Name)
	}
	return names, nil
}

func extractJDKTarGz(archivePath, targetDir string, onProgress func(float64)) error {
	names, err := tarGzNames(archivePath)
	if err != nil {
		return err
	}
	prefix := jdkStripPrefix(names)

	total := 0
	for _, n := range names {
		if relArchivePath(n, prefix) != "" {
			total++
		}
	}
	if total == 0 {
		return fmt.Errorf("архив пуст")
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	done := 0
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		relPath := relArchivePath(hdr.Name, prefix)
		if relPath == "" {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			destPath, err := jdkDestPath(targetDir, relPath)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(destPath, dirMode); err != nil {
				return err
			}
		case tar.TypeSymlink:
			// Цель симлинка задана относительно его собственной папки — переносим как есть.
			if err := jdkWriteSymlink(targetDir, relPath, hdr.Linkname); err != nil {
				return err
			}
		case tar.TypeLink:
			// Жёсткая ссылка: цель задана относительно корня архива.
			linkRel := relArchivePath(hdr.Linkname, prefix)
			if linkRel == "" {
				continue
			}
			destPath, err := jdkDestPath(targetDir, relPath)
			if err != nil {
				return err
			}
			srcPath, err := jdkDestPath(targetDir, linkRel)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(destPath), dirMode); err != nil {
				return err
			}
			_ = os.Remove(destPath)
			if err := os.Link(srcPath, destPath); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := jdkWriteFile(targetDir, relPath, hdr.FileInfo().Mode(), tr); err != nil {
				return err
			}
		default:
			continue
		}

		done++
		if onProgress != nil {
			onProgress(float64(done) / float64(total))
		}
	}
	return nil
}
