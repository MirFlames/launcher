package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const newsCacheFile = "news-cache.json"

// NewsResponse — ответ GET /api/news (расширяемая структура для уведомлений о версии и т.п.)
type NewsResponse struct {
	Authenticated bool        `json:"authenticated"`
	Message       string      `json:"message,omitempty"` // при !authenticated — призыв войти + про пароли
	News          *NewsItem   `json:"news,omitempty"`    // при authenticated — последняя новость
	Update        interface{} `json:"update,omitempty"`  // зарезервировано: уведомление о новой версии лаунчера
}

// NewsItem — одна новость из канала
type NewsItem struct {
	Text      string `json:"text"`
	Link      string `json:"link,omitempty"`
	Published string `json:"published,omitempty"`
}

var (
	latestChannelPost = struct {
		sync.RWMutex
		item *NewsItem
	}{}
)

// LoadNewsCache загружает кэш новостей из файла (вызывается при старте бэкенда).
func LoadNewsCache() {
	data, err := os.ReadFile(newsCacheFile)
	if err != nil {
		return
	}
	var item NewsItem
	if err := json.Unmarshal(data, &item); err != nil {
		return
	}
	if item.Text == "" {
		return
	}
	latestChannelPost.Lock()
	latestChannelPost.item = &item
	latestChannelPost.Unlock()
	log.Printf("[News] Загружена кэшированная новость из %s", newsCacheFile)
}

func saveNewsCache(item *NewsItem) error {
	if item == nil {
		return nil
	}
	data, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(newsCacheFile, data, 0644)
}

// CacheChannelPost сохраняет последний пост из канала (вызывается из telegram_bot при получении channel_post)
func CacheChannelPost(msg *tgbotapi.Message) {
	if msg == nil || msg.Chat == nil {
		return
	}
	channel := strings.TrimSpace(config.TelegramRequiredChannel)
	if channel == "" {
		return
	}
	channelNorm := strings.TrimPrefix(channel, "@")
	chatUser := strings.TrimSpace(msg.Chat.UserName)
	if !strings.EqualFold(chatUser, channelNorm) {
		return
	}

	text := msg.Text
	if text == "" && msg.Caption != "" {
		text = msg.Caption
	}
	if text == "" {
		return
	}
	text = strings.TrimSpace(text)

	link := ""
	if msg.MessageID != 0 && msg.Chat.UserName != "" {
		link = "https://t.me/" + msg.Chat.UserName + "/" + fmt.Sprint(msg.MessageID)
	}

	pub := time.Unix(int64(msg.Date), 0).Format("02.01.2006 15:04")

	item := &NewsItem{Text: text, Link: link, Published: pub}
	latestChannelPost.Lock()
	latestChannelPost.item = item
	latestChannelPost.Unlock()
	if err := saveNewsCache(item); err != nil {
		log.Printf("[News] Ошибка сохранения кэша: %v", err)
	}
	log.Printf("[News] Кэширована новость из канала @%s", msg.Chat.UserName)
}

// handleNews обрабатывает GET /api/news?nickname=X&session_uuid=Y&launcher_version=Z
// При авторизации — возвращает последнюю новость из Telegram-канала (через кэш channel_post).
// Без авторизации — сообщение про вход и что сервер не хранит пароли.
func handleNews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	nickname := strings.TrimSpace(r.URL.Query().Get("nickname"))
	sessionUUID := strings.TrimSpace(r.URL.Query().Get("session_uuid"))
	_ = r.URL.Query().Get("launcher_version") // принимаем, пока не используется (для уведомлений о версии)

	validSessions.RLock()
	entry, ok := validSessions.m[sessionUUID]
	validSessions.RUnlock()

	authenticated := ok && entry.Nickname == nickname

	resp := NewsResponse{Authenticated: authenticated}

	if !authenticated {
		resp.Message = "Авторизуйся чтобы играть, кнопка выше! Сервер не хранит ваши пароли."
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Авторизован — возвращаем кэшированную новость из Telegram API (channel_post)
	latestChannelPost.RLock()
	item := latestChannelPost.item
	latestChannelPost.RUnlock()

	if item == nil {
		resp.Message = "В канале пока нет новостей. Добавьте бота администратором в канал."
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp.News = item
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(resp)
}
