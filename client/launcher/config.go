package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config хранит настройки лаунчера
type Config struct {
	// ApiBaseUrl — базовый URL бэкенда
	ApiBaseUrl string `json:"apiBaseUrl"`
	// ServerHost — IP/хост сервера Minecraft для автоподключения (передаётся в --server)
	ServerHost string `json:"server_host"`
	// ServerPort — порт сервера Minecraft (передаётся в --port)
	ServerPort int `json:"server_port"`
	// SocksProxyHost — SOCKS5 прокси для Java-запросов (authlib-injector и запросы к backend)
	SocksProxyHost string `json:"socks_proxy_host"`
	// SocksProxyPort — порт SOCKS5 прокси
	SocksProxyPort int `json:"socks_proxy_port"`
	// SyncClientSettings — синхронизировать settings-файлы клиента (например options.txt)
	SyncClientSettings bool `json:"sync_client_settings"`
	// AuthlibInjectorDebug — добавить -Dauthlibinjector.debug=verbose,authlib при запуске Java
	AuthlibInjectorDebug bool `json:"authlib_injector_debug"`
	// SkipServerModSync — не скачивать моды и конфиги модов с бэкенда (разработка)
	SkipServerModSync bool `json:"skip_server_mod_sync"`
}

const configFilename = "launcher-config.json"

func getAppConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return "", fmt.Errorf("не удалось определить директорию: %w", err)
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "FamMCLauncher"), nil
}

func getConfigPath() (string, error) {
	dir, err := getAppConfigDir()
	if err != nil {
		return configFilename, nil
	}
	return filepath.Join(dir, "config", configFilename), nil
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
	if cfg.SocksProxyHost == "" && buildDefaultSocksProxyHost != "" {
		cfg.SocksProxyHost = buildDefaultSocksProxyHost
	}
	if cfg.SocksProxyPort <= 0 && buildDefaultSocksProxyPort != "" {
		if p, err := parseInt(buildDefaultSocksProxyPort); err == nil && p > 0 {
			cfg.SocksProxyPort = p
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
	if err := os.MkdirAll(filepath.Dir(path), dirMode); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, configFileMode)
}

func defaultConfig() *Config {
	port := 0
	if buildDefaultServerPort != "" {
		if p, err := parseInt(buildDefaultServerPort); err == nil && p > 0 {
			port = p
		}
	}
	socksPort := 0
	if buildDefaultSocksProxyPort != "" {
		if p, err := parseInt(buildDefaultSocksProxyPort); err == nil && p > 0 {
			socksPort = p
		}
	}
	return &Config{
		ApiBaseUrl:         buildDefaultApiBaseUrl,
		ServerHost:         buildDefaultServerHost,
		ServerPort:         port,
		SocksProxyHost:     buildDefaultSocksProxyHost,
		SocksProxyPort:     socksPort,
		SyncClientSettings: true,
	}
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
