package main

import (
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

// Minecraft nickname: 3-16 символов, буквы, цифры, подчёркивание
var nicknameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,16}$`)

func isValidNickname(s string) bool {
	return nicknameRegex.MatchString(strings.TrimSpace(s))
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
			continue
		}

		// Обработка callback (нажатие кнопки «Проверить подписку»)
		if update.CallbackQuery != nil {
			cb := update.CallbackQuery
			userID := cb.From.ID
			chatID := cb.Message.Chat.ID

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
			sendMessage(bot, chatID, "Введите /start с кодом из лаунчера. Например: /start abc123")
			continue
		}

		cmd := msg.Command()
		if cmd != "start" {
			sendMessage(bot, chatID, "Используйте /start с кодом из лаунчера.")
			continue
		}

		code := strings.TrimSpace(msg.CommandArguments())
		if code == "" {
			sendMessage(bot, chatID, "Нажмите «Войти» в лаунчере — откроется ссылка с кодом. Перейдите по ней и введите код.")
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
