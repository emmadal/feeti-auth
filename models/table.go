package models

import (
	"context"
	"time"
)

// createTables create tables
func createTables() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			first_name VARCHAR(100) NOT NULL,
			last_name VARCHAR(100) NOT NULL,
			phone_number VARCHAR(18) UNIQUE NOT NULL,
			device_token VARCHAR(100) NOT NULL,
			pin VARCHAR(100) NOT NULL,
			quota BIGINT DEFAULT 0 NOT NULL,
			locked BOOLEAN DEFAULT FALSE NOT NULL,
			premium BOOLEAN DEFAULT FALSE NOT NULL,
			photo VARCHAR(200),
			is_active BOOLEAN DEFAULT TRUE NOT NULL,
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS otp (
			id SERIAL PRIMARY KEY,
			code VARCHAR(7) NOT NULL,
			is_used BOOLEAN DEFAULT FALSE NOT NULL,
			phone_number VARCHAR(18) NOT NULL,
			key_uid VARCHAR(100) NOT NULL,
			expiry_at TIMESTAMP,
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS wallets (
			id SERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL,
			balance BIGINT DEFAULT 0 NOT NULL,
			currency VARCHAR(3) DEFAULT 'XAF' NOT NULL,
    		locked BOOLEAN DEFAULT FALSE,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT fk_wallet_user FOREIGN KEY (user_id)
				REFERENCES users (id)
				ON DELETE CASCADE
				ON UPDATE CASCADE
		);`,
	}
	for _, query := range queries {
		if _, err := DB.Exec(ctx, query); err != nil {
			return err
		}
	}
	return nil
}
