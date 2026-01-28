package database

func (db *DB) AddVerificationLog(tgID int64, action string, adminID *int64, details string) error {
	_, err := db.conn.Exec(`
		INSERT INTO verification_logs (telegram_id, action, admin_id, details, timestamp)
		VALUES ($1, $2, $3, $4, NOW())
	`, tgID, action, adminID, details)
	return err
}

func (db *DB) GetUserHistory(tgID int64, limit int) ([]*VerificationLog, error) {
	rows, err := db.conn.Query(`
		SELECT id, telegram_id, action, timestamp, admin_id, details
		FROM verification_logs
		WHERE telegram_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`, tgID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*VerificationLog
	for rows.Next() {
		log := &VerificationLog{}
		err := rows.Scan(&log.ID, &log.TelegramID, &log.Action, &log.Timestamp, &log.AdminID, &log.Details)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

func (db *DB) GetStats() (map[string]int, error) {
	stats := make(map[string]int)

	rows := db.conn.QueryRow(`
		SELECT
			(SELECT COUNT(*) FROM applications WHERE status = 'pending') as pending,
			(SELECT COUNT(*) FROM applications WHERE status = 'rework_requested') as rework,
			(SELECT COUNT(*) FROM whitelist) as approved,
			(SELECT COUNT(*) FROM blacklist) as blocked,
			(SELECT COUNT(*) FROM applications WHERE status = 'rejected') as rejected
	`)

	var pending, rework, approved, blocked, rejected int
	err := rows.Scan(&pending, &rework, &approved, &blocked, &rejected)
	if err != nil {
		return nil, err
	}

	stats["pending"] = pending
	stats["rework"] = rework
	stats["approved"] = approved
	stats["blocked"] = blocked
	stats["rejected"] = rejected

	return stats, nil
}
