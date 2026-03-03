package main

import (
	"crypto/rand"
	_ "embed"
	"encoding/base64"
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

	"github.com/joho/godotenv"
)

//go:embed static/404.png
var img404PNG []byte

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

// getStoredEntryByTelegramID возвращает запись сессии по telegram_id (для повторного входа без запроса никнейма)
func getStoredEntryByTelegramID(telegramID int64) (ValidSessionEntry, bool) {
	return sessionGetByTelegramID(telegramID)
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

// serve404Page отвечает страницей с картинкой при 404
func serve404Page(w http.ResponseWriter, r *http.Request) {
	log.Printf("404 Not Found: %s %s", r.Method, r.URL.Path)
	b64 := base64.StdEncoding.EncodeToString(img404PNG)
	html := `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>404 Not Found</title></head>
<body style="margin:0;display:flex;justify-content:center;align-items:center;min-height:100vh;background:#1a1a2e;">
<img src="data:image/png;base64,` + b64 + `" alt="404" style="max-width:100%;height:auto;">
</body>
</html>`
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte(html))
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

func requireEnv(name string) string {
	v := os.Getenv(name)
	if v == "" {
		log.Fatalf("Переменная окружения %s обязательна. Задайте её в .env", name)
	}
	return v
}

func main() {
	for _, p := range []string{".env", "../.env"} {
		if err := godotenv.Load(p); err == nil {
			break
		}
	}

	rescan := flag.Bool("rescan", false, "Пересчитать config.json из files/mods и files/versions")
	updateHashes := flag.Bool("update-hashes", false, "Пересчитать хэши модов и client_files в config.json")
	flag.Parse()

	// Проверить существование config.json, если нет - сгенерировать
	if _, err := os.Stat("config.json"); os.IsNotExist(err) {
		log.Println("config.json не найден, генерирую автоматически...")
		baseURL := requireEnv("BASE_URL")
		port := requireEnv("PORT")
		filesPath := requireEnv("FILES_PATH")
		if err := generateConfig(filesPath, port, baseURL); err != nil {
			log.Fatalf("Ошибка генерации конфигурации: %v", err)
		}
	}

	// Загрузить конфигурацию
	if err := loadConfig("config.json"); err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Обязательные переменные из env (переопределяют config)
	_ = requireEnv("BASE_URL") // используется в hashutil
	config.Port = requireEnv("PORT")
	config.FilesPath = requireEnv("FILES_PATH")
	config.ServerHost = requireEnv("SERVER_HOST")
	config.ServerPort = requireEnv("SERVER_PORT")
	config.TelegramBotToken = requireEnv("TELEGRAM_BOT_TOKEN")
	config.TelegramBotUsername = requireEnv("TELEGRAM_BOT_USERNAME")
	config.TelegramRequiredChannel = requireEnv("TELEGRAM_REQUIRED_CHANNEL")

	// Режим -update-hashes: пересчитать хэши модов и client_files, сохранить и выйти
	if *updateHashes {
		if err := updateConfigHashes(false); err != nil {
			log.Fatalf("Ошибка пересчёта хэшей: %v", err)
		}
		log.Println("Хэши обновлены в config.json")
		return
	}

	// Режим -rescan: пересчитать mods и client_files из папок, сохранить и выйти
	if *rescan {
		if err := rescanConfig(); err != nil {
			log.Fatalf("Ошибка пересчёта конфигурации: %v", err)
		}
		log.Println("config.json обновлён. Запустите сервер без -rescan.")
		return
	}

	initSessionsDB()
	LoadNewsCache()

	// Настроить маршруты (cors разрешает запросы из Swagger UI и других клиентов)
	cors := corsHandler{}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/version", cors.wrap(handleVersion))
	mux.HandleFunc("/api/launcher/version", cors.wrap(handleLauncherVersion))
	mux.HandleFunc("/api/jdk/info", cors.wrap(handleJDKInfo))
	mux.HandleFunc("/api/modpack", cors.wrap(handleModpack))
	mux.HandleFunc("/api/auth/init", cors.wrap(handleAuthInit))
	mux.HandleFunc("/api/auth/check", cors.wrap(handleAuthCheck))
	mux.HandleFunc("/api/auth/complete", cors.wrap(handleAuthComplete))
	mux.HandleFunc("/api/auth/verify", cors.wrap(handleAuthVerify))
	mux.HandleFunc("/api/auth/invalidate", cors.wrap(handleAuthInvalidate))
	mux.HandleFunc("/api/news", cors.wrap(handleNews))
	mux.HandleFunc("/files/", cors.wrap(handleFileDownload))
	mux.HandleFunc("/", cors.wrap(serve404Page))

	go StartTelegramBot()
	go startHashesWatcher()

	log.Printf("Сервер запущен")
	log.Printf("Файлы раздаются из: %s", config.FilesPath)
	log.Fatal(http.ListenAndServe(":8080", log404Middleware(mux)))
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

	// JDK и LauncherVersion — из config (не секреты)
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
	// PORT, FILES_PATH, SERVER_HOST, TELEGRAM_* задаются из env в main()

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

// handleModpack обрабатывает запрос GET /api/modpack — возвращает modpack.json
func handleModpack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}
	modpackPath := filepath.Join(config.FilesPath, "configs", "modpack.json")
	data, err := os.ReadFile(modpackPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "modpack.json не найден", http.StatusNotFound)
			return
		}
		log.Printf("Ошибка чтения modpack.json: %v", err)
		http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
}

