package models

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"time"
)

// Otp is the struct for OTP in the database
type Otp struct {
	ID          int64     `json:"id" db:"id,omitempty"`
	Code        string    `json:"code" db:"code" binding:"required,len=5,numeric"`
	IsUsed      bool      `json:"is_used" db:"is_used"`
	PhoneNumber string    `json:"phone_number" db:"phone_number" binding:"required,e164,min=11,max=14"`
	KeyUID      string    `json:"key_uid" db:"key_uid" binding:"required,uuid"`
	ExpiryAt    time.Time `json:"expiry_at" db:"expiry_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"created_at"`
}

// CheckOtp is the struct for checking the OTP
type CheckOtp struct {
	Code        string    `json:"code" binding:"required,len=5,numeric"`
	PhoneNumber string    `json:"phone_number" binding:"required,e164,min=11,max=14"`
	KeyUID      string    `json:"key_uid" binding:"required,uuid"`
	ExpiryAt    time.Time `json:"expiry_at"`
}

// NewOtp is the struct for creating a new OTP
type NewOtp struct {
	PhoneNumber string `json:"phone_number" binding:"required,e164,min=11,max=14"`
}

// InsertOTP insert a new OTP into the database
func (otp *Otp) InsertOTP() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	tx, err := DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(
		ctx,
		`INSERT INTO otp(code, phone_number, key_uid, expiry_at) VALUES ($1, $2, $3, $4)`, otp.Code, otp.PhoneNumber,
		otp.KeyUID, otp.ExpiryAt,
	); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// UpdateOTP update the OTP
func (otp *Otp) UpdateOTP() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	tx, err := DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = DB.Exec(
		ctx, `UPDATE otp SET is_used = $1 WHERE code = $2 AND key_uid = $3 AND phone_number = $4`,
		true, otp.Code, otp.KeyUID, otp.PhoneNumber,
	)

	if err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// GetOTP find the OTP in the database
func (otp *Otp) GetOTP() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := DB.QueryRow(
		ctx, "SELECT expiry_at, is_used FROM otp WHERE code = $1 AND phone_number = $2 AND key_uid = $3", otp.Code,
		otp.PhoneNumber, otp.KeyUID,
	).Scan(&otp.ExpiryAt, &otp.IsUsed)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}
		return err
	}
	return nil
}
