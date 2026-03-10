package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/browser"
)

const authSessionFilename = "launcher-auth.json"

// AuthSession — сессия аутентификации
type AuthSession struct {
	Nickname   string `json:"nickname"`
	SessionUUID string `json:"session_uuid"`
}

func (s *AuthSession) isValid() bool {
	return s != nil && strings.TrimSpace(s.Nickname) != "" && strings.TrimSpace(s.SessionUUID) != ""
}

func getAuthFilePath() (string, error) {
	dir, err := getAppConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "auth", authSessionFilename), nil
}

func authLoadSession() (*AuthSession, error) {
	path, err := getAuthFilePath()
	if err != nil {
		logError("auth", "getAuthFilePath error: %v", err)
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		logError("auth", "os.ReadFile auth session error: %v", err)
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	nickname := m["nickname"]
	sessionUUID := m["session_uuid"]
	if nickname == "" || sessionUUID == "" {
		return nil, nil
	}
	s := &AuthSession{Nickname: strings.TrimSpace(nickname), SessionUUID: strings.TrimSpace(sessionUUID)}
	if !s.isValid() {
		return nil, nil
	}
	return s, nil
}

func authSaveSession(s *AuthSession) error {
	if s == nil || !s.isValid() {
		return fmt.Errorf("невалидная сессия")
	}
	path, err := getAuthFilePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return err
	}
	data, err := json.MarshalIndent(map[string]string{
		"nickname":     s.Nickname,
		"session_uuid": s.SessionUUID,
	}, "", "  ")
	if err != nil {
		return err
	}
	logInfo("auth", "сохраняем сессию в %s", path)
	return os.WriteFile(path, data, authFileMode)
}

func authDeleteSession() error {
	path, err := getAuthFilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func getApiBaseUrl() string {
	cfg, err := LoadConfig()
	if err != nil || cfg == nil {
		return strings.TrimSuffix(buildDefaultApiBaseUrl, "/")
	}
	if cfg.ApiBaseUrl == "" {
		return strings.TrimSuffix(buildDefaultApiBaseUrl, "/")
	}
	return strings.TrimSuffix(cfg.ApiBaseUrl, "/")
}

type authInitResp struct {
	Code   string `json:"code"`
	BotURL string `json:"bot_url"`
}

type authCheckResp struct {
	Status     string `json:"status"`
	Nickname   string `json:"nickname"`
	SessionUUID string `json:"session_uuid"`
}

type authVerifyResp struct {
	Valid bool `json:"valid"`
}

func authCallInit() (*authInitResp, error) {
	base := getApiBaseUrl()
	if base == "" {
		return nil, fmt.Errorf("настройте URL API в настройках перед входом")
	}
	resp, err := doWithRetry(func() (*http.Request, error) {
		return http.NewRequest("POST", base+"/api/auth/init", nil)
	}, httpTimeoutShort)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к %s: %w. Проверьте URL бэкенда в настройках", base, err)
	}
	defer resp.Body.Close()
	var r authInitResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}
	if r.Code == "" || r.BotURL == "" {
		return nil, fmt.Errorf("пустой ответ")
	}
	return &r, nil
}

func authCallCheck(code string) (*authCheckResp, error) {
	base := getApiBaseUrl()
	if base == "" {
		return nil, fmt.Errorf("настройте URL API в настройках")
	}
	u := base + "/api/auth/check?code=" + url.QueryEscape(code)
	resp, err := getWithRetry(u, authCheckTimeout)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}
	var r authCheckResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, nil
	}
	return &r, nil
}

func authCallVerify(nickname, sessionUUID string) *bool {
	base := getApiBaseUrl()
	if base == "" {
		return nil
	}
	u := base + "/api/auth/verify?nickname=" + url.QueryEscape(nickname) + "&session_uuid=" + url.QueryEscape(sessionUUID)
	resp, err := getWithRetry(u, authVerifyTimeout)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	var r authVerifyResp
	if err := json.Unmarshal(body, &r); err != nil {
		return nil
	}
	return &r.Valid
}

func authOpenBrowser(urlStr string) {
	_ = browser.OpenURL(urlStr)
}

func authPollUntilAuthenticated(code string) (*AuthSession, error) {
	deadline := time.Now().Add(authTimeout)
	for time.Now().Before(deadline) {
		resp, err := authCallCheck(code)
		if err != nil {
			time.Sleep(authPollInterval)
			continue
		}
		if resp != nil && resp.Status == "authenticated" && resp.Nickname != "" && resp.SessionUUID != "" {
			return &AuthSession{Nickname: resp.Nickname, SessionUUID: resp.SessionUUID}, nil
		}
		time.Sleep(authPollInterval)
	}
	return nil, fmt.Errorf("время ожидания входа истекло. Попробуйте снова")
}
