package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LaunchProgress — callback для отображения прогресса (stage, status, progress 0-1)
type LaunchProgress func(stage, status string, progress float64)

// LaunchProcessStarted — callback при успешном запуске процесса (для скрытия окна и ожидания выхода)
type LaunchProcessStarted func(cmd *exec.Cmd)

func ruleMatchesOS(rule *ModpackRule, currentOS string) bool {
	if rule == nil || rule.OS == nil || rule.OS.Name == "" {
		return true
	}
	return strings.EqualFold(rule.OS.Name, currentOS)
}

func argumentEntryApplies(entry *ModpackArgumentEntry, currentOS string) bool {
	if entry == nil || entry.Values == nil {
		return false
	}
	if entry.Rules == nil || len(entry.Rules) == 0 {
		return true
	}
	for _, r := range entry.Rules {
		osMatch := ruleMatchesOS(&r, currentOS)
		if strings.EqualFold(r.Action, "allow") && osMatch {
			return true
		}
		if strings.EqualFold(r.Action, "disallow") && osMatch {
			return false
		}
	}
	return false
}

func argumentEntryAppliesForGame(entry *ModpackArgumentEntry, currentOS string, ctx *LaunchContext) bool {
	if entry == nil || entry.Values == nil {
		return false
	}
	if entry.Rules == nil || len(entry.Rules) == 0 {
		return true
	}
	for _, r := range entry.Rules {
		if r.OS != nil && r.OS.Name != "" {
			if strings.EqualFold(r.Action, "allow") && strings.EqualFold(r.OS.Name, currentOS) {
				return true
			}
			if strings.EqualFold(r.Action, "disallow") && strings.EqualFold(r.OS.Name, currentOS) {
				return false
			}
		}
		if r.Features != nil {
			if featureMatches(ctx, r.Features) {
				if strings.EqualFold(r.Action, "allow") {
					return true
				}
				if strings.EqualFold(r.Action, "disallow") {
					return false
				}
			}
		}
	}
	return false
}

func featureMatches(ctx *LaunchContext, f *ModpackRuleFeatures) bool {
	if f == nil {
		return false
	}
	if f.IsDemoUser != nil && *f.IsDemoUser != ctx.IsDemoUser {
		return false
	}
	if f.HasCustomResolution != nil && *f.HasCustomResolution != ctx.HasCustomResolution {
		return false
	}
	if f.HasQuickPlaysSupport != nil && *f.HasQuickPlaysSupport != ctx.HasQuickPlaysSupport {
		return false
	}
	if f.IsQuickPlaySingleplayer != nil && *f.IsQuickPlaySingleplayer != ctx.IsQuickPlaySingleplayer {
		return false
	}
	if f.IsQuickPlayMultiplayer != nil && *f.IsQuickPlayMultiplayer != ctx.IsQuickPlayMultiplayer {
		return false
	}
	if f.IsQuickPlayRealms != nil && *f.IsQuickPlayRealms != ctx.IsQuickPlayRealms {
		return false
	}
	return true
}

// LaunchContext — контекст для подстановки плейсхолдеров
type LaunchContext struct {
	GameDirectory           string
	AssetsRoot              string
	AssetsIndexName         string
	AuthPlayerName          string
	VersionName             string
	AuthUUID                string
	AuthAccessToken         string
	ClientID                string
	AuthXuid                string
	VersionType             string
	IsDemoUser              bool
	HasCustomResolution     bool
	HasQuickPlaysSupport    bool
	IsQuickPlaySingleplayer bool
	IsQuickPlayMultiplayer  bool
	IsQuickPlayRealms       bool
}

func (c *LaunchContext) substitute(s string) string {
	if s == "" {
		return s
	}
	s = strings.ReplaceAll(s, "${game_directory}", c.GameDirectory)
	s = strings.ReplaceAll(s, "${assets_root}", c.AssetsRoot)
	s = strings.ReplaceAll(s, "${assets_index_name}", c.AssetsIndexName)
	s = strings.ReplaceAll(s, "${auth_player_name}", c.AuthPlayerName)
	s = strings.ReplaceAll(s, "${version_name}", c.VersionName)
	s = strings.ReplaceAll(s, "${auth_uuid}", c.AuthUUID)
	s = strings.ReplaceAll(s, "${auth_access_token}", c.AuthAccessToken)
	s = strings.ReplaceAll(s, "${clientid}", c.ClientID)
	s = strings.ReplaceAll(s, "${auth_xuid}", c.AuthXuid)
	s = strings.ReplaceAll(s, "${version_type}", c.VersionType)
	return s
}

