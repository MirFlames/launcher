package main

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	logInfo("app", "startup завершён, контекст инициализирован")
	// При запуске всегда выводим окно лаунчера поверх остальных.
	runtime.WindowShow(ctx)
	runtime.WindowSetAlwaysOnTop(ctx, true)
	go func() {
		time.Sleep(150 * time.Millisecond)
		runtime.WindowSetAlwaysOnTop(ctx, false)
	}()
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// GetConfig возвращает текущие настройки
func (a *App) GetConfig() (*Config, error) {
	cfg, err := LoadConfig()
	if err != nil {
		logError("config", "ошибка загрузки конфига: %v", err)
	} else {
		logInfo("config", "конфиг загружен: apiBase=%s server=%s:%d", cfg.ApiBaseUrl, cfg.ServerHost, cfg.ServerPort)
	}
	return cfg, err
}

// SaveConfig сохраняет настройки.
// Эффективные адреса всегда пересобираются из активного профиля окружения:
// профиль — источник правды, верхнеуровневые поля — его копия для остального кода.
func (a *App) SaveConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("пустой конфиг")
	}
	prevBase := ""
	if prev, err := LoadConfig(); err == nil && prev != nil {
		prevBase = normalizeBaseUrl(prev.ApiBaseUrl)
	}
	cfg.ApplyProfile(cfg.EnvName())
	logInfo("config", "сохранение конфига: env=%s apiBase=%s server=%s:%d",
		cfg.EnvName(), cfg.ApiBaseUrl, cfg.ServerHost, cfg.ServerPort)
	if err := SaveConfig(cfg); err != nil {
		logError("config", "ошибка сохранения конфига: %v", err)
		return err
	}
	// Файл сессии один на все окружения и к бэкенду не привязан. Если оставить
	// её при смене адреса, лаунчер пойдёт на новый бэкенд со старым session_uuid,
	// а при недоступном бэкенде verify промолчит и протухшая сессия сойдёт за
	// валидную. Поэтому при смене бэкенда — разлогин.
	if newBase := normalizeBaseUrl(cfg.ApiBaseUrl); newBase != prevBase {
		logInfo("auth", "сменился бэкенд (%q → %q) — сбрасываем локальную сессию", prevBase, newBase)
		if err := authDeleteSession(); err != nil {
			logError("auth", "ошибка сброса сессии при смене бэкенда: %v", err)
		}
	}
	return nil
}

// GetBuildDefaults возвращает адреса, с которыми собран лаунчер (ldflags из .env).
// Нужны настройкам, чтобы показывать их как значения прод-профиля по умолчанию.
func (a *App) GetBuildDefaults() EnvProfile {
	return buildDefaultProfile()
}

// AuthIsAuthenticated возвращает true если есть валидная сессия
func (a *App) AuthIsAuthenticated() (bool, error) {
	s, err := authLoadSession()
	if err != nil || s == nil {
		logError("auth", "ошибка загрузки сессии: %v", err)
		return false, err
	}
	valid := authCallVerify(s.Nickname, s.SessionUUID)
	if valid != nil && !*valid {
		logInfo("auth", "сессия невалидна по verify, удаляем локальную")
		_ = authDeleteSession()
		return false, nil
	}
	return true, nil
}

// AuthGetSession возвращает текущую сессию (nickname, session_uuid)
func (a *App) AuthGetSession() (*AuthSession, error) {
	return authLoadSession()
}

// AuthRefreshSession загружает сессию из файла и проверяет на backend
func (a *App) AuthRefreshSession() (*AuthSession, error) {
	s, err := authLoadSession()
	if err != nil || s == nil {
		logError("auth", "ошибка загрузки сессии: %v", err)
		return nil, err
	}
	valid := authCallVerify(s.Nickname, s.SessionUUID)
	if valid != nil && !*valid {
		logInfo("auth", "сессия стала невалидной по verify, удаляем локальную")
		_ = authDeleteSession()
		return nil, nil
	}
	return s, nil
}

// AuthStartLogin запускает процесс входа: init → открытие бота → polling. Блокирует до завершения или таймаута.
func (a *App) AuthStartLogin() (*AuthSession, error) {
	logInfo("auth", "запуск процесса входа через Telegram")
	initResp, err := authCallInit()
	if err != nil {
		logError("auth", "ошибка authCallInit: %v", err)
		return nil, err
	}
	authOpenBrowser(initResp.BotURL)
	session, err := authPollUntilAuthenticated(initResp.Code)
	if err != nil {
		logError("auth", "ошибка ожидания аутентификации: %v", err)
		return nil, err
	}
	// После получения session_uuid вызываем Yggdrasil authenticate и
	// сохраняем accessToken/профиль в launcher-auth.json.
	if err := authYggdrasilAuthenticate(session); err != nil {
		logError("auth", "ошибка Yggdrasil authenticate: %v", err)
		// при ошибке не сохраняем сессию
		return nil, err
	}
	if err := authSaveSession(session); err != nil {
		logError("auth", "ошибка сохранения сессии: %v", err)
		return nil, err
	}
	return session, nil
}

// AuthLogout удаляет сохранённую сессию
func (a *App) AuthLogout() error {
	logInfo("auth", "выход из аккаунта, удаляем сессию")
	if err := authDeleteSession(); err != nil {
		logError("auth", "ошибка удаления сессии: %v", err)
		return err
	}
	return nil
}

// PlayMinecraft выполняет flow: JDK → modpack → downloads → launch.
// Прогресс отправляется через событие "launch-progress" (stage, status, progress).
// При успешном запуске Java-процесса окно скрывается; при выходе — показывается снова.
func (a *App) PlayMinecraft() error {
	logInfo("launch", "PlayMinecraft вызван")
	onProgress := func(stage, status string, progress float64) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
				"stage":    stage,
				"status":   status,
				"progress": progress,
			})
		}
	}
	onProcessStarted := func(cmd *exec.Cmd) {
		if a.ctx == nil {
			return
		}
		hide := func() { runtime.WindowHide(a.ctx) }
		show := func() {
			runtime.EventsEmit(a.ctx, "launch-ended", nil)
			runtime.WindowSetAlwaysOnTop(a.ctx, true)
			runtime.WindowShow(a.ctx)
			go func() {
				time.Sleep(150 * time.Millisecond)
				runtime.WindowSetAlwaysOnTop(a.ctx, false)
			}()
		}
		onWaiting := func() {
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "launch-progress", map[string]interface{}{
					"stage":    "Запуск",
					"status":   "Скоро на завод...",
					"progress": 1,
				})
			}
		}
		go WaitForProcessWindow(cmd, hide, show, onWaiting)
	}
	if err := LaunchMinecraft(onProgress, onProcessStarted); err != nil {
		logError("launch", "ошибка LaunchMinecraft: %v", err)
		return err
	}
	logInfo("launch", "LaunchMinecraft завершился без ошибок (процесс Java запущен)")
	return nil
}

// GetNewsFeed запрашивает новости у бэкенда (с проверкой сессии).
func (a *App) GetNewsFeed() (*NewsFeedResponse, error) {
	base := getApiBaseUrl()
	session, _ := authLoadSession()
	resp, err := fetchNewsFeed(base, session)
	if err != nil {
		logError("news", "ошибка загрузки новостей: %v", err)
	} else {
		logInfo("news", "новости загружены: authenticated=%v message=%q", resp.Authenticated, resp.Message)
	}
	return resp, err
}

// GetLauncherVersion возвращает текущую версию лаунчера для отображения в UI.
func (a *App) GetLauncherVersion() string {
	return LauncherVersion
}

