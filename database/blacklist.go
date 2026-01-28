package database

func (db *DB) IsBlacklistedByTelegramID(tgID int64) (bool, error) {
	var exists bool
	err := db.conn.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM blacklist WHERE telegram_id = $1)", tgID,
	).Scan(&exists)
	return exists, err
}

func (db *DB) AddToBlacklist(tgID int64, firstName, reason string, adminID int64) error {
	_, err := db.conn.Exec(`
		INSERT INTO blacklist (telegram_id, first_name, reason, added_by, added_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (telegram_id) DO NOTHING
	`, tgID, firstName, reason, adminID)
	return err
}

func (db *DB) RemoveFromBlacklist(tgID int64) error {
	_, err := db.conn.Exec(
		"DELETE FROM blacklist WHERE telegram_id = $1", tgID,
	)
	return err
}

func (db *DB) GetBlacklistAll() ([]*BlacklistEntry, error) {
	rows, err := db.conn.Query(`
		SELECT id, telegram_id, first_name, reason, added_at, added_by
		FROM blacklist
		ORDER BY added_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*BlacklistEntry
	for rows.Next() {
		entry := &BlacklistEntry{}
		err := rows.Scan(&entry.ID, &entry.TelegramID, &entry.FirstName,
			&entry.Reason, &entry.AddedAt, &entry.AddedBy)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}
