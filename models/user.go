package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserLogin is the struct for user login
type UserLogin struct {
	PhoneNumber string `json:"phone_number" binding:"required,e164,min=11,max=14"`
	Pin         string `json:"pin" binding:"required,len=4,numeric"`
	DeviceToken string `json:"device_token" binding:"required,min=10,max=100"`
}

type Wallet struct {
	ID       int64   `json:"id"`
	Balance  float64 `json:"balance"`
	Currency string  `json:"currency"`
}

// User is the struct for a user
type User struct {
	ID          int64     `json:"id" db:"id,omitempty"`
	FirstName   string    `json:"first_name" db:"first_name" binding:"required,alpha,min=3,max=100"`
	LastName    string    `json:"last_name" db:"last_name" binding:"required,alpha,min=3,max=100"`
	PhoneNumber string    `json:"phone_number" db:"phone_number" binding:"required,e164,min=11,max=14"`
	DeviceToken string    `json:"device_token" db:"device_token" binding:"required,min=10,max=100"`
	Pin         string    `json:"pin" db:"pin" binding:"required,len=4,numeric"`
	Quota       uint      `json:"quota" db:"quota"`
	Locked      bool      `json:"locked" db:"locked"`
	Photo       string    `json:"photo" db:"photo,omitempty"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at,omitempty"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at,omitempty"`
}

// Login is the struct for login
type Login struct {
	PhoneNumber string `json:"phone_number" binding:"required,e164,min=11,max=14"`
	Pin         string `json:"pin" binding:"required,len=4,numeric"`
	DeviceToken string `json:"device_token" binding:"required,min=10,max=100"`
}

// ResetPin is the struct for resetting the pin
type ResetPin struct {
	PhoneNumber string `json:"phone_number" binding:"required,e164,min=11,max=14"`
	Pin         string `json:"pin" binding:"required,len=4,numeric"`
	CodeOTP     string `json:"code_otp" binding:"required,len=5,numeric"`
	KeyUID      string `json:"key_uid" binding:"required,uuid"`
}

// UpdatePin is the struct for updating the pin
type UpdatePin struct {
	PhoneNumber string `json:"phone_number" binding:"required,e164,min=11,max=14"`
	OldPin      string `json:"old_pin" binding:"required,len=4,numeric"`
	NewPin      string `json:"new_pin" binding:"required,len=4,numeric"`
	ConfirmPin  string `json:"confirm_pin" binding:"required,len=4,numeric,eqfield=NewPin"`
}

// RemoveUserAccount is the struct to remove user
type RemoveUserAccount struct {
	PhoneNumber string `json:"phone_number" binding:"required,e164,min=11,max=14"`
	Pin         string `json:"pin" binding:"required,len=4,numeric"`
}

type AuthResponse struct {
	User   UserResponse `json:"user"`
	Wallet Wallet       `json:"wallet"`
}

