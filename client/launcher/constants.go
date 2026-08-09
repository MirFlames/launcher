package main

import "time"

// Константы для загрузок и HTTP
const (
	defaultAssetsIndex  = "29"
	downloadBufferSize = 32 * 1024

	httpTimeoutShort    = 15 * time.Second
	httpTimeoutModpack  = 30 * time.Second
	httpTimeoutLong     = 2 * time.Minute
	httpTimeoutJDK      = 10 * time.Minute
	httpTimeoutDownload = 30*time.Second + 2*time.Minute // connect + read для downloadFile
)

// Константы для запуска Minecraft
const (
	defaultVersionType      = "fabric"
	launcherName           = "custom"
	launcherVersionFallback = "1.0.1"

	// Имя папки с игровыми данными в пользовательском каталоге (macOS/Linux).
	launcherDataDirName = "minecraft-online"
)

// Режимы доступа к файлам
const (
	dirMode        = 0755
	authFileMode   = 0600
	configFileMode = 0644
)

// Ретраи при загрузке
const (
	downloadMaxRetries   = 3
	downloadRetryDelayMs = 1000
)

// Константы аутентификации
const (
	authPollInterval   = 2 * time.Second
	authTimeout        = 5 * time.Minute
	authCheckTimeout   = 5 * time.Second
	authVerifyTimeout  = 5 * time.Second
)
