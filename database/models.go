package database

import "time"

type User struct {
	ID         int64
	TelegramID int64
	FirstName  string
	Status     string // approved, rejected, blocked, pending
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Application struct {
	ID              int64
	TelegramID      int64
	DocumentFileID  string
	VoiceNoteFileID string
	Status          string // pending, approved, rejected, rework_requested
	AdminComment    string
	SubmittedAt     time.Time
	ReviewedAt      *time.Time
	ReviewedBy      *int64
	ReworkAttempts  int
}

type VerificationLog struct {
	ID         int64
	TelegramID int64
	Action     string // submitted, approved, rejected, rework_requested, auto_blocked
	Timestamp  time.Time
	AdminID    *int64
	Details    string // JSON
}

type WhitelistEntry struct {
	ID         int64
	TelegramID int64
	FirstName  string
	AddedAt    time.Time
}

type BlacklistEntry struct {
	ID         int64
	TelegramID int64
	FirstName  string
	Reason     string
	AddedAt    time.Time
	AddedBy    int64
}
