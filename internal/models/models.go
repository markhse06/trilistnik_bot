package models

import (
	"context"
	"database/sql"
	"time"
)

type UserStatus string

const (
	StatusPending   UserStatus = "pending"
	StatusApproved  UserStatus = "approved"
	StatusRejected  UserStatus = "rejected"
	StatusBlocked   UserStatus = "blocked"
	StatusUnverifed UserStatus = "unverified"
)

type Store struct {
	DB *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{DB: db}
}

func (s *Store) IsWhitelisted(ctx context.Context, telegramID int64) (bool, error) {
	var exists bool
	err := s.DB.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM whitelist WHERE telegram_id = $1)`, telegramID,
	).Scan(&exists)
	return exists, err
}

func (s *Store) IsBlacklisted(ctx context.Context, telegramID int64) (bool, error) {
	var exists bool
	err := s.DB.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM blacklist WHERE telegram_id = $1)`, telegramID,
	).Scan(&exists)
	return exists, err
}

func (s *Store) AddToWhitelist(ctx context.Context, telegramID int64) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO whitelist (telegram_id) VALUES ($1)
         ON CONFLICT (telegram_id) DO NOTHING`,
		telegramID,
	)
	return err
}

func (s *Store) RemoveFromWhitelist(ctx context.Context, telegramID int64) error {
	_, err := s.DB.ExecContext(ctx,
		`DELETE FROM whitelist WHERE telegram_id = $1`,
		telegramID,
	)
	return err
}

func (s *Store) AddToBlacklist(ctx context.Context, telegramID int64, reason string) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO blacklist (telegram_id, reason) VALUES ($1, $2)
         ON CONFLICT (telegram_id)
         DO UPDATE SET reason = EXCLUDED.reason`,
		telegramID, reason,
	)
	return err
}

func (s *Store) RemoveFromBlacklist(ctx context.Context, telegramID int64) error {
	_, err := s.DB.ExecContext(ctx,
		`DELETE FROM blacklist WHERE telegram_id = $1`,
		telegramID,
	)
	return err
}

func (s *Store) UpsertUser(ctx context.Context, telegramID int64, username, fullName string, status UserStatus) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO users (telegram_id, username, full_name, status)
         VALUES ($1, $2, $3, $4)
         ON CONFLICT (telegram_id)
         DO UPDATE SET username = EXCLUDED.username,
                       full_name = COALESCE(EXCLUDED.full_name, users.full_name),
                       status = EXCLUDED.status,
                       updated_at = NOW()`,
		telegramID, username, fullName, string(status),
	)
	return err
}

func (s *Store) UpdateUserStatus(ctx context.Context, telegramID int64, status UserStatus) error {
	_, err := s.DB.ExecContext(ctx,
		`UPDATE users SET status = $1, updated_at = $2 WHERE telegram_id = $3`,
		string(status), time.Now(), telegramID,
	)
	return err
}

func (s *Store) GetUserStatus(ctx context.Context, telegramID int64) (string, error) {
	isBlack, err := s.IsBlacklisted(ctx, telegramID)
	if err != nil {
		return "", err
	}
	if isBlack {
		return "blacklisted", nil
	}

	isWhite, err := s.IsWhitelisted(ctx, telegramID)
	if err != nil {
		return "", err
	}
	if isWhite {
		return "whitelisted", nil
	}

	return "unverified", nil
}
