package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config хранит настройки лаунчера
type Config struct {
	// NewsFeedUrl — RSS URL ленты новостей (из Telegram через ch2rss и т.п.)
	NewsFeedUrl string `json:"newsFeedUrl"`
	// ApiBaseUrl — базовый URL бэкенда
	ApiBaseUrl string `json:"apiBaseUrl"`
	// ServerHost — IP/хост сервера Minecraft для автоподключения (передаётся в --server)
	ServerHost string `json:"server_host"`
	// ServerPort — порт сервера Minecraft (передаётся в --port)
	ServerPort int `json:"server_port"`
	// SyncClientSettings — синхронизировать settings-файлы клиента (например options.txt)
	SyncClientSettings bool `json:"sync_client_settings"`
}

const configFilename = "launcher-config.json"

func getConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return configFilename, nil
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "FamMCLauncher", "config", configFilename), nil
}

// LoadConfig загружает конфиг из файла
func LoadConfig() (*Config, error) {
	path, err := getConfigPath()
	if err != nil {
		return defaultConfig(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultConfig(), nil
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return defaultConfig(), nil
	}
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(data, &raw)
	if cfg.NewsFeedUrl == "" {
		cfg.NewsFeedUrl = "mc_fam"
	}
	if _, ok := raw["sync_client_settings"]; !ok {
		cfg.SyncClientSettings = true
	}
	// Применяем дефолты из .env (при сборке), если в конфиге пусто
	if cfg.ApiBaseUrl == "" && buildDefaultApiBaseUrl != "" {
		cfg.ApiBaseUrl = buildDefaultApiBaseUrl
	}
	if cfg.ServerHost == "" && buildDefaultServerHost != "" {
		cfg.ServerHost = buildDefaultServerHost
	}
	if cfg.ServerPort <= 0 && buildDefaultServerPort != "" {
		if p, err := parseInt(buildDefaultServerPort); err == nil && p > 0 {
			cfg.ServerPort = p
		}
	}
	return &cfg, nil
}

// SaveConfig сохраняет конфиг в файл
func SaveConfig(cfg *Config) error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func defaultConfig() *Config {
	port := 0
	if buildDefaultServerPort != "" {
		if p, err := parseInt(buildDefaultServerPort); err == nil && p > 0 {
			port = p
		}
	}
	return &Config{
		NewsFeedUrl:        "mc_fam",
		ApiBaseUrl:         buildDefaultApiBaseUrl,
		ServerHost:         buildDefaultServerHost,
		ServerPort:         port,
		SyncClientSettings: true,
	}
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
