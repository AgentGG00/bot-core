package auth

import (
	"database/sql"
	"errors"
	"time"

	"github.com/pquerna/otp/totp"
)

const SessionDuration = 6 * 30 * 24 * time.Hour
const replayWindow = 90 * time.Second

func Login(secret, code string, db *sql.DB, chatID int64) error {
	if !totp.Validate(code, secret) {
		return errors.New("invalid code")
	}
	if err := checkReplay(db, chatID, code); err != nil {
		return err
	}
	now := time.Now()
	_, err := db.Exec(
		`INSERT INTO sessions (chat_id, created_at, expires_at, revoked) VALUES (?, ?, ?, 0)`,
		chatID, now.Unix(), now.Add(SessionDuration).Unix(),
	)
	return err
}

func VerifyStepUp(secret, code string, db *sql.DB, chatID int64) error {
	if !totp.Validate(code, secret) {
		return errors.New("invalid code")
	}
	return checkReplay(db, chatID, code)
}

func checkReplay(db *sql.DB, chatID int64, code string) error {
	var count int
	cutoff := time.Now().Add(-replayWindow).Unix()
	err := db.QueryRow(
		`SELECT COUNT(*) FROM totp_replay WHERE chat_id = ? AND code = ? AND used_at > ?`,
		chatID, code, cutoff,
	).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("code already used")
	}
	_, err = db.Exec(
		`INSERT INTO totp_replay (chat_id, code, used_at) VALUES (?, ?, ?)`,
		chatID, code, time.Now().Unix(),
	)
	return err
}

func Logout(db *sql.DB, chatID int64) error {
	_, err := db.Exec(`UPDATE sessions SET revoked = 1 WHERE chat_id = ?`, chatID)
	return err
}

func HasActiveSession(db *sql.DB, chatID int64) (bool, error) {
	var count int
	now := time.Now().Unix()
	err := db.QueryRow(
		`SELECT COUNT(*) FROM sessions WHERE chat_id = ? AND revoked = 0 AND expires_at > ?`,
		chatID, now,
	).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
