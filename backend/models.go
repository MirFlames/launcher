package main

// ServerInfo представляет структуру ответа API
type ServerInfo struct {
	MinecraftVersion string       `json:"minecraft_version"`
	ModsHash         string       `json:"mods_hash"`
	ClientFiles      []ClientFile `json:"client_files"`
	Mods             []ModFile    `json:"mods"`
}

// ClientFile представляет файл клиента
type ClientFile struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Hash string `json:"hash"`
}

// ModFile представляет файл мода
type ModFile struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Hash string `json:"hash"`
}

// Config представляет конфигурацию сервера
type Config struct {
	MinecraftVersion string       `json:"minecraft_version"`
	ModsHash         string       `json:"mods_hash"`
	ClientFiles      []ClientFile `json:"client_files"`
	Mods             []ModFile    `json:"mods"`
	FilesPath        string       `json:"files_path"`
	Port             string       `json:"port"`
	// Конфигурация лаунчера
	LauncherVersion  string `json:"launcher_version"`
	LauncherDownloadURL string `json:"launcher_download_url"`
	LauncherHash     string `json:"launcher_hash"`
	LauncherSize     int64  `json:"launcher_size"`
	LauncherMandatory bool  `json:"launcher_mandatory"`
	// Telegram бот для аутентификации
	TelegramBotUsername    string `json:"telegram_bot_username"`
	TelegramBotToken       string `json:"telegram_bot_token"`
	TelegramRequiredChannel string `json:"telegram_required_channel"` // Канал для обязательной подписки (например @mc_fam)
	// Конфигурация JDK
	JDK JDKInfo `json:"jdk"`
}

// LauncherVersion представляет информацию о версии лаунчера
type LauncherVersion struct {
	Version      string `json:"version"`
	DownloadURL  string `json:"download_url"`
	Hash         string `json:"hash"`
	Size         int64  `json:"size"`
	ReleaseNotes string `json:"release_notes,omitempty"`
	Mandatory    bool   `json:"mandatory"`
}

// JDKInfo представляет информацию о требуемом JDK
type JDKInfo struct {
	Version        string `json:"version"`         // Формат: jdk-21.0.2
	RelativePath   string `json:"relative_path"`  // Относительный путь от папки Minecraft
	JavaExecutable string `json:"java_executable"` // Путь к java.exe относительно JDK
	Mandatory      bool   `json:"mandatory"`
}

// AuthInitResponse — ответ на POST /api/auth/init
type AuthInitResponse struct {
	Code    string `json:"code"`
	BotURL  string `json:"bot_url"`
}

// AuthCheckResponse — ответ на GET /api/auth/check?code=XXX
type AuthCheckResponse struct {
	Status       string `json:"status"` // "pending" или "authenticated"
	Nickname     string `json:"nickname,omitempty"`
	SessionUUID  string `json:"session_uuid,omitempty"`
}

// AuthCompleteRequest — тело запроса POST /api/auth/complete
type AuthCompleteRequest struct {
	Code             string `json:"code"`
	Nickname         string `json:"nickname"`
	TelegramID       int64  `json:"telegram_id"`
	TelegramUsername string `json:"telegram_username,omitempty"`
}

// ValidSessionEntry — запись в valid-sessions.json (никнейм, Telegram ID и логин)
type ValidSessionEntry struct {
	Nickname         string `json:"nickname"`
	TelegramID       int64  `json:"telegram_id"`
	TelegramUsername string `json:"telegram_username"`
}
