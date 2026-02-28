package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// calculateFileHash вычисляет SHA-256 хеш файла
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("не удалось открыть файл: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("ошибка чтения файла: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// getRelPathFromURL извлекает относительный путь из URL или относительного пути (часть после /files/).
// Поддерживает полные URL (http://host:port/files/mods/X.jar) и относительные пути (/files/mods/X.jar).
func getRelPathFromURL(url string) string {
	if idx := strings.Index(url, "/files/"); idx != -1 {
		return url[idx+len("/files/"):]
	}
	return url
}

// toStoragePath возвращает путь для хранения в config.json: /files/relPath
func toStoragePath(relPath string) string {
	norm := strings.ReplaceAll(relPath, "\\", "/")
	norm = strings.TrimPrefix(norm, "/")
	return "/files/" + norm
}

// updateConfigHashes синхронизирует config.json с файловой системой:
// - Удаляет из config записи о файлах, которых нет на диске
// - Обновляет хеши существующих файлов
// - Добавляет в config новые файлы из папок mods и versions
func updateConfigHashes(quiet bool) error {
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		return fmt.Errorf("BASE_URL обязателен, задайте в .env")
	}
	port := config.Port
	if port == "" {
		return fmt.Errorf("PORT обязателен, задайте в .env")
	}

	// 1. Фильтруем ClientFiles: оставляем только существующие, обновляем хеши
	var newClientFiles []ClientFile
	for _, f := range config.ClientFiles {
		relPath := getRelPathFromURL(f.URL)
		fullPath := filepath.Join(config.FilesPath, filepath.FromSlash(relPath))
		hash, err := calculateFileHash(fullPath)
		if err != nil {
			log.Printf("[Hashes] Удалён из config (файл отсутствует): %s", f.Name)
			continue
		}
		storagePath := toStoragePath(relPath)
		newClientFiles = append(newClientFiles, ClientFile{Name: f.Name, URL: storagePath, Hash: hash})
		if !quiet {
			log.Printf("Обновлен хеш для %s: %s", f.Name, hash)
		}
	}

	// 2. Фильтруем Mods: оставляем только существующие, обновляем хеши
	var newMods []ModFile
	for _, m := range config.Mods {
		relPath := getRelPathFromURL(m.URL)
		fullPath := filepath.Join(config.FilesPath, filepath.FromSlash(relPath))
		hash, err := calculateFileHash(fullPath)
		if err != nil {
			log.Printf("[Hashes] Удалён из config (файл отсутствует): %s", m.Name)
			continue
		}
		storagePath := toStoragePath(relPath)
		newMods = append(newMods, ModFile{Name: m.Name, URL: storagePath, Hash: hash})
		if !quiet {
			log.Printf("Обновлен хеш для %s: %s", m.Name, hash)
		}
	}

	// 3. Собираем множество путей, уже присутствующих в config
	existingModPaths := make(map[string]bool)
	for _, m := range newMods {
		existingModPaths[getRelPathFromURL(m.URL)] = true
	}
	existingClientPaths := make(map[string]bool)
	for _, f := range newClientFiles {
		existingClientPaths[getRelPathFromURL(f.URL)] = true
	}

	// 4. Сканируем mods — добавляем новые файлы
	modsPath := filepath.Join(config.FilesPath, "mods")
	if err := collectNewMods(modsPath, &newMods, existingModPaths, quiet); err != nil && !quiet {
		log.Printf("Предупреждение при сканировании mods: %v", err)
	}

	// 5. Сканируем versions — добавляем новые файлы
	versionsPath := filepath.Join(config.FilesPath, "versions")
	if err := collectNewClientFiles(versionsPath, &newClientFiles, existingClientPaths, quiet); err != nil && !quiet {
		log.Printf("Предупреждение при сканировании versions: %v", err)
	}

	// 6. Сканируем other — добавляем settings-файлы (options.txt и т.п.)
	otherPath := filepath.Join(config.FilesPath, "other")
	if err := collectOtherClientFiles(otherPath, &newClientFiles, existingClientPaths, quiet); err != nil && !quiet {
		log.Printf("Предупреждение при сканировании other: %v", err)
	}

	config.ClientFiles = newClientFiles
	config.Mods = newMods

	// Обновить minecraft_version из client_files
	if len(config.ClientFiles) > 0 {
		for _, file := range config.ClientFiles {
			if idx := strings.Index(file.URL, "/versions/"); idx != -1 {
				pathPart := file.URL[idx+len("/versions/"):]
				if slashIdx := strings.Index(pathPart, "/"); slashIdx != -1 {
					config.MinecraftVersion = pathPart[:slashIdx]
					break
				}
			}
		}
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка кодирования конфигурации: %w", err)
	}
	if err := os.WriteFile("config.json", data, 0644); err != nil {
		return fmt.Errorf("ошибка сохранения конфигурации: %w", err)
	}

	if quiet {
		log.Printf("[Hashes] Синхронизировано: %d модов, %d client_files", len(config.Mods), len(config.ClientFiles))
	} else {
		log.Println("Конфигурация обновлена успешно")
	}
	return nil
}

