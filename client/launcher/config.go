package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	return &Config{
		NewsFeedUrl:        "mc_fam",
		ApiBaseUrl:         "",
		ServerHost:         "",
		ServerPort:         0,
		SyncClientSettings: true,
	}
}

// resolveNewsFeedUrl превращает имя канала или t.me/xxx в RSS URL для ch2rss
func resolveNewsFeedUrl(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}
	// Уже полный URL
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return input
	}
	// t.me/channel или t.me/s/channel
	re := regexp.MustCompile(`(?:t\.me/s?/)?([a-zA-Z0-9_]+)`)
	if m := re.FindStringSubmatch(input); len(m) > 1 {
		channel := m[1]
		if len(channel) >= 5 && len(channel) <= 32 {
			return "https://ch2rss.fflow.net/" + channel
		}
	}
	// Просто имя канала
	if len(input) >= 5 && len(input) <= 32 && regexp.MustCompile(`^\w+$`).MatchString(input) {
		return "https://ch2rss.fflow.net/" + input
	}
	return input
}
