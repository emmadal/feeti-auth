package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/emmadal/feeti-module/models"
	"gorm.io/gorm"
)

// OTP is a local type that embeds the non-local models.Otp type
type OTP struct {
	models.Otp
}

// NewOTP is a local type that embeds the non-local models.NewOtp type
type NewOTP struct {
	models.NewOtp
}

// CheckOTP is a local type that embeds the non-local models.CheckOtp type
type CheckOTP struct {
	models.CheckOtp
}

// ResetPin is a local type that embeds the non-local models.ResetPin type
type ResetPin struct {
	models.ResetPin
}

// UpdatePin is a local type that embeds the non-local models.UpdatePin type
type UpdatePin struct {
	models.UpdatePin
}

// GetUserByPhone get a user by phone number
func (otp OTP) GetUserByPhone() (*models.User, error) {
	var user models.User
	err := DB.Select("id", "first_name", "last_name", "photo", "phone_number").
		Where("phone_number = ? AND is_active = ?", otp.PhoneNumber, true).
		First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("No user found")
	}
	return &user, nil
}

// InsertOTP insert a new OTP into the database
func (otp OTP) InsertOTP() error {
	DB.Transaction(func(tx *gorm.DB) error {
		// do some database operations in the transaction (use 'tx' from this point)
		otp.ExpiryAt = time.Now().Add(2 * time.Minute)
		if err := tx.Create(&otp.Otp).Error; err != nil {
			// return any error will rollback
			return fmt.Errorf("Failed to create OTP")
		}
		// return nil will commit the whole transaction
		return nil
	})
	return nil
}

// UpdateOTP update the OTP
func (otp OTP) UpdateOTP() error {
	DB.Transaction(func(tx *gorm.DB) error {
		// do some database operations in the transaction
		err := tx.Model(&otp).Where("is_used = ? AND phone_number = ? AND key_uid = ? AND code = ?", false, otp.PhoneNumber, otp.KeyUID, otp.Code).Update("is_used", true).Error

		if err != nil {
			// return any error will rollback
			return fmt.Errorf("Failed to confirm OTP")
		}
		// return nil will commit the whole transaction
		return nil
	})
	return nil
}

// GetOTP find the OTP in the database
func (ch CheckOTP) GetOTP() (*OTP, error) {
	var otp OTP
	err := DB.Select("expiry_at", "is_used", "code", "phone_number", "key_uid").Where("phone_number = ? AND key_uid = ? AND code = ?", ch.PhoneNumber, ch.KeyUID, ch.Code).First(&otp).Error

	if err != nil {
		return nil, fmt.Errorf("No OTP found")
	}
	return &otp, nil
}

// GetOTPByCodeAndUID find OTP by code and uid
func GetOTPByCodeAndUID(phone, code, uid string) (*OTP, error) {
	var otp OTP
	err := DB.Select("expiry_at", "is_used", "code", "phone_number", "key_uid").Where("phone_number = ? AND key_uid = ? AND code = ?", phone, uid, code).First(&otp).Error

	if err != nil {
		return nil, fmt.Errorf("No OTP found")
	}
	return &otp, nil
}
