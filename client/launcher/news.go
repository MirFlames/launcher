package main

import (
	"encoding/json"
	"net/url"
)

// NewsFeedResponse — ответ API новостей (расширяемая структура для уведомлений о версии и т.п.)
type NewsFeedResponse struct {
	Authenticated bool      `json:"authenticated"`
	Message       string    `json:"message,omitempty"`
	News          *NewsItem `json:"news,omitempty"`
	Update        any       `json:"update,omitempty"` // зарезервировано: уведомление о новой версии
}

// NewsItem — одна новость
type NewsItem struct {
	Text      string `json:"text"`
	Link      string `json:"link,omitempty"`
	Published string `json:"published,omitempty"`
}

func fetchNewsFeed(baseURL string, session *AuthSession) (*NewsFeedResponse, error) {
	params := url.Values{}
	params.Set("launcher_version", LauncherVersion)
	if session != nil && session.isValid() {
		params.Set("nickname", session.Nickname)
		params.Set("session_uuid", session.SessionUUID)
	}

	reqURL := baseURL + "/api/news?" + params.Encode()
	resp, err := getWithRetry(reqURL, httpTimeoutShort)
	if err != nil {
		return &NewsFeedResponse{
			Authenticated: false,
			Message:       "Не удалось подключиться к серверу. Проверьте URL в настройках.",
		}, nil
	}
	defer resp.Body.Close()

	var result NewsFeedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &NewsFeedResponse{
			Authenticated: false,
			Message:       "Ошибка ответа сервера",
		}, nil
	}
	return &result, nil
}
