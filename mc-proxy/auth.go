package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

const sessionsSyncInterval = 30 * time.Second

var (
	sessionsMu   sync.RWMutex
	sessionsMap  map[string]string // session_uuid -> nickname
)

type sessionExportEntry struct {
	Nickname    string `json:"nickname"`
	SessionUUID string `json:"session_uuid"`
}

func initSessionsDB() {
	apiURL := getEnv("SESSIONS_API_URL", "")
	token := getEnv("SESSIONS_API_TOKEN", "")

	if apiURL == "" || token == "" {
		log.Fatalf("[Auth] SESSIONS_API_URL и SESSIONS_API_TOKEN обязательны для репликации сессий (mc-proxy на отдельном хосте)")
	}

	sessionsMap = make(map[string]string)
	if err := fetchSessions(apiURL, token); err != nil {
		log.Fatalf("[Auth] Не удалось загрузить сессии при старте: %v", err)
	}
	log.Printf("[Auth] Sessions: репликация с %s (%d сессий)", apiURL, len(sessionsMap))

	go syncSessions(apiURL, token)
}

func fetchSessions(apiURL, token string) error {
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Sessions-Token", token)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &httpError{status: resp.StatusCode, msg: resp.Status + ": " + string(body)}
	}

	var entries []sessionExportEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return err
	}

	newMap := make(map[string]string, len(entries))
	for _, e := range entries {
		if e.SessionUUID != "" && e.Nickname != "" {
			newMap[e.SessionUUID] = e.Nickname
		}
	}

	sessionsMu.Lock()
	sessionsMap = newMap
	sessionsMu.Unlock()
	return nil
}

type httpError struct {
	status int
	msg    string
}

func (e *httpError) Error() string {
	return e.msg
}

func syncSessions(apiURL, token string) {
	ticker := time.NewTicker(sessionsSyncInterval)
	defer ticker.Stop()
	for range ticker.C {
		if err := fetchSessions(apiURL, token); err != nil {
			log.Printf("[Auth] sync sessions: %v", err)
			continue
		}
	}
}

func sessionVerify(nickname, sessionUUID string) bool {
	if nickname == "" || sessionUUID == "" {
		return false
	}
	sessionsMu.RLock()
	dbNickname, ok := sessionsMap[sessionUUID]
	sessionsMu.RUnlock()
	return ok && dbNickname == nickname
}
