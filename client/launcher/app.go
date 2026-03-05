package main

import (
	"context"
	"fmt"
	"os/exec"

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
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// GetConfig возвращает текущие настройки
func (a *App) GetConfig() (*Config, error) {
	return LoadConfig()
}

// SaveConfig сохраняет настройки
func (a *App) SaveConfig(cfg *Config) error {
	return SaveConfig(cfg)
}

// AuthIsAuthenticated возвращает true если есть валидная сессия
func (a *App) AuthIsAuthenticated() (bool, error) {
	s, err := authLoadSession()
	if err != nil || s == nil {
		return false, err
	}
	valid := authCallVerify(s.Nickname, s.SessionUUID)
	if valid != nil && !*valid {
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
		return nil, err
	}
	valid := authCallVerify(s.Nickname, s.SessionUUID)
	if valid != nil && !*valid {
		_ = authDeleteSession()
		return nil, nil
	}
	return s, nil
}

// AuthStartLogin запускает процесс входа: init → открытие бота → polling. Блокирует до завершения или таймаута.
func (a *App) AuthStartLogin() (*AuthSession, error) {
	initResp, err := authCallInit()
	if err != nil {
		return nil, err
	}
	authOpenBrowser(initResp.BotURL)
	session, err := authPollUntilAuthenticated(initResp.Code)
	if err != nil {
		return nil, err
	}
	if err := authSaveSession(session); err != nil {
		return nil, err
	}
	return session, nil
}

// AuthLogout удаляет сохранённую сессию
func (a *App) AuthLogout() error {
	return authDeleteSession()
}

// PlayMinecraft выполняет flow: JDK → modpack → downloads → launch.
// Прогресс отправляется через событие "launch-progress" (stage, status, progress).
// При успешном запуске Java-процесса окно скрывается; при выходе — показывается снова.
func (a *App) PlayMinecraft() error {
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
			runtime.WindowShow(a.ctx)
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
	return LaunchMinecraft(onProgress, onProcessStarted)
}

// GetNewsFeed запрашивает новости у бэкенда (с проверкой сессии).
func (a *App) GetNewsFeed() (*NewsFeedResponse, error) {
	base := getApiBaseUrl()
	session, _ := authLoadSession()
	return fetchNewsFeed(base, session)
}