// buildFullURL собирает полный URL из BASE_URL, PORT и относительного пути.
// Если path уже полный URL (http:// или https://), возвращает как есть (обратная совместимость).
func buildFullURL(baseURL, port, path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	return strings.TrimSuffix(baseURL, "/") + ":" + port + path
}

// handleVersion обрабатывает запрос GET /api/version
func handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("Запрошена информация о версии игры.")

	baseURL := os.Getenv("BASE_URL")
	port := config.Port

	clientFiles := make([]ClientFile, len(config.ClientFiles))
	for i, f := range config.ClientFiles {
		clientFiles[i] = ClientFile{Name: f.Name, URL: buildFullURL(baseURL, port, f.URL), Hash: f.Hash}
	}
	mods := make([]ModFile, len(config.Mods))
	for i, m := range config.Mods {
		mods[i] = ModFile{Name: m.Name, URL: buildFullURL(baseURL, port, m.URL), Hash: m.Hash}
	}

	serverInfo := ServerInfo{
		MinecraftVersion: config.MinecraftVersion,
		ModsHash:         config.ModsHash,
		ClientFiles:      clientFiles,
		Mods:             mods,
		ServerHost:       config.ServerHost,
		ServerPort:       config.ServerPort,
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
			serve404Page(w, r)
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

	launcherDownloadURL := fmt.Sprintf("http://%s:%s%s/launcher-%s.exe",
		config.ServerHost, config.Port, config.LauncherDownloadPath, config.LauncherVersion)

	launcherVersion := LauncherVersion{
		Version:      config.LauncherVersion,
		DownloadURL:  launcherDownloadURL,
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
		log.Printf("[Auth] TELEGRAM_BOT_USERNAME не задан")
		http.Error(w, "Бот не настроен: TELEGRAM_BOT_USERNAME обязателен", http.StatusInternalServerError)
		return
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
// С одного Telegram-аккаунта (telegram_id) может быть только один аккаунт на сервере — при повторном входе старая сессия заменяется.
func completeAuth(code, nickname, telegramUsername string, telegramID int64) error {
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

	entry := ValidSessionEntry{
		Nickname:         s.nickname,
		TelegramID:       telegramID,
		TelegramUsername: strings.TrimSpace(telegramUsername),
	}

	// Удалить старую сессию этого Telegram-аккаунта (один аккаунт на сервере)
	sessionDeleteByTelegramID(telegramID)
	if err := sessionSave(s.sessionUUID, entry); err != nil {
		log.Printf("[Auth] Ошибка сохранения сессии: %v", err)
		return err
	}

	log.Printf("[Auth] Сессия завершена: code=%s, nickname=%s, telegram_id=%d, telegram_username=%s, session_uuid=%s",
		code, s.nickname, telegramID, entry.TelegramUsername, s.sessionUUID)
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

	log.Printf("[Auth] Запрос complete: code=%s, nickname=%s, telegram_id=%d, telegram_username=%s", req.Code, req.Nickname, req.TelegramID, req.TelegramUsername)

	if err := completeAuth(req.Code, req.Nickname, req.TelegramUsername, req.TelegramID); err != nil {
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

	entry, ok := sessionGetByUUID(sessionUUID)
	valid := ok && entry.Nickname == nickname
	if valid {
		log.Printf("[Auth] Verify OK: nickname=%s, session_uuid=%s", nickname, sessionUUID)
	} else {
		log.Printf("[Auth] Verify FAIL: nickname=%s, session_uuid=%s (ok=%v)", nickname, sessionUUID, ok)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]bool{"valid": valid})
}

// handleAuthInvalidate обрабатывает POST /api/auth/invalidate?nickname=X или ?session_uuid=Y
// Удаляет сессию из БД (обнуление сессии пользователя)
func handleAuthInvalidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	nickname := strings.TrimSpace(r.URL.Query().Get("nickname"))
	sessionUUID := strings.TrimSpace(r.URL.Query().Get("session_uuid"))

	if nickname == "" && sessionUUID == "" {
		http.Error(w, "Укажите nickname или session_uuid", http.StatusBadRequest)
		return
	}

	if sessionUUID != "" {
		if _, ok := sessionGetByUUID(sessionUUID); ok {
			sessionDeleteByUUID(sessionUUID)
			log.Printf("[Auth] Инвалидация по session_uuid: %s", sessionUUID)
		}
	} else {
		sessionDeleteByNickname(nickname)
		log.Printf("[Auth] Инвалидация по nickname: %s", nickname)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
