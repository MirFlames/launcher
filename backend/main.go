package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var config Config

// responseRecorder оборачивает ResponseWriter для перехвата статус-кода ответа
type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// log404Middleware логирует запросы, завершившиеся с кодом 404
func log404Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		if rec.status == http.StatusNotFound {
			log.Printf("404 Not Found: %s %s", r.Method, r.URL.Path)
		}
	})
}

// corsHandler добавляет CORS-заголовки для работы Swagger UI и других браузерных клиентов
type corsHandler struct{}

func (c corsHandler) wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func main() {
	// Проверить существование config.json, если нет - сгенерировать
	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
		log.Println("config.json не найден, генерирую автоматически...")

		// Получить базовый URL из переменной окружения или использовать localhost
		baseURL := os.Getenv("BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost"
		}

		// Получить порт из переменной окружения или использовать по умолчанию
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}

		// Получить путь к файлам из переменной окружения или использовать по умолчанию
		filesPath := os.Getenv("FILES_PATH")
		if filesPath == "" {
			filesPath = "./files"
		}

		if err := generateConfig(filesPath, port, baseURL); err != nil {
			log.Fatalf("Ошибка генерации конфигурации: %v", err)
		}
	}

	// Загрузить конфигурацию
	if err := loadConfig("config.json"); err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Настроить маршруты (cors разрешает запросы из Swagger UI и других клиентов)
	cors := corsHandler{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/version", cors.wrap(handleVersion))
	mux.HandleFunc("/api/launcher/version", cors.wrap(handleLauncherVersion))
	mux.HandleFunc("/api/jdk/info", cors.wrap(handleJDKInfo))
	mux.HandleFunc("/files/", cors.wrap(handleFileDownload))

	port := config.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("Сервер запущен на порту %s", port)
	log.Printf("Файлы раздаются из: %s", config.FilesPath)
	log.Fatal(http.ListenAndServe(":"+port, log404Middleware(mux)))
}

// loadConfig загружает конфигурацию из JSON файла
func loadConfig(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("не удалось прочитать файл конфигурации: %w", err)
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("не удалось распарсить конфигурацию: %w", err)
	}

	// Валидация версии Minecraft
	if err := validateMinecraftVersion(config.MinecraftVersion); err != nil {
		return fmt.Errorf("невалидная версия Minecraft: %w", err)
	}

	// Валидация путей файлов
	for i, file := range config.ClientFiles {
		if err := validateFileName(file.Name); err != nil {
			return fmt.Errorf("невалидное имя файла клиента [%d]: %w", i, err)
		}
	}
	for i, mod := range config.Mods {
		if err := validateFileName(mod.Name); err != nil {
			return fmt.Errorf("невалидное имя мода [%d]: %w", i, err)
		}
	}

	// Установить значения по умолчанию
	if config.FilesPath == "" {
		config.FilesPath = "./files"
	}
	if config.Port == "" {
		config.Port = "8080"
	}
	if config.LauncherVersion == "" {
		config.LauncherVersion = "1.0.0"
	}
	if config.JDK.Version == "" {
		config.JDK.Version = "jdk-21.0.2"
	}
	if config.JDK.RelativePath == "" {
		config.JDK.RelativePath = "jre_default\\jdk-21.0.2"
	}
	if config.JDK.JavaExecutable == "" {
		config.JDK.JavaExecutable = "bin\\java.exe"
	}

	return nil
}

// validateMinecraftVersion проверяет версию на запрещенные символы Windows
func validateMinecraftVersion(version string) error {
	if version == "" {
		return fmt.Errorf("версия не может быть пустой")
	}

	forbidden := []rune{'<', '>', ':', '"', '/', '\\', '|', '?', '*'}
	for _, char := range forbidden {
		if strings.ContainsRune(version, char) {
			return fmt.Errorf("версия содержит запрещенный символ: %c", char)
		}
	}
	return nil
}

// validateFileName проверяет имя файла на безопасность
func validateFileName(name string) error {
	if name == "" {
		return fmt.Errorf("имя файла не может быть пустым")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("имя файла содержит недопустимую последовательность '..'")
	}
	if filepath.IsAbs(name) {
		return fmt.Errorf("имя файла не может быть абсолютным путем")
	}
	return nil
}

// handleVersion обрабатывает запрос GET /api/version
func handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("Запрошена информация о версии игры.")

	serverInfo := ServerInfo{
		MinecraftVersion: config.MinecraftVersion,
		ModsHash:         config.ModsHash,
		ClientFiles:      config.ClientFiles,
		Mods:             config.Mods,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(serverInfo); err != nil {
		log.Printf("Ошибка кодирования JSON: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
}

// handleFileDownload обрабатывает запросы на скачивание файлов
func handleFileDownload(w http.ResponseWriter, r *http.Request) {
	log.Printf("Запрошен файл: %s", r.URL.Path)
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	// Извлечь путь файла из URL
	filePath := strings.TrimPrefix(r.URL.Path, "/files/")
	if filePath == "" {
		http.Error(w, "Файл не указан", http.StatusBadRequest)
		return
	}

	// Проверить на path traversal
	if strings.Contains(filePath, "..") {
		http.Error(w, "Недопустимый путь", http.StatusBadRequest)
		return
	}

	// Полный путь к файлу
	fullPath := filepath.Join(config.FilesPath, filePath)

	// Дополнительная проверка безопасности: убедиться, что итоговый путь находится внутри FilesPath
	absFilesPath, err := filepath.Abs(config.FilesPath)
	if err != nil {
		log.Printf("Ошибка получения абсолютного пути: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		log.Printf("Ошибка получения абсолютного пути: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
	if !strings.HasPrefix(absFullPath, absFilesPath) {
		http.Error(w, "Недопустимый путь", http.StatusBadRequest)
		return
	}

	// Проверить существование файла
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Файл не найден", http.StatusNotFound)
			return
		}
		log.Printf("Ошибка доступа к файлу: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	// Проверить, что это файл, а не директория
	if fileInfo.IsDir() {
		http.Error(w, "Это директория, а не файл", http.StatusBadRequest)
		return
	}

	// Открыть файл
	file, err := os.Open(fullPath)
	if err != nil {
		log.Printf("Ошибка открытия файла: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Установить заголовки
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))
	w.Header().Set("Accept-Ranges", "bytes")

	// Скопировать файл в ответ
	if _, err := io.Copy(w, file); err != nil {
		log.Printf("Ошибка отправки файла: %v", err)
		return
	}

	log.Printf("Файл %s отправлен.", filePath)
}

// handleLauncherVersion обрабатывает запрос GET /api/launcher/version
func handleLauncherVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("Запрошена информация о версии лаунчера.")

	launcherVersion := LauncherVersion{
		Version:      config.LauncherVersion,
		DownloadURL:  config.LauncherDownloadURL,
		Hash:         config.LauncherHash,
		Size:         config.LauncherSize,
		ReleaseNotes: "",
		Mandatory:    config.LauncherMandatory,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(launcherVersion); err != nil {
		log.Printf("Ошибка кодирования JSON: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
}

// handleJDKInfo обрабатывает запрос GET /api/jdk/info
func handleJDKInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("Запрошена информация о JDK.")

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(config.JDK); err != nil {
		log.Printf("Ошибка кодирования JSON: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
}
