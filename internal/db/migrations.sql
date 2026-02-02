CREATE TABLE IF NOT EXISTS whitelist (
                                         id SERIAL PRIMARY KEY,
                                         telegram_id BIGINT UNIQUE NOT NULL,
                                         created_at TIMESTAMP NOT NULL DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS blacklist (
                                         id SERIAL PRIMARY KEY,
                                         telegram_id BIGINT UNIQUE NOT NULL,
                                         reason TEXT,
                                         created_at TIMESTAMP NOT NULL DEFAULT NOW()
    );

CREATE TABLE IF NOT EXISTS users (
                                     id SERIAL PRIMARY KEY,
                                     telegram_id BIGINT UNIQUE NOT NULL,
                                     full_name TEXT,
                                     username TEXT,
                                     status TEXT NOT NULL, -- pending, approved, rejected, blocked, unverified
                                     created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
    );
