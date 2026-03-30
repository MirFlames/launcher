package main

// Дефолтные значения, задаваемые при сборке через -ldflags "-X main.buildDefaultApiBaseUrl=...".
// Загружаются из .env при вызове build.ps1.
var (
	buildDefaultApiBaseUrl string // API_BASE_URL — URL бэкенда
	buildDefaultServerHost string // SERVER_HOST — IP/хост Minecraft-сервера
	buildDefaultServerPort string // SERVER_PORT — порт Minecraft-сервера
	buildDefaultSocksProxyHost string // SOCKS_PROXY_HOST — SOCKS5 прокси для Java (authlib/backend)
	buildDefaultSocksProxyPort string // SOCKS_PROXY_PORT — порт SOCKS5 прокси

	// Настройки механизма автообновления лаунчера.
	// Рекомендуется задавать через -ldflags или .env, чтобы не хардкодить в репозитории.
	//
	// Пример (Windows, PowerShell):
	//   go build -ldflags "-X main.buildUpdateManifestURL=https://github.com/OWNER/REPO/releases/latest/download/launcher-update.json -X main.buildUpdateSignatureURL=https://github.com/OWNER/REPO/releases/latest/download/launcher-update.json.sig -X main.buildUpdatePublicKeyHex=0123abcd..."
	//
	// buildUpdateManifestURL — URL JSON-манифеста последней версии лаунчера (в релизах GitHub).
	// buildUpdateSignatureURL — URL файла подписи (hex-строка подписи Ed25519 содержимого манифеста).
	// buildUpdatePublicKeyHex — публичный ключ Ed25519 в hex (32 байта => 64 hex-символа).
	buildUpdateManifestURL   string // UPDATE_MANIFEST_URL
	buildUpdateSignatureURL  string // UPDATE_SIGNATURE_URL
	buildUpdatePublicKeyHex  string // UPDATE_PUBLIC_KEY_HEX
)
