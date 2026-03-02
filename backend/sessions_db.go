package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"sync"

	_ "modernc.org/sqlite"
)

const sessionsDBFile = "sessions.db"
const validSessionsFile = "valid-sessions.json"

var (
	sessionsDB   *sql.DB
	sessionsDBMu sync.RWMutex
)

func initSessionsDB() {
	db, err := sql.Open("sqlite", sessionsDBFile+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		log.Fatalf("[Auth] Не удалось открыть SQLite: %v", err)
	}
	sessionsDB = db

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			session_uuid TEXT PRIMARY KEY,
			nickname TEXT NOT NULL,
			telegram_id INTEGER NOT NULL DEFAULT 0,
			telegram_username TEXT NOT NULL DEFAULT ''
		);
		CREATE INDEX IF NOT EXISTS idx_sessions_telegram_id ON sessions(telegram_id) WHERE telegram_id != 0;
	`)
	if err != nil {
		log.Fatalf("[Auth] Не удалось создать таблицу sessions: %v", err)
	}

	migrateFromJSON()
	log.Printf("[Auth] SQLite sessions: %s", sessionsDBFile)
}

func migrateFromJSON() {
	data, err := os.ReadFile(validSessionsFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[Auth] Не удалось прочитать %s для миграции: %v", validSessionsFile, err)
		}
		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		log.Printf("[Auth] Невалидный %s для миграции: %v", validSessionsFile, err)
		return
	}

	count := 0
	for k, v := range raw {
		if k == "" {
			continue
		}
		var entry ValidSessionEntry
		if err := json.Unmarshal(v, &entry); err == nil && entry.Nickname != "" {
			if err := sessionSave(k, entry); err == nil {
				count++
			}
		} else {
			var nickname string
			if err := json.Unmarshal(v, &nickname); err == nil && nickname != "" {
				if err := sessionSave(k, ValidSessionEntry{Nickname: nickname}); err == nil {
					count++
				}
			}
		}
	}
	if count > 0 {
		log.Printf("[Auth] Мигрировано %d сессий из %s", count, validSessionsFile)
		// Переименуем старый файл как бэкап
		_ = os.Rename(validSessionsFile, validSessionsFile+".bak")
	}
}

func sessionGetByUUID(sessionUUID string) (ValidSessionEntry, bool) {
	sessionsDBMu.RLock()
	defer sessionsDBMu.RUnlock()
	var nickname, telegramUsername string
	var telegramID int64
	err := sessionsDB.QueryRow(
		`SELECT nickname, telegram_id, telegram_username FROM sessions WHERE session_uuid = ?`,
		sessionUUID,
	).Scan(&nickname, &telegramID, &telegramUsername)
	if err == sql.ErrNoRows || err != nil {
		return ValidSessionEntry{}, false
	}
	return ValidSessionEntry{Nickname: nickname, TelegramID: telegramID, TelegramUsername: telegramUsername}, true
}

func sessionGetByTelegramID(telegramID int64) (ValidSessionEntry, bool) {
	if telegramID == 0 {
		return ValidSessionEntry{}, false
	}
	sessionsDBMu.RLock()
	defer sessionsDBMu.RUnlock()
	var nickname, telegramUsername string
	var tid int64
	err := sessionsDB.QueryRow(
		`SELECT nickname, telegram_id, telegram_username FROM sessions WHERE telegram_id = ? LIMIT 1`,
		telegramID,
	).Scan(&nickname, &tid, &telegramUsername)
	if err == sql.ErrNoRows || err != nil {
		return ValidSessionEntry{}, false
	}
	return ValidSessionEntry{Nickname: nickname, TelegramID: tid, TelegramUsername: telegramUsername}, true
}

func sessionSave(sessionUUID string, entry ValidSessionEntry) error {
	sessionsDBMu.Lock()
	defer sessionsDBMu.Unlock()
	_, err := sessionsDB.Exec(
		`INSERT OR REPLACE INTO sessions (session_uuid, nickname, telegram_id, telegram_username) VALUES (?, ?, ?, ?)`,
		sessionUUID, entry.Nickname, entry.TelegramID, entry.TelegramUsername,
	)
	return err
}

func sessionDeleteByUUID(sessionUUID string) {
	sessionsDBMu.Lock()
	defer sessionsDBMu.Unlock()
	sessionsDB.Exec(`DELETE FROM sessions WHERE session_uuid = ?`, sessionUUID)
}

func sessionDeleteByNickname(nickname string) {
	sessionsDBMu.Lock()
	defer sessionsDBMu.Unlock()
	sessionsDB.Exec(`DELETE FROM sessions WHERE LOWER(nickname) = LOWER(?)`, nickname)
}

func sessionDeleteByTelegramID(telegramID int64) {
	if telegramID == 0 {
		return
	}
	sessionsDBMu.Lock()
	defer sessionsDBMu.Unlock()
	sessionsDB.Exec(`DELETE FROM sessions WHERE telegram_id = ?`, telegramID)
}
