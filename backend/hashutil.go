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
)

const defaultBaseURL = "http://62.182.138.124"

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

// updateConfigHashes обновляет хеши в конфигурации на основе реальных файлов
func updateConfigHashes() error {
	// Обновить хеши клиентских файлов
	for i := range config.ClientFiles {
		// Извлечь путь из URL (убрать протокол и хост)
		filePath := config.ClientFiles[i].URL
		if idx := strings.Index(filePath, "/files/"); idx != -1 {
			filePath = filePath[idx+len("/files/"):]
		}
		fullPath := filepath.Join(config.FilesPath, filePath)

		hash, err := calculateFileHash(fullPath)
		if err != nil {
			log.Printf("Предупреждение: не удалось вычислить хеш для %s: %v", fullPath, err)
			continue
		}
		config.ClientFiles[i].Hash = hash
		log.Printf("Обновлен хеш для %s: %s", config.ClientFiles[i].Name, hash)
	}

	// Обновить хеши модов
	for i := range config.Mods {
		// Извлечь путь из URL (убрать протокол и хост)
		filePath := config.Mods[i].URL
		if idx := strings.Index(filePath, "/files/"); idx != -1 {
			filePath = filePath[idx+len("/files/"):]
		}
		fullPath := filepath.Join(config.FilesPath, filePath)

		hash, err := calculateFileHash(fullPath)
		if err != nil {
			log.Printf("Предупреждение: не удалось вычислить хеш для %s: %v", fullPath, err)
			continue
		}
		config.Mods[i].Hash = hash
		log.Printf("Обновлен хеш для %s: %s", config.Mods[i].Name, hash)
	}

	// Сохранить обновленную конфигурацию
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка кодирования конфигурации: %w", err)
	}

	if err := os.WriteFile("config.json", data, 0644); err != nil {
		return fmt.Errorf("ошибка сохранения конфигурации: %w", err)
	}

	log.Println("Конфигурация обновлена успешно")
	return nil
}

// generateConfig генерирует config.json на основе файлов в директориях
func generateConfig(filesPath string, port string, baseURL string) error {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if port == "" {
		port = "80"
	}
	if filesPath == "" {
		filesPath = "./files"
	}

	config := Config{
		FilesPath: filesPath,
		Port:      port,
		ModsHash:  "",
	}

	// Сканировать моды из files/mods
	modsPath := filepath.Join(filesPath, "mods")
	if err := scanMods(modsPath, baseURL, port, &config); err != nil {
		log.Printf("Предупреждение при сканировании модов: %v", err)
	}

	// Сканировать версии из files/versions
	versionsPath := filepath.Join(filesPath, "versions")
	if err := scanVersions(versionsPath, baseURL, port, &config); err != nil {
		log.Printf("Предупреждение при сканировании версий: %v", err)
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
		baseURL = defaultBaseURL
	}
	port := config.Port
	if port == "" {
		port = "80"
	}

	// Очистить и заново отсканировать mods и client_files
	config.Mods = nil
	config.ClientFiles = nil

	modsPath := filepath.Join(config.FilesPath, "mods")
	if err := scanMods(modsPath, baseURL, port, &config); err != nil {
		return fmt.Errorf("ошибка сканирования модов: %w", err)
	}

	versionsPath := filepath.Join(config.FilesPath, "versions")
	if err := scanVersions(versionsPath, baseURL, port, &config); err != nil {
		return fmt.Errorf("ошибка сканирования версий: %w", err)
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
func scanMods(modsPath string, baseURL string, port string, config *Config) error {
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

		// Создать URL
		url := fmt.Sprintf("%s:%s/files/%s", baseURL, port, strings.ReplaceAll(relPath, "\\", "/"))

		// Добавить мод в конфигурацию
		mod := ModFile{
			Name: filepath.Base(path),
			URL:  url,
			Hash: hash,
		}
		config.Mods = append(config.Mods, mod)
		log.Printf("Найден мод: %s", mod.Name)

		return nil
	})

	return err
}

// scanVersions сканирует директорию с версиями майнкрафта
func scanVersions(versionsPath string, baseURL string, port string, config *Config) error {
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

			// Создать URL
			url := fmt.Sprintf("%s:%s/files/%s", baseURL, port, strings.ReplaceAll(relPath, "\\", "/"))

			// Добавить файл клиента в конфигурацию
			clientFile := ClientFile{
				Name: info.Name(),
				URL:  url,
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
