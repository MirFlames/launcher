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

func sessionCount() int {
	sessionsDBMu.RLock()
	defer sessionsDBMu.RUnlock()
	var n int
	if err := sessionsDB.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n); err != nil {
		return 0
	}
	return n
}

// SessionExportEntry — запись для экспорта в mc-proxy (минимальный набор для верификации)
type SessionExportEntry struct {
	Nickname    string `json:"nickname"`
	SessionUUID string `json:"session_uuid"`
}

// sessionExportForProxy возвращает все сессии для репликации в mc-proxy
func sessionExportForProxy() []SessionExportEntry {
	sessionsDBMu.RLock()
	defer sessionsDBMu.RUnlock()
	rows, err := sessionsDB.Query(`SELECT nickname, session_uuid FROM sessions WHERE session_uuid != '' AND nickname != ''`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []SessionExportEntry
	for rows.Next() {
		var e SessionExportEntry
		if err := rows.Scan(&e.Nickname, &e.SessionUUID); err != nil {
			continue
		}
		result = append(result, e)
	}
	return result
}

func sessionGetByUUID(sessionUUID string) (ValidSessionEntry, bool) {
	sessionsDBMu.RLock()
	defer sessionsDBMu.RUnlock()
	var nickname, telegramUsername, launcherVersion string
	var telegramID int64
	var lastLoginAt, lastNotifiedAt sql.NullInt64
	var notifyThreshold int
	err := sessionsDB.QueryRow(
		`SELECT nickname, telegram_id, telegram_username, launcher_version, last_login_at, COALESCE(notify_threshold, 2), last_notified_at FROM sessions WHERE session_uuid = ?`,
		sessionUUID,
	).Scan(&nickname, &telegramID, &telegramUsername, &launcherVersion, &lastLoginAt, &notifyThreshold, &lastNotifiedAt)
	if err == sql.ErrNoRows || err != nil {
		return ValidSessionEntry{}, false
	}
	entry := ValidSessionEntry{
		Nickname:        nickname,
		TelegramID:      telegramID,
		TelegramUsername: telegramUsername,
		LauncherVersion: launcherVersion,
		NotifyThreshold: notifyThreshold,
	}
	if lastLoginAt.Valid {
		entry.LastLoginAt = &lastLoginAt.Int64
	}
	if lastNotifiedAt.Valid {
		entry.LastNotifiedAt = &lastNotifiedAt.Int64
	}
	return entry, true
}

func sessionGetByTelegramID(telegramID int64) (ValidSessionEntry, bool) {
	if telegramID == 0 {
		return ValidSessionEntry{}, false
	}
	sessionsDBMu.RLock()
	defer sessionsDBMu.RUnlock()
	var nickname, telegramUsername, launcherVersion string
	var tid int64
	var lastLoginAt, lastNotifiedAt sql.NullInt64
	var notifyThreshold int
	err := sessionsDB.QueryRow(
		`SELECT nickname, telegram_id, telegram_username, launcher_version, last_login_at, COALESCE(notify_threshold, 2), last_notified_at FROM sessions WHERE telegram_id = ? LIMIT 1`,
		telegramID,
	).Scan(&nickname, &tid, &telegramUsername, &launcherVersion, &lastLoginAt, &notifyThreshold, &lastNotifiedAt)
	if err == sql.ErrNoRows || err != nil {
		return ValidSessionEntry{}, false
	}
	entry := ValidSessionEntry{
		Nickname:        nickname,
		TelegramID:      tid,
		TelegramUsername: telegramUsername,
		LauncherVersion: launcherVersion,
		NotifyThreshold: notifyThreshold,
	}
	if lastLoginAt.Valid {
		entry.LastLoginAt = &lastLoginAt.Int64
	}
	if lastNotifiedAt.Valid {
		entry.LastNotifiedAt = &lastNotifiedAt.Int64
	}
	return entry, true
}

func sessionSave(sessionUUID string, entry ValidSessionEntry) error {
	sessionsDBMu.Lock()
	defer sessionsDBMu.Unlock()
	threshold := entry.NotifyThreshold
	if threshold <= 0 {
		threshold = 2
	}
	_, err := sessionsDB.Exec(
		`INSERT OR REPLACE INTO sessions (session_uuid, nickname, telegram_id, telegram_username, launcher_version, last_login_at, notify_threshold, last_notified_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		sessionUUID,
		entry.Nickname,
		entry.TelegramID,
		entry.TelegramUsername,
		entry.LauncherVersion,
		nullInt64(entry.LastLoginAt),
		threshold,
		nullInt64(entry.LastNotifiedAt),
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

func sessionUpdateLauncherVersion(sessionUUID, launcherVersion string) {
	if sessionUUID == "" || launcherVersion == "" {
		return
	}
	sessionsDBMu.Lock()
	defer sessionsDBMu.Unlock()
	sessionsDB.Exec(`UPDATE sessions SET launcher_version = ? WHERE session_uuid = ?`, launcherVersion, sessionUUID)
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

// NotificationEntry — запись для фонового воркера уведомлений
type NotificationEntry struct {
	TelegramID      int64
	Nickname        string
	LastLoginAt     *int64
	NotifyThreshold int
	LastNotifiedAt  *int64
}

func sessionGetAllForNotifications() []NotificationEntry {
	sessionsDBMu.RLock()
	defer sessionsDBMu.RUnlock()
	rows, err := sessionsDB.Query(
		`SELECT telegram_id, nickname, last_login_at, COALESCE(notify_threshold, 2), last_notified_at FROM sessions WHERE telegram_id != 0`,
	)
	if err != nil {
		log.Printf("[Sessions] sessionGetAllForNotifications: %v", err)
		return nil
	}
	defer rows.Close()
	var result []NotificationEntry
	for rows.Next() {
		var e NotificationEntry
		var lastLoginAt, lastNotifiedAt sql.NullInt64
		if err := rows.Scan(&e.TelegramID, &e.Nickname, &lastLoginAt, &e.NotifyThreshold, &lastNotifiedAt); err != nil {
			log.Printf("[Sessions] scan: %v", err)
			continue
		}
		if lastLoginAt.Valid {
			e.LastLoginAt = &lastLoginAt.Int64
		}
		if lastNotifiedAt.Valid {
			e.LastNotifiedAt = &lastNotifiedAt.Int64
		}
		result = append(result, e)
	}
	return result
}

type AdminSessionEntry struct {
	Nickname        string
	LauncherVersion string
	LastLoginAt     *int64
}

func sessionGetAllForAdmin() []AdminSessionEntry {
	sessionsDBMu.RLock()
	defer sessionsDBMu.RUnlock()
	rows, err := sessionsDB.Query(
		`SELECT nickname, launcher_version, last_login_at FROM sessions ORDER BY nickname COLLATE NOCASE`,
	)
	if err != nil {
		log.Printf("[Sessions] sessionGetAllForAdmin: %v", err)
		return nil
	}
	defer rows.Close()
	var result []AdminSessionEntry
	for rows.Next() {
		var e AdminSessionEntry
		var lastLoginAt sql.NullInt64
		if err := rows.Scan(&e.Nickname, &e.LauncherVersion, &lastLoginAt); err != nil {
			log.Printf("[Sessions] scan admin: %v", err)
			continue
		}
		if lastLoginAt.Valid {
			v := lastLoginAt.Int64
			e.LastLoginAt = &v
		}
		result = append(result, e)
	}
	return result
}

func sessionUpdateNotifyThreshold(telegramID int64, threshold int) {
	if telegramID == 0 || threshold <= 0 {
		return
	}
	sessionsDBMu.Lock()
	defer sessionsDBMu.Unlock()
	sessionsDB.Exec(`UPDATE sessions SET notify_threshold = ? WHERE telegram_id = ?`, threshold, telegramID)
}

func sessionUpdateLastNotified(telegramID int64) {
	if telegramID == 0 {
		return
	}
	sessionsDBMu.Lock()
	defer sessionsDBMu.Unlock()
	res, err := sessionsDB.Exec(`UPDATE sessions SET last_notified_at = ? WHERE telegram_id = ?`, time.Now().Unix(), telegramID)
	if err != nil {
		log.Printf("[Sessions] sessionUpdateLastNotified FAILED telegram_id=%d: %v", telegramID, err)
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		log.Printf("[Sessions] sessionUpdateLastNotified: no rows updated for telegram_id=%d (session may not exist)", telegramID)
	}
}