func resolveJvmArguments(modpack *ModpackConfig, nativesDir, classpath, launcherName, launcherVersion string) []string {
	currentOS := getCurrentOS()
	var result []string
	if modpack.Arguments == nil || modpack.Arguments.JVM == nil {
		return result
	}
	for _, entry := range modpack.Arguments.JVM {
		if !argumentEntryApplies(&entry, currentOS) {
			continue
		}
		for _, v := range entry.Values {
			if v == "" {
				continue
			}
			v = strings.ReplaceAll(v, "${natives_directory}", nativesDir)
			v = strings.ReplaceAll(v, "${classpath}", classpath)
			v = strings.ReplaceAll(v, "${launcher_name}", launcherName)
			v = strings.ReplaceAll(v, "${launcher_version}", launcherVersion)
			result = append(result, v)
		}
	}
	return result
}

func resolveGameArguments(modpack *ModpackConfig, ctx *LaunchContext) []string {
	currentOS := getCurrentOS()
	var result []string
	if modpack.Arguments == nil || modpack.Arguments.Game == nil {
		return result
	}
	for _, entry := range modpack.Arguments.Game {
		if !argumentEntryAppliesForGame(&entry, currentOS, ctx) {
			continue
		}
		for _, v := range entry.Values {
			if v == "" {
				continue
			}
			result = append(result, ctx.substitute(v))
		}
	}
	return result
}

