package models

import (
	"errors"
	"fmt"

	"github.com/emmadal/feeti-module/models"
	"gorm.io/gorm"
)

// OTP is a local type that embeds the non-local models.Otp type
type OTP struct {
	models.Otp
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
	err := DB.Create(&otp.Otp).Error
	if err != nil {
		return fmt.Errorf("Failed to create OTP")
	}
	return nil
}

// UpdateOTP update the OTP
func (otp OTP) UpdateOTP() error {
	err := DB.Model(&otp).Where("is_used = ? AND phone_number = ? AND key_uid = ? AND code = ?", false, otp.PhoneNumber, otp.KeyUID, otp.Code).Update("is_used", true).Error
	if err != nil {
		return fmt.Errorf("Failed to confirm OTP")
	}
	return nil
}

// GetOTP find the OTP in the database
func (ch CheckOTP) GetOTP() (*OTP, error) {
	var otp OTP
	err := DB.Select("expiry_at", "is_used", "code", "phone_number", "key_uid").Where("phone_number = ? AND key_uid = ? AND code = ?", ch.PhoneNumber, ch.KeyUID, ch.Code).First(&otp).Error

	if err != nil {
		return nil, errors.New("No OTP found")
	}
	return &otp, nil
}

// GetOTPByCodeAndUID find OTP by code and uid
func GetOTPByCodeAndUID(phone, code, uid string) (*OTP, error) {
	var otp OTP
	err := DB.Select("expiry_at", "is_used", "code", "phone_number", "key_uid").Where("phone_number = ? AND key_uid = ? AND code = ?", phone, uid, code).First(&otp).Error

	if err != nil {
		return nil, errors.New("No OTP found")
	}
	return &otp, nil
}