type UserResponse struct {
	ID          int64  `json:"id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	PhoneNumber string `json:"phone_number"`
	Photo       string `json:"photo"`
	DeviceToken string `json:"device_token"`
}

// UpdateUserPin updates the pin of a user
func (user *User) UpdateUserPin() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	tx, err := DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(
		ctx,
		`UPDATE users SET pin = $1 WHERE phone_number = $2 AND is_active = true AND locked = false AND quota = 0`,
		user.Pin, user.PhoneNumber,
	)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// UpdateUserQuota updates the user data
func (user *User) UpdateUserQuota() error {
	ctx := context.Background()
	_, err := WithTransaction(
		DB, func(tx pgx.Tx) (any, error) {
			_, err := tx.Exec(
				ctx, `UPDATE users SET quota = quota + 1 WHERE phone_number = $1 AND is_active = true 
                AND locked = false AND quota < 3`, user.PhoneNumber,
			)
			return nil, err
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// LockUser locks a user account
func (user *User) LockUser() error {
	ctx := context.Background()
	_, err := WithTransaction(
		DB, func(tx pgx.Tx) (any, error) {
			_, err := tx.Exec(
				ctx, "UPDATE users SET locked = $1 WHERE id = $2 AND is_active = true AND quota >= 3",
				true, user.ID,
			)
			return nil, err
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// ResetUserQuota resets the user quota
func (user *User) ResetUserQuota() error {
	ctx := context.Background()
	_, err := WithTransaction(
		DB, func(tx pgx.Tx) (any, error) {
			_, err := tx.Exec(
				ctx,
				"UPDATE users SET quota = $1 WHERE phone_number = $2 AND is_active = true",
				0, user.PhoneNumber,
			)
			return nil, err
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// CreateUser creates a new user
func (user *User) CreateUser() (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var photo sql.RawBytes

	tx, err := DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var newUser User
	err = tx.QueryRow(
		ctx,
		`INSERT INTO users(first_name, last_name, phone_number, pin, device_token)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING id, first_name, last_name, phone_number, photo, device_token`,
		user.FirstName, user.LastName, user.PhoneNumber, user.Pin, user.DeviceToken,
	).Scan(
		&newUser.ID,
		&newUser.FirstName,
		&newUser.LastName,
		&newUser.PhoneNumber,
		&photo,
		&newUser.DeviceToken,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &newUser, nil
}

func (user *User) RollbackUser() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	tx, err := DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	_, err = tx.Exec(ctx, `DELETE FROM users WHERE id = $1 `, user.ID)
	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// DeactivateUserAccount deactivate the user account
func (user *User) DeactivateUserAccount() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	tx, err := DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx) // no-op if already committed
	}()

	batch := &pgx.Batch{}
	batch.Queue(`UPDATE users SET is_active = $1, locked = $2, quota = $3 WHERE id = $4`, false, true, 3, user.ID)
	batch.Queue(`UPDATE wallets SET is_active = $1, locked = $2 WHERE user_id = $3`, false, true, user.ID)

	batchResults := tx.SendBatch(ctx, batch)

	// Consume all batch results before continuing
	for i := 0; i < 2; i++ {
		if _, err := batchResults.Exec(); err != nil {
			_ = batchResults.Close()
			return err
		}
	}

	// Now it's safe to close
	if err := batchResults.Close(); err != nil {
		return err
	}

	// Then commit transaction
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// GetUserByPhoneNumber find user by phone number
func GetUserByPhoneNumber(phone string) (*User, error) {
	var user User
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var photo sql.RawBytes
	err := DB.QueryRow(
		ctx,
		`SELECT id, first_name, last_name, phone_number, pin, device_token, photo
            FROM users WHERE phone_number = $1 AND is_active = $2`, phone, true,
	).Scan(
		&user.ID, &user.FirstName, &user.LastName, &user.PhoneNumber, &user.Pin, &user.DeviceToken,
		&photo,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, pgx.ErrNoRows
		}
		return nil, err
	}
	return &user, nil
}

// CheckUserByPhone verify if a phone number exists
func (user *User) CheckUserByPhone() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var id int
	err := DB.QueryRow(
		ctx,
		`SELECT id FROM users WHERE phone_number = $1 AND is_active = true`,
		user.PhoneNumber,
	).Scan(&id)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			return false // No user found
		}
		return false // Query failed for some reason
	}

	return true // User found
}

// GetUserByPhone find user by phone number
func (user *User) GetUserByPhone() (*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var photo sql.RawBytes
	err := DB.QueryRow(
		ctx,
		`SELECT id, first_name, last_name, phone_number, device_token, pin, quota, locked, photo
         FROM users WHERE phone_number = $1 AND is_active = true`,
		user.PhoneNumber,
	).Scan(
		&user.ID, &user.FirstName, &user.LastName, &user.PhoneNumber, &user.DeviceToken, &user.Pin, &user.Quota,
		&user.Locked, &photo,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return user, nil
}

// UpdateDeviceToken update user device token
func (user *User) UpdateDeviceToken() error {
	ctx := context.Background()
	_, err := WithTransaction(
		DB, func(tx pgx.Tx) (any, error) {
			_, err := tx.Exec(
				ctx, "UPDATE users SET device_token = $1 WHERE phone_number = $2 AND is_active = true",
				user.DeviceToken, user.PhoneNumber,
			)
			return nil, err
		},
	)
	if err != nil {
		return err
	}
	return nil
}

// WithTransaction is a function to create a transaction
func WithTransaction[T any](conn *pgxpool.Pool, fn func(tx pgx.Tx) (T, error)) (T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var zero T

	tx, err := conn.Begin(ctx)
	if err != nil {
		return zero, err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	result, err := fn(tx)
	if err != nil {
		return zero, err
	}

	err = tx.Commit(ctx)
	return result, err
}
