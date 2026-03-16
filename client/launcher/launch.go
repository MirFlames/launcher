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

// evaluateRules возвращает true если правила разрешают применение.
// emptyOSMatchesAll: при true правило без OS считается совпавшим (для JVM args)
func evaluateRules(rules []ModpackRule, currentPlatform string, emptyOSMatchesAll bool, featureChecker func(*ModpackRuleFeatures) bool) bool {
	if len(rules) == 0 {
		return true
	}
	for _, r := range rules {
		osMatch := false
		if r.OS == nil || r.OS.Name == "" {
			osMatch = emptyOSMatchesAll
		} else {
			osMatch = strings.EqualFold(r.OS.Name, currentPlatform)
		}
		if osMatch {
			if strings.EqualFold(r.Action, "allow") {
				return true
			}
			if strings.EqualFold(r.Action, "disallow") {
				return false
			}
		}
		if r.Features != nil && featureChecker != nil && featureChecker(r.Features) {
			if strings.EqualFold(r.Action, "allow") {
				return true
			}
			if strings.EqualFold(r.Action, "disallow") {
				return false
			}
		}
	}
	return false
}

func jvmArgumentApplies(entry *ModpackArgumentEntry, currentPlatform string) bool {
	if entry == nil || entry.Values == nil {
		return false
	}
	return evaluateRules(entry.Rules, currentPlatform, true, nil)
}

