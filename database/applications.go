package database

import (
	"database/sql"
	"time"
)

func (db *DB) CreateApplication(app *Application) error {
	_, err := db.conn.Exec(`
		INSERT INTO applications (telegram_id, document_file_id, voice_note_file_id, status)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (telegram_id) DO UPDATE SET 
			document_file_id = $2,
			voice_note_file_id = $3,
			status = $4,
			submitted_at = NOW(),
			rework_attempts = rework_attempts + 1
	`, app.TelegramID, app.DocumentFileID, app.VoiceNoteFileID, app.Status)
	return err
}

func (db *DB) GetActiveApplication(tgID int64) (*Application, error) {
	app := &Application{}
	err := db.conn.QueryRow(`
		SELECT id, telegram_id, document_file_id, voice_note_file_id, status, submitted_at, rework_attempts
		FROM applications
		WHERE telegram_id = $1 AND status IN ('pending', 'rework_requested')
		LIMIT 1
	`, tgID).Scan(&app.ID, &app.TelegramID, &app.DocumentFileID, &app.VoiceNoteFileID,
		&app.Status, &app.SubmittedAt, &app.ReworkAttempts)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	return app, err
}

func (db *DB) GetPendingApplications() ([]*Application, error) {
	rows, err := db.conn.Query(`
		SELECT id, telegram_id, document_file_id, voice_note_file_id, status, submitted_at, rework_attempts
		FROM applications
		WHERE status IN ('pending', 'rework_requested')
		ORDER BY submitted_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var apps []*Application
	for rows.Next() {
		app := &Application{}
		err := rows.Scan(&app.ID, &app.TelegramID, &app.DocumentFileID, &app.VoiceNoteFileID,
			&app.Status, &app.SubmittedAt, &app.ReworkAttempts)
		if err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}

	return apps, rows.Err()
}

func (db *DB) UpdateApplicationStatus(tgID int64, status string) error {
	now := time.Now()
	_, err := db.conn.Exec(`
		UPDATE applications SET status = $1, reviewed_at = $2 WHERE telegram_id = $3
	`, status, now, tgID)
	return err
}

func (db *DB) UpdateApplicationRework(tgID int64, comment string) error {
	_, err := db.conn.Exec(`
		UPDATE applications 
		SET status = 'rework_requested', admin_comment = $1 
		WHERE telegram_id = $2
	`, comment, tgID)
	return err
}

func (db *DB) ApproveApplication(tgID int64, adminID int64) error {
	now := time.Now()
	_, err := db.conn.Exec(`
		UPDATE applications 
		SET status = 'approved', reviewed_at = $1, reviewed_by = $2 
		WHERE telegram_id = $3
	`, now, adminID, tgID)
	return err
}

func (db *DB) RejectApplication(tgID int64, adminID int64) error {
	now := time.Now()
	_, err := db.conn.Exec(`
		UPDATE applications 
		SET status = 'rejected', reviewed_at = $1, reviewed_by = $2 
		WHERE telegram_id = $3
	`, now, adminID, tgID)
	return err
}

func (db *DB) DeleteApplicationDocuments(tgID int64) error {
	_, err := db.conn.Exec(`
		UPDATE applications 
		SET document_file_id = NULL, voice_note_file_id = NULL 
		WHERE telegram_id = $1
	`, tgID)
	return err
}