// LaunchMinecraft выполняет полный flow: JDK → modpack → downloads → launch.
// Если onProcessStarted задан, вызывается при успешном запуске процесса (для скрытия окна и ожидания выхода).
func LaunchMinecraft(onProgress LaunchProgress, onProcessStarted LaunchProcessStarted) error {
	launcherDir, err := getLauncherDir()
	if err != nil {
		return fmt.Errorf("папка лаунчера: %w", err)
	}
	gameDir := launcherDir

	if onProgress != nil {
		onProgress("Подготовка", "Проверка JDK...", 0)
	}
	javaExe, err := EnsureJDK(launcherDir, onProgress)
	if err != nil {
		return fmt.Errorf("JDK: %w", err)
	}

	if onProgress != nil {
		onProgress("Модпак", "Загрузка modpack...", 0)
	}
	if cfg, _ := LoadConfig(); cfg == nil || cfg.ApiBaseUrl == "" {
		return fmt.Errorf("настройте URL API в настройках перед запуском")
	}
	modpack, err := LoadModpack()
	if err != nil {
		return fmt.Errorf("modpack: %w", err)
	}

	if onProgress != nil {
		onProgress("Модпак", "Проверка версии...", 0)
	}
	version, _ := FetchServerVersion()

	if onProgress != nil {
		onProgress("Загрузка модов", "Скачивание модов...", 0)
	}
	if err := EnsureMods(gameDir, version, onProgress); err != nil {
		return fmt.Errorf("моды: %w", err)
	}

	// Синхронизация settings-файлов (options.txt) при обновлении сборки
	if version != nil {
		lastModsHash := loadLastModsHash(gameDir)
		buildChanged := lastModsHash == "" || version.ModsHash != lastModsHash
		if buildChanged {
			if cfg, _ := LoadConfig(); cfg != nil && cfg.SyncClientSettings {
				if err := EnsureClientFiles(gameDir, version, onProgress); err != nil {
					return fmt.Errorf("client_files: %w", err)
				}
			}
			_ = saveLastModsHash(gameDir, version.ModsHash)
		}
	}

	if onProgress != nil {
		onProgress("Загрузка файлов", "Библиотеки и client.jar...", 0)
	}
	if err := EnsureLibraries(gameDir, modpack, onProgress); err != nil {
		return fmt.Errorf("библиотеки: %w", err)
	}
	if err := EnsureClientJar(gameDir, modpack, onProgress); err != nil {
		return fmt.Errorf("client.jar: %w", err)
	}

	if onProgress != nil {
		onProgress("Подготовка", "Извлечение natives...", 0)
	}
	_ = ExtractNatives(gameDir, modpack)

	if onProgress != nil {
		onProgress("Загрузка ассетов", "Индекс ассетов...", 0)
	}
	if err := EnsureAssetIndex(gameDir, modpack); err != nil {
		// не критично
	}
	if onProgress != nil {
		onProgress("Загрузка ассетов", "Скачивание ассетов...", 0)
	}
	if err := EnsureAssets(gameDir, modpack, onProgress); err != nil {
		// не критично
	}

	versionID := modpack.ID
	if versionID == "" {
		versionID = "modpack"
	}
	clientJar := filepath.Join(gameDir, "versions", versionID, versionID+".jar")
	if _, err := os.Stat(clientJar); err != nil {
		clientJar = filepath.Join(gameDir, "versions", versionID, "client.jar")
	}
	if _, err := os.Stat(clientJar); err != nil {
		return fmt.Errorf("client.jar не найден в versions/%s/", versionID)
	}

	currentOS := getCurrentOS()
	var classpath []string
	classpath = append(classpath, clientJar)
	for i := range modpack.Libraries {
		lib := &modpack.Libraries[i]
		if !libraryApplies(lib, currentOS) {
			continue
		}
		path := getLibraryPath(gameDir, lib)
		if path != "" {
			if _, err := os.Stat(path); err == nil {
				classpath = append(classpath, path)
			}
		}
	}

	sep := string(os.PathListSeparator)
	cp := strings.Join(classpath, sep)
	if len(classpath) <= 1 {
		return fmt.Errorf("classpath пуст")
	}

	base := strings.ReplaceAll(gameDir, "\\", "/")
	nativesPath := base + "/natives"
	assetsRoot := base + "/assets"
	assetsIndex := modpack.Assets
	if assetsIndex == "" {
		assetsIndex = "29"
	}

	session, _ := authLoadSession()
	playerName := "Player"
	authUUID := "00000000-0000-0000-0000-000000000000"
	authToken := "0"
	if session != nil && session.isValid() {
		playerName = session.Nickname
		authUUID = session.SessionUUID
		authToken = session.SessionUUID
	}

	ctx := &LaunchContext{
		GameDirectory:   base,
		AssetsRoot:      assetsRoot,
		AssetsIndexName: assetsIndex,
		AuthPlayerName:  playerName,
		VersionName:     versionID,
		AuthUUID:        authUUID,
		AuthAccessToken: authToken,
		ClientID:        "0",
		AuthXuid:        "0",
		VersionType:     "fabric",
	}

	jvmArgs := resolveJvmArguments(modpack, nativesPath, cp, "custom", "1.0")
	gameArgs := resolveGameArguments(modpack, ctx)

	// Передаём --server и --port для автоподключения (мод launcher_auto_connect читает их из аргументов)
	// Берём из конфига лаунчера, при отсутствии — из ответа API /api/version. Без дефолтов.
	serverHost, serverPort := "", ""
	if cfg, _ := LoadConfig(); cfg != nil {
		if cfg.ServerHost != "" {
			serverHost = cfg.ServerHost
		}
		if cfg.ServerPort > 0 {
			serverPort = fmt.Sprintf("%d", cfg.ServerPort)
		}
	}
	if serverHost == "" && version != nil {
		serverHost = version.ServerHost
		serverPort = version.ServerPort
	}
	if serverHost != "" {
		if serverPort == "" {
			return fmt.Errorf("server_port не задан. Укажите в настройках или задайте SERVER_PORT в .env на бэкенде")
		}
		gameArgs = append(gameArgs, "--server", serverHost, "--port", serverPort)
	}

	mainClass := modpack.MainClass
	if mainClass == "" {
		return fmt.Errorf("mainClass не указан в modpack.json")
	}

	var args []string
	args = append(args, jvmArgs...)
	args = append(args, mainClass)
	args = append(args, gameArgs...)

	if onProgress != nil {
		onProgress("Запуск", "Запуск Minecraft...", 1)
	}

	cmd := exec.Command(javaExe, args...)
	cmd.Dir = gameDir
	cmd.SysProcAttr = sysProcAttrForLaunch
	// inheritIO — дочерний процесс наследует stdin/stdout/stderr (на Windows с CREATE_NO_WINDOW консоль не показывается)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("запуск: %w", err)
	}
	if onProcessStarted != nil {
		onProcessStarted(cmd)
	}
	// Не вызываем Release() — процесс остаётся дочерним, как в Java ProcessBuilder
	return nil
}
