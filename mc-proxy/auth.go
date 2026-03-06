package main

import (
	"database/sql"
	"log"
	"os"
	"sync"

	_ "modernc.org/sqlite"
)

const sessionsDBFile = "sessions.db"

var (
	sessionsDB   *sql.DB
	sessionsDBMu sync.RWMutex
)

func initSessionsDB() {
	path := getEnv("SESSIONS_DB_PATH", sessionsDBFile)
	if _, err := os.Stat(path); err != nil {
		if path == sessionsDBFile {
			for _, fallback := range []string{"../backend/data/sessions.db", "backend/data/sessions.db"} {
				if _, err := os.Stat(fallback); err == nil {
					path = fallback
					break
				}
			}
		}
	}
	if _, err := os.Stat(path); err != nil {
		log.Fatalf("[Auth] Sessions DB не найден: %s — mc-proxy не запускается без сессий", path)
	}
	db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
	if err != nil {
		log.Fatalf("[Auth] Не удалось открыть sessions DB (read-only): %v", err)
	}
	sessionsDB = db
	log.Printf("[Auth] Sessions DB (read-only): %s", path)
}

func sessionVerify(nickname, sessionUUID string) bool {
	if nickname == "" || sessionUUID == "" {
		return false
	}
	sessionsDBMu.RLock()
	defer sessionsDBMu.RUnlock()
	if sessionsDB == nil {
		return false
	}
	var dbNickname string
	err := sessionsDB.QueryRow(
		`SELECT nickname FROM sessions WHERE session_uuid = ?`,
		sessionUUID,
	).Scan(&dbNickname)
	if err == sql.ErrNoRows || err != nil {
		return false
	}
	return dbNickname == nickname
}
