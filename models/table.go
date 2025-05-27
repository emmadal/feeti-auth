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
		`CREATE INDEX IF NOT EXISTS idx_users_lookup ON users (phone_number, is_active, quota, locked, premium);`,
	}
	for _, query := range queries {
		if _, err := DB.Exec(ctx, query); err != nil {
			return err
		}
	}
	return nil
}
