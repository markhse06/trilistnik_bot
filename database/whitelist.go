package database

func (db *DB) IsWhitelistedByTelegramID(tgID int64) (bool, error) {
	var exists bool
	err := db.conn.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM whitelist WHERE telegram_id = $1)", tgID,
	).Scan(&exists)
	return exists, err
}

func (db *DB) AddToWhitelist(tgID int64, firstName string) error {
	_, err := db.conn.Exec(`
		INSERT INTO whitelist (telegram_id, first_name, added_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (telegram_id) DO NOTHING
	`, tgID, firstName)
	return err
}

func (db *DB) RemoveFromWhitelist(tgID int64) error {
	_, err := db.conn.Exec(
		"DELETE FROM whitelist WHERE telegram_id = $1", tgID,
	)
	return err
}

func (db *DB) GetWhitelistAll() ([]*WhitelistEntry, error) {
	rows, err := db.conn.Query(`
		SELECT id, telegram_id, first_name, added_at
		FROM whitelist
		ORDER BY added_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*WhitelistEntry
	for rows.Next() {
		entry := &WhitelistEntry{}
		err := rows.Scan(&entry.ID, &entry.TelegramID, &entry.FirstName, &entry.AddedAt)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}
