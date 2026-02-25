package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var config Config

// authSession — сессия аутентификации (pending или completed)
type authSession struct {
	code        string
	nickname    string
	sessionUUID string
	createdAt   time.Time
	completed   bool
}

var authStore = struct {
	sync.RWMutex
	sessions map[string]*authSession
}{sessions: make(map[string]*authSession)}

// validSessions — сессии, прошедшие аутентификацию (для проверки сервером)
var validSessions = struct {
	sync.RWMutex
	m map[string]string // session_uuid -> nickname
}{m: make(map[string]string)}

const validSessionsFile = "valid-sessions.json"
const authCodeTTL = 5 * time.Minute
const authCodeLength = 6

func generateAuthCode() (string, error) {
	b := make([]byte, authCodeLength/2)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:authCodeLength], nil
}

func generateSessionUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func cleanupExpiredAuthSessions() {
	authStore.Lock()
	defer authStore.Unlock()
	now := time.Now()
	for code, s := range authStore.sessions {
		if now.Sub(s.createdAt) > authCodeTTL {
			delete(authStore.sessions, code)
		}
	}
}

// loadValidSessions загружает сессии из файла (переживают перезапуск backend)
func loadValidSessions() {
	data, err := os.ReadFile(validSessionsFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[Auth] Не удалось загрузить %s: %v", validSessionsFile, err)
		}
		return
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		log.Printf("[Auth] Невалидный %s: %v", validSessionsFile, err)
		return
	}
	validSessions.Lock()
	n := 0
	for k, v := range m {
		if k != "" && v != "" {
			validSessions.m[k] = v
			n++
		}
	}
	validSessions.Unlock()
	log.Printf("[Auth] Загружено %d сессий из %s", n, validSessionsFile)
}

