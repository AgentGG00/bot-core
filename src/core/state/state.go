package state

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

func Init(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	schema := `
CREATE TABLE IF NOT EXISTS sessions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	chat_id INTEGER NOT NULL,
	created_at INTEGER NOT NULL,
	expires_at INTEGER NOT NULL,
	revoked INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS totp_replay (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	chat_id INTEGER NOT NULL,
	code TEXT NOT NULL,
	used_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS jobs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	target TEXT NOT NULL,
	version TEXT NOT NULL,
	scheduled_at INTEGER NOT NULL,
	status TEXT NOT NULL,
	created_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS versions (
	target TEXT PRIMARY KEY,
	current_version TEXT NOT NULL,
	updated_at INTEGER NOT NULL
);
`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// GetCurrentVersion returns the last version recorded for target.
// found is false if nothing has been recorded yet.
func GetCurrentVersion(db *sql.DB, target string) (version string, found bool, err error) {
	err = db.QueryRow(`SELECT current_version FROM versions WHERE target = ?`, target).Scan(&version)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return version, true, nil
}

// SetCurrentVersion records version as the last known version for target.
func SetCurrentVersion(db *sql.DB, target, version string) error {
	now := time.Now().Unix()
	_, err := db.Exec(
		`INSERT INTO versions (target, current_version, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(target) DO UPDATE SET current_version = excluded.current_version, updated_at = excluded.updated_at`,
		target, version, now,
	)
	return err
}
