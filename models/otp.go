package models

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type OTP struct {
	ID          uint      `json:"id" gorm:"primaryKey;unique"`
	Code        string    `json:"code" gorm:"type:varchar(6)"`
	IsUsed      bool      `json:"is_used" gorm:"type:boolean;not null;default:false"`
	PhoneNumber string    `json:"phone_number" gorm:"type:varchar(14);not null" binding:"required,e164,min=11,max=14"`
	KeyUID      string    `json:"key_uid" gorm:"type:varchar(100);not null"`
	ExpiryAt    time.Time `json:"expiry_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CheckOTP is the struct for checking the OTP
type CheckOTP struct {
	Code        string    `json:"code" binding:"required,min=6,max=6,numeric"`
	PhoneNumber string    `json:"phone_number" binding:"required,e164,min=11,max=14"`
	KeyUID      string    `json:"key_uid" binding:"required,uuid"`
	ExpiryAt    time.Time `json:"expiry_at"`
	CreatedAt   time.Time `json:"created_at"`
}

// GetUserByPhone get a user by phone number
func (otp OTP) GetUserByPhone() (*User, error) {
	var user User
	err := DB.Select("id", "first_name", "last_name", "photo", "phone_number").
		Where("phone_number = ? AND is_active = ?", otp.PhoneNumber, true).
		First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("No user found")
	}
	return &user, nil
}

// InsertOTP insert a new OTP into the database
func (otp OTP) InsertOTP(expirationTime time.Duration) error {
	result := DB.Create(&OTP{
		Code:        otp.Code,
		PhoneNumber: otp.PhoneNumber,
		KeyUID:      otp.KeyUID,
		ExpiryAt:    time.Now().Add(time.Minute * expirationTime),
	})

	if result.Error != nil {
		return errors.New("Unexpected error with OTP")
	}
	return nil
}

// UpdateOTP update the OTP
func (otp CheckOTP) UpdateOTP() error {
	err := DB.Model(&OTP{}).Where("is_used = ?", false).Where("phone_number = ?", otp.PhoneNumber).Where("key_uid = ?", otp.KeyUID).Where("code = ?", otp.Code).Updates(
		map[string]interface{}{
			"updated_at": time.Now(),
			"is_used":    true,
		},
	).Error
	return err
}

// GetOTP get the OTP by phone number and code
func (ch CheckOTP) GetOTP() (*OTP, error) {
	var otp OTP
	err := DB.Select("is_used", "expiry_at").Where("phone_number = ? AND key_uid = ? AND code = ?", ch.PhoneNumber, ch.KeyUID, ch.Code).First(&otp).Error

	if err != nil {
		return nil, errors.New("OTP not found")
	}
	return &otp, nil
}

// GetOTPByParams get the OTP  by phone number, code adn uuid
func GetOTPByParams(key_uid, phone_number, code string) (*OTP, error) {
	var otp OTP
	err := DB.Select("expiry_at", "is_used", "code", "phone_number", "key_uid").Where("phone_number = ? AND key_uid = ? AND code = ?", phone_number, key_uid, code).First(&otp).Error

	if err != nil {
		return nil, errors.New("No OTP found")
	}
	return &otp, nil
}