// saveValidSessions сохраняет сессии в файл
func saveValidSessions() {
	validSessions.RLock()
	m := make(map[string]string, len(validSessions.m))
	for k, v := range validSessions.m {
		m[k] = v
	}
	validSessions.RUnlock()

	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		log.Printf("[Auth] Ошибка сериализации сессий: %v", err)
		return
	}
	if err := os.WriteFile(validSessionsFile, data, 0600); err != nil {
		log.Printf("[Auth] Ошибка записи %s: %v", validSessionsFile, err)
	}
}

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
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func main() {
	rescan := flag.Bool("rescan", false, "Пересчитать config.json из files/mods и files/versions")
	flag.Parse()

	// Проверить существование config.json, если нет - сгенерировать
	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
		log.Println("config.json не найден, генерирую автоматически...")

		baseURL := os.Getenv("BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost"
		}
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
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

	// Режим -rescan: пересчитать mods и client_files из папок, сохранить и выйти
	if *rescan {
		if err := rescanConfig(); err != nil {
			log.Fatalf("Ошибка пересчёта конфигурации: %v", err)
		}
		log.Println("config.json обновлён. Запустите сервер без -rescan.")
		return
	}

	loadValidSessions()

	// Настроить маршруты (cors разрешает запросы из Swagger UI и других клиентов)
	cors := corsHandler{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/version", cors.wrap(handleVersion))
	mux.HandleFunc("/api/launcher/version", cors.wrap(handleLauncherVersion))
	mux.HandleFunc("/api/jdk/info", cors.wrap(handleJDKInfo))
	mux.HandleFunc("/api/auth/init", cors.wrap(handleAuthInit))
	mux.HandleFunc("/api/auth/check", cors.wrap(handleAuthCheck))
	mux.HandleFunc("/api/auth/complete", cors.wrap(handleAuthComplete))
	mux.HandleFunc("/api/auth/verify", cors.wrap(handleAuthVerify))
	mux.HandleFunc("/files/", cors.wrap(handleFileDownload))

	port := config.Port
	if port == "" {
		port = "8080"
	}

	go StartTelegramBot()

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
	// Токен бота: переменная окружения имеет приоритет (безопасность)
	if envToken := os.Getenv("TELEGRAM_BOT_TOKEN"); envToken != "" {
		config.TelegramBotToken = envToken
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

// handleAuthInit обрабатывает POST /api/auth/init
func handleAuthInit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	code, err := generateAuthCode()
	if err != nil {
		log.Printf("Ошибка генерации кода: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	botUsername := config.TelegramBotUsername
	if botUsername == "" {
		botUsername = "YourLauncherBot"
	}
	botURL := fmt.Sprintf("https://t.me/%s?start=%s", strings.TrimPrefix(botUsername, "@"), code)

	authStore.Lock()
	authStore.sessions[code] = &authSession{code: code, createdAt: time.Now()}
	authStore.Unlock()

	go cleanupExpiredAuthSessions()

	log.Printf("[Auth] Init: code=%s, bot_url=%s", code, botURL)

	resp := AuthInitResponse{Code: code, BotURL: botURL}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Ошибка кодирования JSON: %v", err)
	}
}

// handleAuthCheck обрабатывает GET /api/auth/check?code=XXX
func handleAuthCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "Параметр code обязателен", http.StatusBadRequest)
		return
	}

	authStore.RLock()
	s, ok := authStore.sessions[code]
	authStore.RUnlock()

	if !ok {
		resp := AuthCheckResponse{Status: "pending"}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(resp)
		return
	}

	if time.Since(s.createdAt) > authCodeTTL {
		authStore.Lock()
		delete(authStore.sessions, code)
		authStore.Unlock()
		resp := AuthCheckResponse{Status: "pending"}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(resp)
		return
	}

	if s.completed {
		log.Printf("[Auth] Check: code=%s -> authenticated, nickname=%s", code, s.nickname)
		resp := AuthCheckResponse{Status: "authenticated", Nickname: s.nickname, SessionUUID: s.sessionUUID}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := AuthCheckResponse{Status: "pending"}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(resp)
}

// completeAuth завершает аутентификацию по коду. Вызывается из handleAuthComplete и из Telegram-бота.
// Возвращает ошибку при невалидном коде или истёкшем TTL.
func completeAuth(code, nickname string, telegramID int64) error {
	authStore.Lock()
	defer authStore.Unlock()

	s, ok := authStore.sessions[code]
	if !ok {
		return fmt.Errorf("код не найден")
	}
	if time.Since(s.createdAt) > authCodeTTL {
		delete(authStore.sessions, code)
		return fmt.Errorf("код истёк")
	}
	if s.completed {
		return fmt.Errorf("сессия уже завершена")
	}

	s.nickname = strings.TrimSpace(nickname)
	s.sessionUUID = generateSessionUUID()
	s.completed = true

	validSessions.Lock()
	validSessions.m[s.sessionUUID] = s.nickname
	validSessions.Unlock()
	saveValidSessions()

	log.Printf("[Auth] Сессия завершена: code=%s, nickname=%s, telegram_id=%d, session_uuid=%s", code, s.nickname, telegramID, s.sessionUUID)
	return nil
}

// handleAuthComplete обрабатывает POST /api/auth/complete (вызывает Telegram-бот)
func handleAuthComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	var req AuthCompleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[Auth] handleAuthComplete: невалидный JSON: %v", err)
		http.Error(w, "Невалидный JSON", http.StatusBadRequest)
		return
	}
	if req.Code == "" || req.Nickname == "" {
		http.Error(w, "Поля code и nickname обязательны", http.StatusBadRequest)
		return
	}

	log.Printf("[Auth] Запрос complete: code=%s, nickname=%s, telegram_id=%d", req.Code, req.Nickname, req.TelegramID)

	if err := completeAuth(req.Code, req.Nickname, req.TelegramID); err != nil {
		log.Printf("[Auth] completeAuth ошибка: %v", err)
		if err.Error() == "код не найден" || err.Error() == "код истёк" {
			http.Error(w, err.Error(), http.StatusGone)
			return
		}
		if err.Error() == "сессия уже завершена" {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleAuthVerify обрабатывает GET /api/auth/verify?nickname=X&session_uuid=Y (для Minecraft-сервера)
func handleAuthVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	nickname := strings.TrimSpace(r.URL.Query().Get("nickname"))
	sessionUUID := strings.TrimSpace(r.URL.Query().Get("session_uuid"))

	if nickname == "" || sessionUUID == "" {
		http.Error(w, "Параметры nickname и session_uuid обязательны", http.StatusBadRequest)
		return
	}

	validSessions.RLock()
	storedNickname, ok := validSessions.m[sessionUUID]
	validSessions.RUnlock()

	valid := ok && storedNickname == nickname
	if valid {
		log.Printf("[Auth] Verify OK: nickname=%s, session_uuid=%s", nickname, sessionUUID)
	} else {
		log.Printf("[Auth] Verify FAIL: nickname=%s, session_uuid=%s (ok=%v)", nickname, sessionUUID, ok)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]bool{"valid": valid})
}
