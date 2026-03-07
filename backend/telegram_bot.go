package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// pendingNickname — пользователь ожидает ввода никнейма
type pendingNickname struct {
	code      string
	createdAt time.Time
}

// pendingSubscription — пользователь ожидает проверки подписки на канал
type pendingSubscription struct {
	code      string
	createdAt time.Time
}

var (
	nicknamePending = struct {
		sync.RWMutex
		m map[int64]pendingNickname
	}{m: make(map[int64]pendingNickname)}
	subscriptionPending = struct {
		sync.RWMutex
		m map[int64]pendingSubscription
	}{m: make(map[int64]pendingSubscription)}
)

const nicknamePendingTTL = 5 * time.Minute
const subscriptionPendingTTL = 10 * time.Minute

const serverStatusButtonText = "А сервер-то работает?"
const notificationsButtonText = "Настройка уведомлений"

// Minecraft nickname: 3-16 символов, буквы, цифры, подчёркивание
var nicknameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,16}$`)

func isValidNickname(s string) bool {
	return nicknameRegex.MatchString(strings.TrimSpace(s))
}

// serverStatusKeyboard — постоянная клавиатура с кнопками (всегда видна внизу чата)
func serverStatusKeyboard() tgbotapi.ReplyKeyboardMarkup {
	k := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(serverStatusButtonText)),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(notificationsButtonText)),
	)
	k.ResizeKeyboard = true
	return k
}

// getServerStatusMessage возвращает текст в зависимости от статуса Minecraft-сервера
func getServerStatusMessage(status MinecraftStatusResult) string {
	if !status.Online {
		return "Неа. Кажись упал."
	}
	switch {
	case status.Count == 0:
		return "Да, сейчас работает! Но все игроки разбрелись: кто на работе, кто в дурке. Заходи, будь первым!"
	case status.Count == 1:
		return "Да, работает! И там кто-то сейчас играет! Подключайся!"
	default:
		return fmt.Sprintf("Работает! Онлайн на сервере: %d чел. Будешь %d-м?", status.Count, status.Count+1)
	}
}

// getRequiredChannel возвращает канал для проверки подписки (пустая строка = проверка отключена)
func getRequiredChannel() string {
	return strings.TrimSpace(config.TelegramRequiredChannel)
}

// checkSubscription проверяет, подписан ли пользователь на требуемый канал.
// Бот должен быть администратором канала. Возвращает true если подписан или проверка отключена.
func checkSubscription(bot *tgbotapi.BotAPI, userID int64) bool {
	channel := getRequiredChannel()
	if channel == "" {
		return true
	}
	if !strings.HasPrefix(channel, "@") {
		channel = "@" + channel
	}
	req := tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{SuperGroupUsername: channel, UserID: userID},
	}
	member, err := bot.GetChatMember(req)
	if err != nil {
		log.Printf("[Telegram] Ошибка проверки подписки user_id=%d: %v", userID, err)
		return false
	}
	status := member.Status
	return status == "member" || status == "administrator" || status == "creator"
}

// sendSubscriptionRequired отправляет сообщение с кнопками подписки и проверки
func sendSubscriptionRequired(bot *tgbotapi.BotAPI, chatID int64, code string) {
	channel := getRequiredChannel()
	if channel == "" {
		return
	}
	if !strings.HasPrefix(channel, "@") {
		channel = "@" + channel
	}
	msg := tgbotapi.NewMessage(chatID, "Для входа необходимо подписаться на канал.\n\nПодпишитесь и нажмите «Проверить подписку»:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Подписаться", "https://t.me/"+strings.TrimPrefix(channel, "@")),
			tgbotapi.NewInlineKeyboardButtonData("Проверить подписку", "check_sub"),
		),
	)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("[Telegram] Ошибка отправки сообщения: %v", err)
	}
}

// proceedAfterSubscription выполняет сценарий после успешной проверки подписки (сессия или никнейм)
func proceedAfterSubscription(bot *tgbotapi.BotAPI, chatID int64, userID int64, code, telegramUsername string) {
	if stored, ok := getStoredEntryByTelegramID(userID); ok && stored.Nickname != "" {
		if err := completeAuth(code, stored.Nickname, telegramUsername, userID); err != nil {
			log.Printf("[Telegram] Ошибка completeAuth (возврат) для user_id=%d: %v", userID, err)
			sendMessage(bot, chatID, "Ошибка при входе. Попробуйте снова.")
		} else {
			log.Printf("[Telegram] Возврат: user_id=%d, nickname=%s, code=%s", userID, stored.Nickname, code)
			sendMessage(bot, chatID, "С возвращением! Вернитесь в лаунчер — кнопка изменится на «Играть».")
		}
		return
	}
	nicknamePending.Lock()
	nicknamePending.m[userID] = pendingNickname{code: code, createdAt: time.Now()}
	nicknamePending.Unlock()
	log.Printf("[Telegram] Ожидание никнейма от user_id=%d для code=%s", userID, code)
	sendMessage(bot, chatID, "Введите ваш никнейм для Minecraft (3–16 символов, латиница, цифры, подчёркивание):")
}

// StartTelegramBot запускает бота в фоне. Если токен пуст — не запускает.
func StartTelegramBot() {
	if config.TelegramBotToken == "" {
		log.Printf("[Telegram] Токен бота не задан, запуск пропущен")
		return
	}

	bot, err := tgbotapi.NewBotAPI(config.TelegramBotToken)
	if err != nil {
		log.Printf("[Telegram] Ошибка инициализации бота: %v", err)
		return
	}

	bot.Debug = false
	log.Printf("[Telegram] Бот авторизован: @%s", bot.Self.UserName)

	go startNotificationWorker(bot)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	u.AllowedUpdates = []string{"message", "edited_message", "channel_post", "edited_channel_post", "callback_query"}

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		// Кэширование постов из канала для /api/news (Telegram API)
		if post := update.ChannelPost; post != nil {
			CacheChannelPost(post)
			continue
		}
		if post := update.EditedChannelPost; post != nil {
			CacheChannelPost(post)
			continue
		}

		// Обработка callback (нажатие кнопок)
		if update.CallbackQuery != nil {
			cb := update.CallbackQuery
			userID := cb.From.ID
			chatID := cb.Message.Chat.ID

			if cb.Data == "server_status" {
				status := CheckMinecraftServerStatus()
				m := tgbotapi.NewMessage(chatID, getServerStatusMessage(status))
				m.ReplyMarkup = serverStatusKeyboard()
				answerCallback(bot, cb.ID, "")
				if _, err := bot.Send(m); err != nil {
					log.Printf("[Telegram] Ошибка отправки сообщения: %v", err)
				}
				continue
			}

			if strings.HasPrefix(cb.Data, "notif_thresh_") {
				handleNotificationThresholdCallback(bot, cb, userID, chatID)
				continue
			}

			if cb.Data == "check_sub" {
				subscriptionPending.RLock()
				pending, ok := subscriptionPending.m[userID]
				subscriptionPending.RUnlock()

				if ok {
					if time.Since(pending.createdAt) > subscriptionPendingTTL {
						subscriptionPending.Lock()
						delete(subscriptionPending.m, userID)
						subscriptionPending.Unlock()
						answerCallback(bot, cb.ID, "Время ожидания истекло. Начните заново.")
						sendMessage(bot, chatID, "Время ожидания истекло. Нажмите «Войти» в лаунчере и попробуйте снова.")
						continue
					}

					if checkSubscription(bot, userID) {
						subscriptionPending.Lock()
						delete(subscriptionPending.m, userID)
						subscriptionPending.Unlock()
						answerCallback(bot, cb.ID, "Подписка подтверждена!")
						telegramUsername := ""
						if cb.From.UserName != "" {
							telegramUsername = cb.From.UserName
						}
						proceedAfterSubscription(bot, chatID, userID, pending.code, telegramUsername)
					} else {
						answerCallback(bot, cb.ID, "Подписка не обнаружена")
						sendMessage(bot, chatID, "Подписка не обнаружена. Подпишитесь на канал и нажмите «Проверить подписку» снова.")
						sendSubscriptionRequired(bot, chatID, pending.code)
					}
				} else {
					answerCallback(bot, cb.ID, "")
				}
			}
			continue
		}

		if update.Message == nil {
			continue
		}

		msg := update.Message
		chatID := msg.Chat.ID
		userID := msg.From.ID
		text := strings.TrimSpace(msg.Text)

		log.Printf("[Telegram] Сообщение от %d (@%s): %s", userID, msg.From.UserName, text)

		// Кнопка «А сервер-то работает?» — обрабатываем в любом состоянии
		if text == serverStatusButtonText {
			status := CheckMinecraftServerStatus()
			m := tgbotapi.NewMessage(chatID, getServerStatusMessage(status))
			m.ReplyMarkup = serverStatusKeyboard()
			if _, err := bot.Send(m); err != nil {
				log.Printf("[Telegram] Ошибка отправки сообщения: %v", err)
			}
			continue
		}

		// Кнопка «Настройка уведомлений» — настройки уведомлений
		if text == notificationsButtonText {
			handleNotificationsSettings(bot, chatID, userID)
			continue
		}

		// Проверяем, ожидает ли пользователь проверки подписки
		subscriptionPending.RLock()
		subPending, subPendingOk := subscriptionPending.m[userID]
		subscriptionPending.RUnlock()

		if subPendingOk {
			if time.Since(subPending.createdAt) > subscriptionPendingTTL {
				subscriptionPending.Lock()
				delete(subscriptionPending.m, userID)
				subscriptionPending.Unlock()
				sendMessage(bot, chatID, "Время ожидания истекло. Нажмите «Войти» в лаунчере и попробуйте снова.")
				continue
			}
			if checkSubscription(bot, userID) {
				subscriptionPending.Lock()
				delete(subscriptionPending.m, userID)
				subscriptionPending.Unlock()
				telegramUsername := ""
				if msg.From != nil && msg.From.UserName != "" {
					telegramUsername = msg.From.UserName
				}
				proceedAfterSubscription(bot, chatID, userID, subPending.code, telegramUsername)
			} else {
				sendMessage(bot, chatID, "Подписка не обнаружена. Подпишитесь на канал и нажмите «Проверить подписку» или отправьте любое сообщение.")
				sendSubscriptionRequired(bot, chatID, subPending.code)
			}
			continue
		}

		// Проверяем, ожидаем ли мы никнейм от этого пользователя
		nicknamePending.RLock()
		pending, pendingOk := nicknamePending.m[userID]
		nicknamePending.RUnlock()

		if pendingOk {
			if time.Since(pending.createdAt) > nicknamePendingTTL {
				nicknamePending.Lock()
				delete(nicknamePending.m, userID)
				nicknamePending.Unlock()
				sendMessage(bot, chatID, "Время ожидания истекло. Начните заново: нажмите «Войти» в лаунчере.")
				log.Printf("[Telegram] Истёк TTL для user_id=%d", userID)
				continue
			}

			if !isValidNickname(text) {
				sendMessage(bot, chatID, "Никнейм должен быть 3–16 символов (латиница, цифры, подчёркивание). Попробуйте снова:")
				continue
			}

			nickname := strings.TrimSpace(text)
			telegramUsername := ""
			if msg.From != nil && msg.From.UserName != "" {
				telegramUsername = msg.From.UserName
			}

			if err := completeAuth(pending.code, nickname, telegramUsername, userID); err != nil {
				log.Printf("[Telegram] Ошибка completeAuth для user_id=%d, code=%s: %v", userID, pending.code, err)
				sendMessage(bot, chatID, "Ошибка при входе. Код мог истечь. Нажмите «Войти» в лаунчере и попробуйте снова.")
			} else {
				log.Printf("[Telegram] Успешный вход: user_id=%d, nickname=%s, code=%s", userID, nickname, pending.code)
				sendMessage(bot, chatID, "Вы успешно вошли! Вернитесь в лаунчер — кнопка изменится на «Играть».")
			}

			nicknamePending.Lock()
			delete(nicknamePending.m, userID)
			nicknamePending.Unlock()
			continue
		}

		// Обработка /start
		if !msg.IsCommand() {
			m := tgbotapi.NewMessage(chatID, "Введите /start с кодом из лаунчера. Например: /start abc123")
			m.ReplyMarkup = serverStatusKeyboard()
			if _, err := bot.Send(m); err != nil {
				log.Printf("[Telegram] Ошибка отправки сообщения: %v", err)
			}
			continue
		}

		cmd := msg.Command()
		if cmd != "start" {
			sendMessage(bot, chatID, "Используйте /start с кодом из лаунчера.")
			continue
		}

		code := strings.TrimSpace(msg.CommandArguments())
		if code == "" {
			m := tgbotapi.NewMessage(chatID, "Нажмите «Войти» в лаунчере — откроется ссылка с кодом. Перейдите по ней и введите код.")
			m.ReplyMarkup = serverStatusKeyboard()
			if _, err := bot.Send(m); err != nil {
				log.Printf("[Telegram] Ошибка отправки сообщения: %v", err)
			}
			continue
		}

		// Проверяем, что код существует
		authStore.RLock()
		s, ok := authStore.sessions[code]
		authStore.RUnlock()

		if !ok {
			log.Printf("[Telegram] Неизвестный код от user_id=%d: %s", userID, code)
			sendMessage(bot, chatID, "Код не найден или истёк. Нажмите «Войти» в лаунчере и перейдите по новой ссылке.")
			continue
		}

		if time.Since(s.createdAt) > authCodeTTL {
			authStore.Lock()
			delete(authStore.sessions, code)
			authStore.Unlock()
			log.Printf("[Telegram] Истёкший код от user_id=%d: %s", userID, code)
			sendMessage(bot, chatID, "Код истёк. Нажмите «Войти» в лаунчере и попробуйте снова.")
			continue
		}

		if s.completed {
			sendMessage(bot, chatID, "Этот код уже использован. Нажмите «Войти» в лаунчере для нового входа.")
			continue
		}

		// Проверка подписки на канал (перед проверкой сессии)
		if getRequiredChannel() != "" && !checkSubscription(bot, userID) {
			subscriptionPending.Lock()
			subscriptionPending.m[userID] = pendingSubscription{code: code, createdAt: time.Now()}
			subscriptionPending.Unlock()
			sendSubscriptionRequired(bot, chatID, code)
			continue
		}

		// Повторный вход: если у пользователя уже есть сессия — не спрашиваем никнейм
		telegramUsername := ""
		if msg.From != nil && msg.From.UserName != "" {
			telegramUsername = msg.From.UserName
		}
		if stored, ok := getStoredEntryByTelegramID(userID); ok && stored.Nickname != "" {
			if err := completeAuth(code, stored.Nickname, telegramUsername, userID); err != nil {
				log.Printf("[Telegram] Ошибка completeAuth (возврат) для user_id=%d: %v", userID, err)
				sendMessage(bot, chatID, "Ошибка при входе. Попробуйте снова.")
			} else {
				log.Printf("[Telegram] Возврат: user_id=%d, nickname=%s, code=%s", userID, stored.Nickname, code)
				sendMessage(bot, chatID, "С возвращением! Вернитесь в лаунчер — кнопка изменится на «Играть».")
			}
			continue
		}

		nicknamePending.Lock()
		nicknamePending.m[userID] = pendingNickname{code: code, createdAt: time.Now()}
		nicknamePending.Unlock()

		log.Printf("[Telegram] Ожидание никнейма от user_id=%d для code=%s", userID, code)
		sendMessage(bot, chatID, "Введите ваш никнейм для Minecraft (3–16 символов, латиница, цифры, подчёркивание):")
	}
}

func sendMessage(bot *tgbotapi.BotAPI, chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = serverStatusKeyboard()
	if _, err := bot.Send(msg); err != nil {
		log.Printf("[Telegram] Ошибка отправки сообщения: %v", err)
	}
}

func answerCallback(bot *tgbotapi.BotAPI, callbackID string, text string) {
	cfg := tgbotapi.NewCallback(callbackID, text)
	if _, err := bot.Request(cfg); err != nil {
		log.Printf("[Telegram] Ошибка answerCallback: %v", err)
	}
}

// handleNotificationsSettings показывает настройки уведомлений и inline-кнопки для выбора порога
func handleNotificationsSettings(bot *tgbotapi.BotAPI, chatID, userID int64) {
	stored, ok := sessionGetByTelegramID(userID)
	if !ok || stored.Nickname == "" {
		sendMessage(bot, chatID, "Уведомления доступны после входа. Нажмите «Войти» в лаунчере и пройдите авторизацию через /start с кодом.")
		return
	}
	threshold := stored.NotifyThreshold
	if threshold <= 0 {
		threshold = config.NotificationDefaultThreshold
		if threshold <= 0 {
			threshold = 2
		}
	}
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Уведомлять при количестве игроков: %d\n\nВыберите порог:", threshold))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("1", "notif_thresh_1"),
			tgbotapi.NewInlineKeyboardButtonData("2", "notif_thresh_2"),
			tgbotapi.NewInlineKeyboardButtonData("3", "notif_thresh_3"),
			tgbotapi.NewInlineKeyboardButtonData("4", "notif_thresh_4"),
			tgbotapi.NewInlineKeyboardButtonData("5", "notif_thresh_5"),
		),
	)
	if _, err := bot.Send(msg); err != nil {
		log.Printf("[Telegram] Ошибка отправки: %v", err)
	}
}

// handleNotificationThresholdCallback обрабатывает выбор порога уведомлений
func handleNotificationThresholdCallback(bot *tgbotapi.BotAPI, cb *tgbotapi.CallbackQuery, userID, chatID int64) {
	prefix := "notif_thresh_"
	if !strings.HasPrefix(cb.Data, prefix) {
		return
	}
	nStr := strings.TrimPrefix(cb.Data, prefix)
	var n int
	if _, err := fmt.Sscanf(nStr, "%d", &n); err != nil || n < 1 || n > 99 {
		answerCallback(bot, cb.ID, "Неверное значение")
		return
	}
	sessionUpdateNotifyThreshold(userID, n)
	answerCallback(bot, cb.ID, fmt.Sprintf("Готово! Буду уведомлять при %d игроках.", n))
	m := tgbotapi.NewMessage(chatID, fmt.Sprintf("Готово! Буду уведомлять при %d игроках.", n))
	m.ReplyMarkup = serverStatusKeyboard()
	if _, err := bot.Send(m); err != nil {
		log.Printf("[Telegram] Ошибка отправки: %v", err)
	}
}

const notificationWorkerInterval = 90 * time.Second

// startNotificationWorker запускает фоновую проверку онлайна и отправку уведомлений
func startNotificationWorker(bot *tgbotapi.BotAPI) {
	cooldownHours := config.NotificationCooldownHours
	if cooldownHours <= 0 {
		cooldownHours = 2
	}
	cooldown := time.Duration(cooldownHours) * time.Hour

	ticker := time.NewTicker(notificationWorkerInterval)
	defer ticker.Stop()

	for range ticker.C {
		status := CheckMinecraftServerStatus()
		if !status.Online || status.Count == 0 {
			continue
		}
		// Players.Sample может быть пустым (ограничение vanilla — макс 12 игроков, или сервер не отдаёт список).
		// Без списка имён нельзя определить, кто уже на сервере — не рассылаем уведомления, чтобы не флудить.
		if len(status.PlayerNames) == 0 {
			continue
		}

		users := sessionGetAllForNotifications()
		playerNamesSet := make(map[string]bool)
		for _, n := range status.PlayerNames {
			playerNamesSet[strings.ToLower(n)] = true
		}

		playerListStr := strings.Join(status.PlayerNames, ", ")
		if playerListStr == "" {
			playerListStr = fmt.Sprintf("%d игроков", status.Count)
		}

		now := time.Now()

		for _, u := range users {
			if status.Count < u.NotifyThreshold {
				continue
			}
			if playerNamesSet[strings.ToLower(u.Nickname)] {
				continue // пользователь уже на сервере
			}
			if u.LastNotifiedAt != nil {
				lastNotif := time.Unix(*u.LastNotifiedAt, 0)
				if now.Sub(lastNotif) < cooldown {
					continue
				}
			}

			text := buildNotificationText(u, playerListStr)
			msg := tgbotapi.NewMessage(u.TelegramID, text)
			if _, err := bot.Send(msg); err != nil {
				log.Printf("[Telegram] Ошибка отправки уведомления user_id=%d: %v", u.TelegramID, err)
				continue
			}
			sessionUpdateLastNotified(u.TelegramID)
			log.Printf("[Telegram] Уведомление отправлено user_id=%d", u.TelegramID)
		}
	}
}

func buildNotificationText(u NotificationEntry, playerListStr string) string {
	var greeting string
	if u.LastLoginAt == nil {
		greeting = "Давно не видел тебя в игре!"
	} else {
		lastLogin := time.Unix(*u.LastLoginAt, 0)
		if time.Since(lastLogin) < 24*time.Hour {
			greeting = "Давно не видел тебя в игре!"
		} else {
			days := int(time.Since(lastLogin).Hours() / 24)
			greeting = fmt.Sprintf("Тебя не было в игре уже целых %d суток!", days)
		}
	}
	return fmt.Sprintf("Привет! %s Сейчас на сервере: %s. Заходи :)", greeting, playerListStr)
}
