package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Имена профилей окружения (значения поля Config.Env).
const (
	envProd = "prod"
	envDev  = "dev"
)

// EnvProfile — сетевой набор одного окружения (прод / dev-стенд).
// Пустое поле означает «взять дефолт сборки» (ldflags из .env), поэтому
// нетронутый прод-профиль не замораживает адреса в конфиге пользователя
// и продолжает следовать за тем, с чем собран лаунчер.
type EnvProfile struct {
	ApiBaseUrl     string `json:"apiBaseUrl,omitempty"`
	ServerHost     string `json:"server_host,omitempty"`
	ServerPort     int    `json:"server_port,omitempty"`
	SocksProxyHost string `json:"socks_proxy_host,omitempty"`
	SocksProxyPort int    `json:"socks_proxy_port,omitempty"`
}

// Config хранит настройки лаунчера
//
// Поля ApiBaseUrl/ServerHost/ServerPort/SocksProxy* — ЭФФЕКТИВНЫЕ значения:
// их читает весь остальной код (downloader, jdk, modpack, launch, auth).
// Активный профиль окружения (Env + EnvProfiles) лишь копируется в них при
// сохранении настроек, поэтому переключение окружения ничего больше не ломает.
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
	// Env — активный профиль окружения ("prod" | "dev"); пустая строка = "prod"
	Env string `json:"env,omitempty"`
	// EnvProfiles — сохранённые наборы адресов по окружениям (ключи "prod"/"dev")
	EnvProfiles map[string]EnvProfile `json:"env_profiles,omitempty"`
	// AuthlibInjectorDebug — добавить -Dauthlibinjector.debug=verbose,authlib при запуске Java
	AuthlibInjectorDebug bool `json:"authlib_injector_debug"`
	// SkipServerModSync — не скачивать моды и конфиги модов с бэкенда (разработка)
	SkipServerModSync bool `json:"skip_server_mod_sync"`
	// DevMarketingSessionID — dev-only значение для --marketing-session-id
	DevMarketingSessionID string `json:"dev_marketing_session_id"`
	// DevSourceCode — dev-only значение для --source-code
	DevSourceCode string `json:"dev_source_code"`
	// DevUTMSource — dev-only значение для --utm-source
	DevUTMSource string `json:"dev_utm_source"`
	// DevUTMMedium — dev-only значение для --utm-medium
	DevUTMMedium string `json:"dev_utm_medium"`
	// DevUTMCampaign — dev-only значение для --utm-campaign
	DevUTMCampaign string `json:"dev_utm_campaign"`
	// DevUTMContent — dev-only значение для --utm-content
	DevUTMContent string `json:"dev_utm_content"`
	// DevLandingPath — dev-only значение для --landing-path
	DevLandingPath string `json:"dev_landing_path"`
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
	// Раскладываем старый одиночный набор адресов по профилям (только в памяти,
	// файл при этом не переписываем — LoadConfig вызывается часто и писать не должен).
	cfg.migrateEnvProfiles()
	// Применяем дефолты из .env (при сборке), если в конфиге пусто
	d := buildDefaultProfile()
	if cfg.ApiBaseUrl == "" {
		cfg.ApiBaseUrl = d.ApiBaseUrl
	}
	if cfg.ServerHost == "" {
		cfg.ServerHost = d.ServerHost
	}
	if cfg.ServerPort <= 0 {
		cfg.ServerPort = d.ServerPort
	}
	if cfg.SocksProxyHost == "" {
		cfg.SocksProxyHost = d.SocksProxyHost
	}
	if cfg.SocksProxyPort <= 0 {
		cfg.SocksProxyPort = d.SocksProxyPort
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
	d := buildDefaultProfile()
	return &Config{
		ApiBaseUrl:     d.ApiBaseUrl,
		ServerHost:     d.ServerHost,
		ServerPort:     d.ServerPort,
		SocksProxyHost: d.SocksProxyHost,
		SocksProxyPort: d.SocksProxyPort,
		Env:            envProd,
	}
}

// buildDefaultProfile — набор адресов, с которым собран лаунчер (ldflags из .env).
func buildDefaultProfile() EnvProfile {
	return EnvProfile{
		ApiBaseUrl:     buildDefaultApiBaseUrl,
		ServerHost:     buildDefaultServerHost,
		ServerPort:     parsePositiveInt(buildDefaultServerPort),
		SocksProxyHost: buildDefaultSocksProxyHost,
		SocksProxyPort: parsePositiveInt(buildDefaultSocksProxyPort),
	}
}

// EnvName — активный профиль; всё, что не "dev", считаем прод-профилем.
func (c *Config) EnvName() string {
	if c != nil && c.Env == envDev {
		return envDev
	}
	return envProd
}

// Profile — сохранённый набор окружения. Для прода пустые поля дозаполняются
// дефолтами сборки; для dev остаются пустыми — иначе недонастроенный стенд
// молча увёл бы разработчика на прод-адреса.
func (c *Config) Profile(env string) EnvProfile {
	var p EnvProfile
	if c != nil {
		p = c.EnvProfiles[env]
	}
	if env == envDev {
		return p
	}
	d := buildDefaultProfile()
	if p.ApiBaseUrl == "" {
		p.ApiBaseUrl = d.ApiBaseUrl
	}
	if p.ServerHost == "" {
		p.ServerHost = d.ServerHost
	}
	if p.ServerPort <= 0 {
		p.ServerPort = d.ServerPort
	}
	if p.SocksProxyHost == "" {
		p.SocksProxyHost = d.SocksProxyHost
	}
	if p.SocksProxyPort <= 0 {
		p.SocksProxyPort = d.SocksProxyPort
	}
	return p
}

// ApplyProfile делает профиль активным: копирует его в эффективные поля конфига.
func (c *Config) ApplyProfile(env string) {
	if c == nil {
		return
	}
	if env != envDev {
		env = envProd
	}
	p := c.Profile(env)
	c.Env = env
	c.ApiBaseUrl = p.ApiBaseUrl
	c.ServerHost = p.ServerHost
	c.ServerPort = p.ServerPort
	c.SocksProxyHost = p.SocksProxyHost
	c.SocksProxyPort = p.SocksProxyPort
}

// migrateEnvProfiles — конфиги, созданные до появления профилей, хранят один
// набор адресов. Раскладываем его: прод-профиль остаётся пустым (= дефолты
// сборки), а набор, отличающийся от сборки, считаем dev-стендом. Эффективные
// поля не трогаем — поведение уже установленного лаунчера не меняется.
func (c *Config) migrateEnvProfiles() {
	if c == nil || c.EnvProfiles != nil {
		return
	}
	c.EnvProfiles = map[string]EnvProfile{}
	current := EnvProfile{
		ApiBaseUrl:     c.ApiBaseUrl,
		ServerHost:     c.ServerHost,
		ServerPort:     c.ServerPort,
		SocksProxyHost: c.SocksProxyHost,
		SocksProxyPort: c.SocksProxyPort,
	}
	if current == (EnvProfile{}) || current == buildDefaultProfile() {
		c.Env = envProd
		return
	}
	c.EnvProfiles[envDev] = current
	c.Env = envDev
}

func parsePositiveInt(s string) int {
	if s == "" {
		return 0
	}
	if n, err := parseInt(s); err == nil && n > 0 {
		return n
	}
	return 0
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
