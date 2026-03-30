package main

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
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

// AuthSession — сессия аутентификации (Telegram + Yggdrasil/access token)
type AuthSession struct {
	Nickname    string `json:"nickname"`
	SessionUUID string `json:"session_uuid"`

	// Поля ниже нужны для интеграции с Yggdrasil/authlib-injector.
	AccessToken string `json:"access_token,omitempty"`
	ProfileID   string `json:"profile_id,omitempty"`
	ProfileName string `json:"profile_name,omitempty"`
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
	s := &AuthSession{
		Nickname:    strings.TrimSpace(nickname),
		SessionUUID: strings.TrimSpace(sessionUUID),
		AccessToken: strings.TrimSpace(m["access_token"]),
		ProfileID:   strings.TrimSpace(m["profile_id"]),
		ProfileName: strings.TrimSpace(m["profile_name"]),
	}
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
	payload := map[string]string{
		"nickname":     s.Nickname,
		"session_uuid": s.SessionUUID,
	}
	if s.AccessToken != "" {
		payload["access_token"] = s.AccessToken
	}
	if s.ProfileID != "" {
		payload["profile_id"] = s.ProfileID
	}
	if s.ProfileName != "" {
		payload["profile_name"] = s.ProfileName
	}
	data, err := json.MarshalIndent(payload, "", "  ")
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

// yggAuthRequest/yggAuthResponse — структуры для Yggdrasil authenticate.
type yggAuthRequest struct {
	Username     string `json:"username"`
	SessionToken string `json:"sessionToken"`
	ClientToken  string `json:"clientToken"`
	RequestUser  bool   `json:"requestUser"`
}

type yggProfile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type yggAuthResponse struct {
	AccessToken     string      `json:"accessToken"`
	ClientToken     string      `json:"clientToken"`
	SelectedProfile *yggProfile `json:"selectedProfile"`
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

// generateClientToken генерирует произвольный UUID-подобный токен клиента (32 hex-символа).
// offlineUUIDNoDashes возвращает offline UUID профиля Minecraft (без дефисов), как UUID.nameUUIDFromBytes("OfflinePlayer:"+nick).
func offlineUUIDNoDashes(nickname string) string {
	name := strings.TrimSpace(nickname)
	if name == "" {
		return ""
	}
	sum := md5.Sum([]byte("OfflinePlayer:" + name))
	b := sum[:]
	b[6] = (b[6] & 0x0f) | 0x30
	b[8] = (b[8] & 0x3f) | 0x80
	return hex.EncodeToString(b)
}

func generateClientToken() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return hex.EncodeToString(buf)
}

// authYggdrasilAuthenticate выполняет POST backendURL/yggdrasil/authserver/authenticate
// и дополняет AuthSession полями accessToken/selectedProfile.
func authYggdrasilAuthenticate(s *AuthSession) error {
	if s == nil || !s.isValid() {
		return fmt.Errorf("пустая или невалидная сессия для Yggdrasil")
	}
	base := getApiBaseUrl()
	if base == "" {
		return fmt.Errorf("настройте URL API в настройках")
	}
	endpoint := strings.TrimSuffix(base, "/") + "/yggdrasil/authserver/authenticate"

	clientToken := generateClientToken()
	if clientToken == "" {
		return fmt.Errorf("не удалось сгенерировать clientToken")
	}

	reqBody := &yggAuthRequest{
		Username:     s.Nickname,
		SessionToken: s.SessionUUID,
		ClientToken:  clientToken,
		RequestUser:  false,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logError("auth", "yggdrasil authenticate error: status=%d body=%s", resp.StatusCode, string(body))
		return fmt.Errorf("ошибка Yggdrasil authenticate: %s", resp.Status)
	}

	var yggResp yggAuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&yggResp); err != nil {
		return err
	}
	if yggResp.AccessToken == "" || yggResp.SelectedProfile == nil {
		return fmt.Errorf("неполный ответ Yggdrasil authenticate")
	}

	s.AccessToken = yggResp.AccessToken
	s.ProfileID = yggResp.SelectedProfile.ID
	s.ProfileName = yggResp.SelectedProfile.Name
	return nil
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
