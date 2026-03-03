package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

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

	goose.SetBaseFS(embedMigrations)
	if err := goose.SetDialect("sqlite3"); err != nil {
		log.Fatalf("[Auth] goose SetDialect: %v", err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		log.Fatalf("[Auth] goose Up: %v", err)
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
	var lastLoginAt sql.NullInt64
	err := sessionsDB.QueryRow(
		`SELECT nickname, telegram_id, telegram_username, last_login_at FROM sessions WHERE session_uuid = ?`,
		sessionUUID,
	).Scan(&nickname, &telegramID, &telegramUsername, &lastLoginAt)
	if err == sql.ErrNoRows || err != nil {
		return ValidSessionEntry{}, false
	}
	entry := ValidSessionEntry{Nickname: nickname, TelegramID: telegramID, TelegramUsername: telegramUsername}
	if lastLoginAt.Valid {
		entry.LastLoginAt = &lastLoginAt.Int64
	}
	return entry, true
}

func sessionGetByTelegramID(telegramID int64) (ValidSessionEntry, bool) {
	if telegramID == 0 {
		return ValidSessionEntry{}, false
	}
	sessionsDBMu.RLock()
	defer sessionsDBMu.RUnlock()
	var nickname, telegramUsername string
	var tid int64
	var lastLoginAt sql.NullInt64
	err := sessionsDB.QueryRow(
		`SELECT nickname, telegram_id, telegram_username, last_login_at FROM sessions WHERE telegram_id = ? LIMIT 1`,
		telegramID,
	).Scan(&nickname, &tid, &telegramUsername, &lastLoginAt)
	if err == sql.ErrNoRows || err != nil {
		return ValidSessionEntry{}, false
	}
	entry := ValidSessionEntry{Nickname: nickname, TelegramID: tid, TelegramUsername: telegramUsername}
	if lastLoginAt.Valid {
		entry.LastLoginAt = &lastLoginAt.Int64
	}
	return entry, true
}

func sessionSave(sessionUUID string, entry ValidSessionEntry) error {
	sessionsDBMu.Lock()
	defer sessionsDBMu.Unlock()
	_, err := sessionsDB.Exec(
		`INSERT OR REPLACE INTO sessions (session_uuid, nickname, telegram_id, telegram_username, last_login_at) VALUES (?, ?, ?, ?, ?)`,
		sessionUUID, entry.Nickname, entry.TelegramID, entry.TelegramUsername, nullInt64(entry.LastLoginAt),
	)
	return err
}

func nullInt64(p *int64) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func sessionUpdateLastLogin(sessionUUID string) {
	sessionsDBMu.Lock()
	defer sessionsDBMu.Unlock()
	sessionsDB.Exec(`UPDATE sessions SET last_login_at = ? WHERE session_uuid = ?`, time.Now().Unix(), sessionUUID)
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