// collectNewMods добавляет в mods файлы из modsPath, которых нет в existingPaths.
func collectNewMods(modsPath string, mods *[]ModFile, existingPaths map[string]bool, quiet bool) error {
	if _, err := os.Stat(modsPath); os.IsNotExist(err) {
		return nil
	}
	return filepath.Walk(modsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relPath, err := filepath.Rel(config.FilesPath, path)
		if err != nil {
			return nil
		}
		relPathNorm := strings.ReplaceAll(relPath, "\\", "/")
		if existingPaths[relPathNorm] {
			return nil
		}
		hash, err := calculateFileHash(path)
		if err != nil {
			return nil
		}
		storagePath := toStoragePath(relPathNorm)
		*mods = append(*mods, ModFile{Name: filepath.Base(path), URL: storagePath, Hash: hash})
		if !quiet {
			log.Printf("[Hashes] Добавлен в config: %s", filepath.Base(path))
		} else {
			log.Printf("[Hashes] Добавлен: %s", filepath.Base(path))
		}
		return nil
	})
}

// collectNewClientFiles добавляет в clientFiles jar-файлы из versionsPath, которых нет в existingPaths.
func collectNewClientFiles(versionsPath string, clientFiles *[]ClientFile, existingPaths map[string]bool, quiet bool) error {
	if _, err := os.Stat(versionsPath); os.IsNotExist(err) {
		return nil
	}
	entries, err := os.ReadDir(versionsPath)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		versionDir := filepath.Join(versionsPath, entry.Name())
		_ = filepath.Walk(versionDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return err
			}
			if !strings.HasSuffix(strings.ToLower(info.Name()), ".jar") {
				return nil
			}
			relPath, err := filepath.Rel(config.FilesPath, path)
			if err != nil {
				return nil
			}
			relPathNorm := strings.ReplaceAll(relPath, "\\", "/")
			if existingPaths[relPathNorm] {
				return nil
			}
			hash, err := calculateFileHash(path)
			if err != nil {
				return nil
			}
			storagePath := toStoragePath(relPathNorm)
			*clientFiles = append(*clientFiles, ClientFile{Name: info.Name(), URL: storagePath, Hash: hash})
			if !quiet {
				log.Printf("[Hashes] Добавлен в config: %s (версия: %s)", info.Name(), entry.Name())
			} else {
				log.Printf("[Hashes] Добавлен: %s", info.Name())
			}
			return nil
		})
	}
	return nil
}

// collectOtherClientFiles добавляет в clientFiles файлы из otherPath (files/other/), которых нет в existingPaths.
// Добавляются все файлы, не только .jar (options.txt и т.п.).
func collectOtherClientFiles(otherPath string, clientFiles *[]ClientFile, existingPaths map[string]bool, quiet bool) error {
	if _, err := os.Stat(otherPath); os.IsNotExist(err) {
		return nil
	}
	return filepath.Walk(otherPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relPath, err := filepath.Rel(config.FilesPath, path)
		if err != nil {
			return nil
		}
		relPathNorm := strings.ReplaceAll(relPath, "\\", "/")
		if existingPaths[relPathNorm] {
			return nil
		}
		hash, err := calculateFileHash(path)
		if err != nil {
			return nil
		}
		storagePath := toStoragePath(relPathNorm)
		*clientFiles = append(*clientFiles, ClientFile{Name: info.Name(), URL: storagePath, Hash: hash})
		if !quiet {
			log.Printf("[Hashes] Добавлен в config (other): %s", info.Name())
		} else {
			log.Printf("[Hashes] Добавлен: %s", info.Name())
		}
		return nil
	})
}

const hashesCheckInterval = 60 * time.Second