func gameArgumentApplies(entry *ModpackArgumentEntry, currentPlatform string, ctx *LaunchContext) bool {
	if entry == nil || entry.Values == nil {
		return false
	}
	return evaluateRules(entry.Rules, currentPlatform, false, func(f *ModpackRuleFeatures) bool {
		return featureMatches(ctx, f)
	})
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

func resolveJvmArguments(modpack *ModpackConfig, nativesDir, classpath, launcherNameVal, launcherVersion string) []string {
	currentPlatform := getCurrentPlatform()
	var result []string
	if modpack.Arguments == nil || modpack.Arguments.JVM == nil {
		return result
	}
	for _, entry := range modpack.Arguments.JVM {
		if !jvmArgumentApplies(&entry, currentPlatform) {
			continue
		}
		for _, v := range entry.Values {
			if v == "" {
				continue
			}
			v = strings.ReplaceAll(v, "${natives_directory}", nativesDir)
			v = strings.ReplaceAll(v, "${classpath}", classpath)
			v = strings.ReplaceAll(v, "${launcher_name}", launcherNameVal)
			v = strings.ReplaceAll(v, "${launcher_version}", launcherVersion)
			result = append(result, v)
		}
	}
	return result
}

func resolveGameArguments(modpack *ModpackConfig, ctx *LaunchContext) []string {
	currentPlatform := getCurrentPlatform()
	var result []string
	if modpack.Arguments == nil || modpack.Arguments.Game == nil {
		return result
	}
	for _, entry := range modpack.Arguments.Game {
		if !gameArgumentApplies(&entry, currentPlatform, ctx) {
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

func ensurePrerequisites(launcherDir string, onProgress LaunchProgress) (javaExe string, cfg *Config, modpack *ModpackConfig, version *ServerVersion, err error) {
	logInfo("launch", "ensurePrerequisites: launcherDir=%s", launcherDir)
	if onProgress != nil {
		onProgress("Подготовка", "Проверка JDK...", 0)
	}
	javaExe, err = EnsureJDK(launcherDir, onProgress)
	if err != nil {
		logError("launch", "EnsureJDK error: %v", err)
		return "", nil, nil, nil, fmt.Errorf("JDK: %w", err)
	}

	if onProgress != nil {
		onProgress("Модпак", "Загрузка modpack...", 0)
	}
	cfg, _ = LoadConfig()
	if cfg == nil || cfg.ApiBaseUrl == "" {
		return "", nil, nil, nil, fmt.Errorf("настройте URL API в настройках перед запуском")
	}
	logInfo("launch", "конфиг загружен в ensurePrerequisites: apiBase=%s server=%s:%d", cfg.ApiBaseUrl, cfg.ServerHost, cfg.ServerPort)

	modpack, err = LoadModpack()
	if err != nil {
		logError("launch", "LoadModpack error: %v", err)
		return "", nil, nil, nil, fmt.Errorf("modpack: %w", err)
	}

	if onProgress != nil {
		onProgress("Модпак", "Проверка версии...", 0)
	}
	version, _ = FetchServerVersion()
	if version != nil {
		logInfo("launch", "FetchServerVersion: mc=%s host=%s port=%s mods=%d", version.MinecraftVersion, version.ServerHost, version.ServerPort, len(version.Mods))
	} else {
		logInfo("launch", "FetchServerVersion: версия сервера не получена (nil)")
	}
	return javaExe, cfg, modpack, version, nil
}

func ensureGameFiles(gameDir string, version *ServerVersion, modpack *ModpackConfig, cfg *Config, onProgress LaunchProgress) error {
	logInfo("launch", "ensureGameFiles: gameDir=%s", gameDir)

	serverHost, _ := resolveServerConnection(cfg, version)
	skipModSync := serverHost == "127.0.0.1" || serverHost == "localhost"

	var modsDownloaded bool
	if !skipModSync {
		if onProgress != nil {
			onProgress("Загрузка модов", "Скачивание модов...", 0)
		}
		var err error
		modsDownloaded, err = EnsureMods(gameDir, version, onProgress)
		if err != nil {
			logError("launch", "EnsureMods error: %v", err)
			return fmt.Errorf("моды: %w", err)
		}

		if onProgress != nil && version != nil && len(version.ConfigFiles) > 0 {
			onProgress("Загрузка конфигов модов", "Скачивание конфигов...", 0)
		}
		if err := EnsureModConfigs(gameDir, version, onProgress); err != nil {
			logError("launch", "EnsureModConfigs error: %v", err)
			return fmt.Errorf("конфиги модов: %w", err)
		}
	} else {
		logInfo("launch", "пропуск синхронизации модов: сервер %s (локальный)", serverHost)
	}

	if version != nil && modsDownloaded && cfg != nil && cfg.SyncClientSettings {
		if err := EnsureClientFiles(gameDir, version, onProgress); err != nil {
			logError("launch", "EnsureClientFiles error: %v", err)
			return fmt.Errorf("client_files: %w", err)
		}
	}

	if onProgress != nil {
		onProgress("Загрузка файлов", "Библиотеки и client.jar...", 0)
	}
	if err := EnsureLibraries(gameDir, modpack, onProgress); err != nil {
		logError("launch", "EnsureLibraries error: %v", err)
		return fmt.Errorf("библиотеки: %w", err)
	}
	if err := EnsureClientJar(gameDir, modpack, onProgress); err != nil {
		logError("launch", "EnsureClientJar error: %v", err)
		return fmt.Errorf("client.jar: %w", err)
	}

	if onProgress != nil {
		onProgress("Подготовка", "Извлечение natives...", 0)
	}
	_ = ExtractNatives(gameDir, modpack)

	if onProgress != nil {
		onProgress("Загрузка ассетов", "Индекс ассетов...", 0)
	}
	_ = EnsureAssetIndex(gameDir, modpack)
	if onProgress != nil {
		onProgress("Загрузка ассетов", "Скачивание ассетов...", 0)
	}
	_ = EnsureAssets(gameDir, modpack, onProgress)
	return nil
}

func findClientJar(gameDir, versionID string) (string, error) {
	clientJar := filepath.Join(gameDir, "versions", versionID, versionID+".jar")
	if _, err := os.Stat(clientJar); err != nil {
		clientJar = filepath.Join(gameDir, "versions", versionID, "client.jar")
	}
	if _, err := os.Stat(clientJar); err != nil {
		return "", fmt.Errorf("client.jar не найден в versions/%s/", versionID)
	}
	return clientJar, nil
}

func buildClasspath(gameDir string, modpack *ModpackConfig, clientJar string) (string, error) {
	currentPlatform := getCurrentPlatform()
	var classpath []string
	classpath = append(classpath, clientJar)
	for i := range modpack.Libraries {
		lib := &modpack.Libraries[i]
		if !libraryApplies(lib, currentPlatform) {
			continue
		}
		path := getLibraryPath(gameDir, lib)
		if path != "" {
			if _, err := os.Stat(path); err == nil {
				classpath = append(classpath, path)
			}
		}
	}
	if len(classpath) <= 1 {
		return "", fmt.Errorf("classpath пуст")
	}
	return strings.Join(classpath, string(os.PathListSeparator)), nil
}

func buildLaunchContext(session *AuthSession, versionID, gameDir string) *LaunchContext {
	base := strings.ReplaceAll(gameDir, "\\", "/")
	assetsIndex := defaultAssetsIndex
	// assetsIndex из modpack не передаём сюда — используем дефолт, т.к. modpack не в сигнатуре
	playerName := "Player"
	authUUID := "00000000-0000-0000-0000-000000000000"
	authToken := "0"
	if session != nil && session.isValid() {
		playerName = session.Nickname
		authUUID = session.SessionUUID
		authToken = session.SessionUUID
	}
	return &LaunchContext{
		GameDirectory:   base,
		AssetsRoot:      base + "/assets",
		AssetsIndexName: assetsIndex,
		AuthPlayerName:  playerName,
		VersionName:     versionID,
		AuthUUID:        authUUID,
		AuthAccessToken: authToken,
		ClientID:        "0",
		AuthXuid:        "0",
		VersionType:     defaultVersionType,
	}
}

func resolveServerConnection(cfg *Config, version *ServerVersion) (host, port string) {
	if cfg != nil {
		if cfg.ServerHost != "" {
			host = cfg.ServerHost
		}
		if cfg.ServerPort > 0 {
			port = fmt.Sprintf("%d", cfg.ServerPort)
		}
	}
	if host == "" && version != nil {
		host = version.ServerHost
		port = version.ServerPort
	}
	if host == "" && buildDefaultServerHost != "" {
		host = buildDefaultServerHost
		port = buildDefaultServerPort
	}
	return host, port
}

func spawnMinecraftProcess(javaExe, gameDir string, jvmArgs []string, mainClass string, gameArgs []string, onProgress LaunchProgress, onProcessStarted LaunchProcessStarted) error {
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
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("запуск: %w", err)
	}
	if onProcessStarted != nil {
		onProcessStarted(cmd)
	}
	return nil
}

// LaunchMinecraft выполняет полный flow: JDK → modpack → downloads → launch.
// Если onProcessStarted задан, вызывается при успешном запуске процесса (для скрытия окна и ожидания выхода).
func LaunchMinecraft(onProgress LaunchProgress, onProcessStarted LaunchProcessStarted) error {
	launcherDir, err := getLauncherDir()
	if err != nil {
		return fmt.Errorf("папка лаунчера: %w", err)
	}
	logInfo("launch", "LaunchMinecraft: launcherDir=%s", launcherDir)

	javaExe, cfg, modpack, version, err := ensurePrerequisites(launcherDir, onProgress)
	if err != nil {
		return err
	}

	if err := ensureGameFiles(launcherDir, version, modpack, cfg, onProgress); err != nil {
		return err
	}

	versionID := modpack.ID
	if versionID == "" {
		versionID = "modpack"
	}
	clientJar, err := findClientJar(launcherDir, versionID)
	if err != nil {
		return err
	}

	cp, err := buildClasspath(launcherDir, modpack, clientJar)
	if err != nil {
		return err
	}

	base := strings.ReplaceAll(launcherDir, "\\", "/")
	nativesPath := base + "/natives"
	assetsIndex := modpack.Assets
	if assetsIndex == "" {
		assetsIndex = defaultAssetsIndex
	}

	session, _ := authLoadSession()
	ctx := buildLaunchContext(session, versionID, launcherDir)
	ctx.AssetsIndexName = assetsIndex

	jvmArgs := resolveJvmArguments(modpack, nativesPath, cp, launcherName, launcherVersionFallback)
	gameArgs := resolveGameArguments(modpack, ctx)

	serverHost, serverPort := resolveServerConnection(cfg, version)
	if serverHost != "" {
		if serverPort == "" {
			return fmt.Errorf("server_port не задан. Укажите в настройках или задайте SERVER_PORT в .env при сборке")
		}
		gameArgs = append(gameArgs, "--server", serverHost, "--port", serverPort)
	}

	backendURL := getApiBaseUrl()
	if backendURL != "" {
		gameArgs = append(gameArgs, "--backend-url", backendURL)
		jvmArgs = append(jvmArgs, "-Dminecraft.api.session.host="+backendURL)
		logInfo("launch", "добавлен аргумент запуска: --backend-url=%s", backendURL)
	}

	gameArgs = append(gameArgs, "--session-uuid", ctx.AuthUUID)

	mainClass := modpack.MainClass
	if mainClass == "" {
		return fmt.Errorf("mainClass не указан в modpack.json")
	}

	err = spawnMinecraftProcess(javaExe, launcherDir, jvmArgs, mainClass, gameArgs, onProgress, onProcessStarted)
	if err != nil {
		logError("launch", "spawnMinecraftProcess error: %v", err)
		return err
	}
	logInfo("launch", "spawnMinecraftProcess успешно запустил Java-процесс")
	return nil
}
