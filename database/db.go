package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

type DB struct {
	conn *sql.DB
}

func Connect(dsn string) (*DB, error) {
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	conn.SetMaxOpenConns(25)
	conn.SetMaxIdleConns(5)

	log.Println("✅ Database connected")
	return &DB{conn: conn}, nil
}

func (db *DB) RunMigrations() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id BIGSERIAL PRIMARY KEY,
		telegram_id BIGINT UNIQUE NOT NULL,
		first_name VARCHAR(255),
		status VARCHAR(50) DEFAULT 'pending',
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS applications (
		id BIGSERIAL PRIMARY KEY,
		telegram_id BIGINT UNIQUE NOT NULL,
		document_file_id VARCHAR(500),
		voice_note_file_id VARCHAR(500),
		status VARCHAR(50) DEFAULT 'pending',
		admin_comment TEXT,
		submitted_at TIMESTAMP DEFAULT NOW(),
		reviewed_at TIMESTAMP,
		reviewed_by BIGINT,
		rework_attempts INT DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS whitelist (
		id BIGSERIAL PRIMARY KEY,
		telegram_id BIGINT UNIQUE NOT NULL,
		first_name VARCHAR(255),
		added_at TIMESTAMP DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS blacklist (
		id BIGSERIAL PRIMARY KEY,
		telegram_id BIGINT UNIQUE NOT NULL,
		first_name VARCHAR(255),
		reason VARCHAR(500),
		added_at TIMESTAMP DEFAULT NOW(),
		added_by BIGINT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS verification_logs (
		id BIGSERIAL PRIMARY KEY,
		telegram_id BIGINT NOT NULL,
		action VARCHAR(50),
		timestamp TIMESTAMP DEFAULT NOW(),
		admin_id BIGINT,
		details TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_users_tg_id ON users(telegram_id);
	CREATE INDEX IF NOT EXISTS idx_applications_tg_id ON applications(telegram_id);
	CREATE INDEX IF NOT EXISTS idx_whitelist_tg_id ON whitelist(telegram_id);
	CREATE INDEX IF NOT EXISTS idx_blacklist_tg_id ON blacklist(telegram_id);
	CREATE INDEX IF NOT EXISTS idx_logs_tg_id ON verification_logs(telegram_id);
	`

	_, err := db.conn.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Println("✅ Migrations completed")
	return nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}
