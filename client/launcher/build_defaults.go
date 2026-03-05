package main

// Дефолтные значения, задаваемые при сборке через -ldflags "-X main.buildDefaultApiBaseUrl=...".
// Загружаются из .env при вызове build.ps1.
var (
	buildDefaultApiBaseUrl string // API_BASE_URL — URL бэкенда
	buildDefaultServerHost string // SERVER_HOST — IP/хост Minecraft-сервера
	buildDefaultServerPort string // SERVER_PORT — порт Minecraft-сервера
)
