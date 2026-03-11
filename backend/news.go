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

// isLegacyLauncherVersion возвращает true для старых лаунчеров без автообновления.
// Считаем устаревшими все версии строго меньше 1.0.5 (включая 0.x и ранние 1.0.x).
func isLegacyLauncherVersion(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" {
		return false
	}
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return false
	}
	major, minor, patch := 0, 0, 0
	_, _ = fmt.Sscanf(parts[0], "%d", &major)
	_, _ = fmt.Sscanf(parts[1], "%d", &minor)
	if len(parts) == 3 {
		_, _ = fmt.Sscanf(parts[2], "%d", &patch)
	}
	// target = 1.0.5
	if major < 1 {
		return true
	}
	if major > 1 {
		return false
	}
	// major == 1
	if minor < 0 {
		return true
	}
	if minor > 0 {
		return false
	}
	// 1.0.x
	return patch < 5
}

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

	newsRequestsTotal.Inc()
	nickname := strings.TrimSpace(r.URL.Query().Get("nickname"))
	sessionUUID := strings.TrimSpace(r.URL.Query().Get("session_uuid"))
	launcherVersion := strings.TrimSpace(r.URL.Query().Get("launcher_version"))

	entry, ok := sessionGetByUUID(sessionUUID)
	authenticated := ok && entry.Nickname == nickname

	resp := NewsResponse{Authenticated: authenticated}

	if !authenticated {
		resp.Message = "Авторизуйся чтобы играть, кнопка выше! Сервер не хранит ваши пароли."
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Авторизован — возвращаем кэшированную новость из Telegram API (channel_post)
	// Зафиксируем версию лаунчера у игрока (для админки), если она передана.
	if launcherVersion != "" {
		sessionUpdateLauncherVersion(sessionUUID, launcherVersion)
		entry.LauncherVersion = launcherVersion
	}

	// Если это очень старая версия лаунчера (до ветки 1.x), поверх новостей показываем предупреждение.
	if isLegacyLauncherVersion(launcherVersion) {
		resp.Message = "Ваша версия лаунчера устарела! Пожалуйста, скачай новый лаунчер по ссылке: https://github.com/MirFlames/launcher/releases/latest/download/launcher.zip . После скачивания закрой лаунчер и замени launcher.exe на новый из архива. При проблемах с установкой новой версии обратись к админу."
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

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