// computeFilesSignature возвращает строку-подпись по путям и времени модификации файлов
// в папках mods, versions и other. Если подпись изменилась — файлы изменились (добавлены, удалены или изменены).
func computeFilesSignature() string {
	var b strings.Builder
	modsPath := filepath.Join(config.FilesPath, "mods")
	_ = filepath.Walk(modsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		fmt.Fprintf(&b, "%s|%d;", path, info.ModTime().UnixNano())
		return nil
	})
	versionsPath := filepath.Join(config.FilesPath, "versions")
	entries, _ := os.ReadDir(versionsPath)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		versionDir := filepath.Join(versionsPath, entry.Name())
		_ = filepath.Walk(versionDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			fmt.Fprintf(&b, "%s|%d;", path, info.ModTime().UnixNano())
			return nil
		})
	}
	otherPath := filepath.Join(config.FilesPath, "other")
	_ = filepath.Walk(otherPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		fmt.Fprintf(&b, "%s|%d;", path, info.ModTime().UnixNano())
		return nil
	})
	return b.String()
}

// startHashesWatcher запускает фоновую горутину, периодически проверяющую изменения файлов.
// Хэши пересчитываются только при изменении модов или client_files.
func startHashesWatcher() {
	go func() {
		ticker := time.NewTicker(hashesCheckInterval)
		defer ticker.Stop()
		lastSig := computeFilesSignature()
		for range ticker.C {
			sig := computeFilesSignature()
			if sig != lastSig {
				lastSig = sig
				if err := updateConfigHashes(true); err != nil {
					log.Printf("[Hashes] Ошибка пересчёта: %v", err)
				}
			}
		}
	}()
	log.Println("[Hashes] Автообновление хэшей включено (при изменении файлов)")
}

// generateConfig генерирует config.json на основе файлов в директориях.
// baseURL, port, filesPath должны быть переданы из env (без дефолтов).
func generateConfig(filesPath string, port string, baseURL string) error {
	if baseURL == "" || port == "" || filesPath == "" {
		return fmt.Errorf("BASE_URL, PORT, FILES_PATH обязательны")
	}

	config := Config{
		FilesPath: filesPath,
		Port:      port,
		ModsHash:  "",
	}

	// Сканировать моды из files/mods
	modsPath := filepath.Join(filesPath, "mods")
	if err := scanMods(modsPath, &config); err != nil {
		log.Printf("Предупреждение при сканировании модов: %v", err)
	}

	// Сканировать версии из files/versions
	versionsPath := filepath.Join(filesPath, "versions")
	if err := scanVersions(versionsPath, &config); err != nil {
		log.Printf("Предупреждение при сканировании версий: %v", err)
	}

	// Сканировать other (options.txt и т.п.)
	otherPath := filepath.Join(filesPath, "other")
	if err := scanOther(otherPath, &config); err != nil {
		log.Printf("Предупреждение при сканировании other: %v", err)
	}

	// Определить версию майнкрафта из найденных версий
	if len(config.ClientFiles) > 0 {
		// Извлечь версию из первого файла (например, versions/1.20.1/client.jar -> 1.20.1)
		for _, file := range config.ClientFiles {
			if idx := strings.Index(file.URL, "/versions/"); idx != -1 {
				pathPart := file.URL[idx+len("/versions/"):]
				if slashIdx := strings.Index(pathPart, "/"); slashIdx != -1 {
					config.MinecraftVersion = pathPart[:slashIdx]
					break
				}
			}
		}
	}

	// Сохранить конфигурацию
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка кодирования конфигурации: %w", err)
	}

	if err := os.WriteFile("config.json", data, 0644); err != nil {
		return fmt.Errorf("ошибка сохранения конфигурации: %w", err)
	}

	log.Println("Конфигурация успешно сгенерирована")
	return nil
}

// rescanConfig пересчитывает mods и client_files из папок, сохраняя остальные настройки config
func rescanConfig() error {
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		return fmt.Errorf("BASE_URL обязателен, задайте в .env")
	}
	port := config.Port
	if port == "" {
		return fmt.Errorf("PORT обязателен, задайте в .env")
	}

	// Очистить и заново отсканировать mods и client_files
	config.Mods = nil
	config.ClientFiles = nil

	modsPath := filepath.Join(config.FilesPath, "mods")
	if err := scanMods(modsPath, &config); err != nil {
		return fmt.Errorf("ошибка сканирования модов: %w", err)
	}

	versionsPath := filepath.Join(config.FilesPath, "versions")
	if err := scanVersions(versionsPath, &config); err != nil {
		return fmt.Errorf("ошибка сканирования версий: %w", err)
	}

	otherPath := filepath.Join(config.FilesPath, "other")
	if err := scanOther(otherPath, &config); err != nil {
		return fmt.Errorf("ошибка сканирования other: %w", err)
	}

	// Обновить minecraft_version из client_files
	if len(config.ClientFiles) > 0 {
		for _, file := range config.ClientFiles {
			if idx := strings.Index(file.URL, "/versions/"); idx != -1 {
				pathPart := file.URL[idx+len("/versions/"):]
				if slashIdx := strings.Index(pathPart, "/"); slashIdx != -1 {
					config.MinecraftVersion = pathPart[:slashIdx]
					break
				}
			}
		}
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка кодирования конфигурации: %w", err)
	}
	if err := os.WriteFile("config.json", data, 0644); err != nil {
		return fmt.Errorf("ошибка сохранения config.json: %w", err)
	}

	log.Printf("Пересчитано: %d модов, %d client_files", len(config.Mods), len(config.ClientFiles))
	return nil
}

