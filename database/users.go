package database

import (
	"database/sql"
	"time"
)

func (db *DB) GetUser(tgID int64) (*User, error) {
	user := &User{}
	err := db.conn.QueryRow(`
		SELECT id, telegram_id, first_name, status, created_at, updated_at
		FROM users WHERE telegram_id = $1
	`, tgID).Scan(&user.ID, &user.TelegramID, &user.FirstName, &user.Status, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return user, err
}

func (db *DB) CreateUser(tgID int64, firstName string) (*User, error) {
	user := &User{
		TelegramID: tgID,
		FirstName:  firstName,
		Status:     "pending",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := db.conn.QueryRow(`
		INSERT INTO users (telegram_id, first_name, status)
		VALUES ($1, $2, $3)
		ON CONFLICT (telegram_id) DO UPDATE SET updated_at = NOW()
		RETURNING id
	`, tgID, firstName, user.Status).Scan(&user.ID)

	return user, err
}

func (db *DB) UpdateUserStatus(tgID int64, status string) error {
	_, err := db.conn.Exec(`
		UPDATE users SET status = $1, updated_at = NOW() WHERE telegram_id = $2
	`, status, tgID)
	return err
}

func (db *DB) DetermineUserState(tgID int64) (string, error) {
	// Проверка 1: BLACKLIST
	var inBlacklist bool
	err := db.conn.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM blacklist WHERE telegram_id = $1)", tgID,
	).Scan(&inBlacklist)
	if err != nil {
		return "", err
	}
	if inBlacklist {
		return "blocked", nil
	}

	// Проверка 2: WHITELIST
	var inWhitelist bool
	err = db.conn.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM whitelist WHERE telegram_id = $1)", tgID,
	).Scan(&inWhitelist)
	if err != nil {
		return "", err
	}
	if inWhitelist {
		return "approved", nil
	}

	// Проверка 3: Ни там ни там
	return "unverified", nil
}
