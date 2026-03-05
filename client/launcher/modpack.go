package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ModpackConfig — корневая структура modpack.json (формат Mojang/Fabric)
type ModpackConfig struct {
	ID                    string             `json:"id,omitempty"`
	Time                  string             `json:"time,omitempty"`
	ReleaseTime           string             `json:"releaseTime,omitempty"`
	Type                  string             `json:"type,omitempty"`
	MainClass             string             `json:"mainClass,omitempty"`
	MinimumLauncherVersion *int              `json:"minimumLauncherVersion,omitempty"`
	Assets                string             `json:"assets,omitempty"`
	Arguments             *ModpackArguments  `json:"arguments,omitempty"`
	Libraries             []ModpackLibrary   `json:"libraries,omitempty"`
	AssetIndex            *ModpackAssetIndex `json:"assetIndex,omitempty"`
	Downloads             *ModpackDownloads  `json:"downloads,omitempty"`
	JavaVersion           *ModpackJavaVersion `json:"javaVersion,omitempty"`
	Logging               *ModpackLogging    `json:"logging,omitempty"`
}

// ModpackArguments — JVM и игровые аргументы
type ModpackArguments struct {
	JVM             []ModpackArgumentEntry `json:"jvm,omitempty"`
	Game            []ModpackArgumentEntry `json:"game,omitempty"`
	DefaultUserJVM  []string               `json:"default_user_jvm,omitempty"`
}

// ModpackArgumentEntry — элемент аргументов с опциональными правилами
type ModpackArgumentEntry struct {
	Values []string     `json:"values,omitempty"`
	Rules  []ModpackRule `json:"rules,omitempty"`
}

// ModpackRule — правило применения (по ОС или features)
type ModpackRule struct {
	Action   string              `json:"action,omitempty"`
	OS       *ModpackRuleOs      `json:"os,omitempty"`
	Features *ModpackRuleFeatures `json:"features,omitempty"`
}

// ModpackRuleOs — условие по ОС: name = "windows" | "linux" | "osx"
type ModpackRuleOs struct {
	Name string `json:"name,omitempty"`
	Arch string `json:"arch,omitempty"`
}

// ModpackRuleFeatures — условие по features
type ModpackRuleFeatures struct {
	IsDemoUser              *bool `json:"is_demo_user,omitempty"`
	HasCustomResolution     *bool `json:"has_custom_resolution,omitempty"`
	HasQuickPlaysSupport    *bool `json:"has_quick_plays_support,omitempty"`
	IsQuickPlaySingleplayer *bool `json:"is_quick_play_singleplayer,omitempty"`
	IsQuickPlayMultiplayer  *bool `json:"is_quick_play_multiplayer,omitempty"`
	IsQuickPlayRealms       *bool `json:"is_quick_play_realms,omitempty"`
}

// ModpackLibrary — библиотека из modpack.json
type ModpackLibrary struct {
	Name      string           `json:"name,omitempty"`
	Rules     []ModpackRule    `json:"rules,omitempty"`
	Artifact  *ModpackArtifact `json:"artifact,omitempty"`
	Downloads []ModpackArtifact `json:"downloads,omitempty"`
}

// ModpackArtifact — артефакт библиотеки (path, url, sha1, size)
type ModpackArtifact struct {
	Path  string `json:"path,omitempty"`
	URL   string `json:"url,omitempty"`
	SHA1  string `json:"sha1,omitempty"`
	Size  int64  `json:"size,omitempty"`
}

// ModpackAssetIndex — метаданные индекса ассетов
type ModpackAssetIndex struct {
	ID        string `json:"id,omitempty"`
	TotalSize int64  `json:"totalSize,omitempty"`
	Size      int64  `json:"size,omitempty"`
	URL       string `json:"url,omitempty"`
}

// ModpackDownloads — ссылки на загрузки (client, server и т.д.)
type ModpackDownloads struct {
	Client         *ModpackArtifact `json:"client,omitempty"`
	ClientMappings *ModpackArtifact `json:"client_mappings,omitempty"`
	Server         *ModpackArtifact `json:"server,omitempty"`
	ServerMappings *ModpackArtifact `json:"server_mappings,omitempty"`
}

// ModpackJavaVersion — требования к версии Java
type ModpackJavaVersion struct {
	Component    string  `json:"component,omitempty"`
	MajorVersion float64 `json:"majorVersion,omitempty"` // в JSON может быть 21 или 21.0
}

// ModpackLogging — конфигурация логирования (Log4j2)
type ModpackLogging struct {
	Client *ModpackLoggingClient `json:"client,omitempty"`
}

// ModpackLoggingClient — клиентская конфигурация логирования
type ModpackLoggingClient struct {
	Argument string              `json:"argument,omitempty"`
	File     *ModpackLoggingFile `json:"file,omitempty"`
	Type     string              `json:"type,omitempty"`
}

// ModpackLoggingFile — файл конфигурации логирования
type ModpackLoggingFile struct {
	ID   string `json:"id,omitempty"`
	SHA1 string `json:"sha1,omitempty"`
	Size int64  `json:"size,omitempty"`
	URL  string `json:"url,omitempty"`
}

// getLauncherDir возвращает папку лаунчера (рядом с exe)
func getLauncherDir() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	if dir == "" {
		return ".", nil
	}
	return dir, nil
}

// LoadModpack загружает modpack с бэкенда (GET /api/modpack). Не сохраняет локально.
func LoadModpack() (*ModpackConfig, error) {
	cfg, _ := LoadConfig()
	base := ""
	if cfg != nil && cfg.ApiBaseUrl != "" {
		base = strings.TrimSuffix(cfg.ApiBaseUrl, "/")
	}
	if base == "" {
		return nil, fmt.Errorf("укажите URL API в настройках")
	}

	url := base + "/api/modpack"
	mp, fetchErr := fetchModpackFromURL(url)
	if fetchErr != nil || mp == nil {
		return nil, fmt.Errorf("не удалось загрузить modpack с бэкенда (%s): %w. Проверьте URL в настройках и что сервер запущен", base, fetchErr)
	}
	return mp, nil
}

func fetchModpackFromURL(url string) (*ModpackConfig, error) {
	resp, err := getWithRetry(url, httpTimeoutModpack)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var mp ModpackConfig
	if err := json.Unmarshal(body, &mp); err != nil {
		return nil, err
	}
	return &mp, nil
}