// scanMods сканирует директорию с модами
func scanMods(modsPath string, config *Config) error {
	// Проверить существование директории
	if _, err := os.Stat(modsPath); os.IsNotExist(err) {
		log.Printf("Директория модов не найдена: %s", modsPath)
		return nil
	}

	// Сканировать все файлы в директории mods
	err := filepath.Walk(modsPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Пропустить директории
		if info.IsDir() {
			return nil
		}

		// Вычислить относительный путь от filesPath
		relPath, err := filepath.Rel(config.FilesPath, path)
		if err != nil {
			return err
		}

		// Вычислить хеш файла
		hash, err := calculateFileHash(path)
		if err != nil {
			log.Printf("Предупреждение: не удалось вычислить хеш для %s: %v", path, err)
			return nil
		}

		// Путь для хранения в config.json
		relPathNorm := strings.ReplaceAll(relPath, "\\", "/")
		storagePath := toStoragePath(relPathNorm)

		// Добавить мод в конфигурацию
		mod := ModFile{
			Name: filepath.Base(path),
			URL:  storagePath,
			Hash: hash,
		}
		config.Mods = append(config.Mods, mod)
		log.Printf("Найден мод: %s", mod.Name)

		return nil
	})

	return err
}

// scanVersions сканирует директорию с версиями майнкрафта
func scanVersions(versionsPath string, config *Config) error {
	// Проверить существование директории
	if _, err := os.Stat(versionsPath); os.IsNotExist(err) {
		log.Printf("Директория версий не найдена: %s", versionsPath)
		return nil
	}

	// Сканировать поддиректории версий
	entries, err := os.ReadDir(versionsPath)
	if err != nil {
		return fmt.Errorf("ошибка чтения директории версий: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		versionDir := filepath.Join(versionsPath, entry.Name())

		// Сканировать jar файлы в директории версии
		err := filepath.Walk(versionDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Пропустить директории
			if info.IsDir() {
				return nil
			}

			// Проверить, что это jar файл
			if !strings.HasSuffix(strings.ToLower(info.Name()), ".jar") {
				return nil
			}

			// Вычислить относительный путь от filesPath
			relPath, err := filepath.Rel(config.FilesPath, path)
			if err != nil {
				return err
			}

			// Вычислить хеш файла
			hash, err := calculateFileHash(path)
			if err != nil {
				log.Printf("Предупреждение: не удалось вычислить хеш для %s: %v", path, err)
				return nil
			}

			// Путь для хранения в config.json
			relPathNorm := strings.ReplaceAll(relPath, "\\", "/")
			storagePath := toStoragePath(relPathNorm)

			// Добавить файл клиента в конфигурацию
			clientFile := ClientFile{
				Name: info.Name(),
				URL:  storagePath,
				Hash: hash,
			}
			config.ClientFiles = append(config.ClientFiles, clientFile)
			log.Printf("Найден файл версии: %s (версия: %s)", clientFile.Name, entry.Name())

			return nil
		})

		if err != nil {
			log.Printf("Ошибка при сканировании версии %s: %v", entry.Name(), err)
		}
	}

	return nil
}

// scanOther сканирует директорию other (options.txt и другие settings-файлы)
func scanOther(otherPath string, config *Config) error {
	if _, err := os.Stat(otherPath); os.IsNotExist(err) {
		log.Printf("Директория other не найдена: %s", otherPath)
		return nil
	}
	return filepath.Walk(otherPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(config.FilesPath, path)
		if err != nil {
			return err
		}
		hash, err := calculateFileHash(path)
		if err != nil {
			log.Printf("Предупреждение: не удалось вычислить хеш для %s: %v", path, err)
			return nil
		}
		relPathNorm := strings.ReplaceAll(relPath, "\\", "/")
		storagePath := toStoragePath(relPathNorm)
		clientFile := ClientFile{
			Name: info.Name(),
			URL:  storagePath,
			Hash: hash,
		}
		config.ClientFiles = append(config.ClientFiles, clientFile)
		log.Printf("Найден файл other: %s", clientFile.Name)
		return nil
	})
}
