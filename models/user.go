package models

import (
	"errors"
	"net/http"

	"github.com/emmadal/feeti-module/models"
	"gorm.io/gorm"
)

// User is a local type that embeds the non-local models.User type
type User struct {
	models.User
}

// Wallet is a local type that embeds the non-local models.Wallet type
type Wallet struct {
	models.Wallet
}

// UserLogin is a local type that embeds the non-local models.Login type
type UserLogin struct {
	models.Login
}

// UpdateUserPin updates the pin of a user
func (user User) UpdateUserPin() error {
	err := DB.Model(&models.User{}).Where("phone_number = ? AND is_active = ?", user.PhoneNumber, true).Update("pin", user.Pin).Error

	if err != nil {
		return errors.New("Failed to update user pin")
	}
	return nil
}

// UpdateUserQuota updates the user data
func (user User) UpdateUserQuota() error {
	err := DB.Model(&models.User{}).Where("phone_number = ? AND is_active = ? AND locked = ?", user.PhoneNumber, true, false).Update("quota", gorm.Expr("quota + ?", 1)).Error
	if err != nil {
		return errors.New("Failed to update login attempts")
	}
	return nil
}

// LockUser locks a user
func (user User) LockUser() error {
	err := DB.Model(&models.User{}).Where("phone_number = ? AND is_active = ? AND quota = ?", user.PhoneNumber, true, user.Quota).Update("locked", true).Error
	if err != nil {
		return errors.New("Failed to lock account")
	}
	return nil
}

// CreateUser insert user data in database
func (user User) CreateUser() (int, *models.User, error) {
	var response models.User
	result := DB.Create(&user.User).Select("id", "phone_number").First(&response)
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return http.StatusConflict, nil, errors.New("Account already exist")
	}

	if errors.Is(result.Error, gorm.ErrRegistered) {
		return http.StatusInternalServerError, nil, errors.New("Unable to register user")
	}
	return 0, &user.User, nil
}

// CreateUser insert user data in database
func (w *Wallet) CreateUserWallet() error {
	err := DB.Create(&w).Error
	if err != nil {
		return errors.New("Failed to create user wallet")
	}

	return nil
}

// GetUserByPhoneNumber find user by phone number
func GetUserByPhoneNumber(phone string) (*models.User, error) {
	var user models.User
	err := DB.Select("id", "first_name", "last_name", "photo", "phone_number", "pin", "quota", "locked", "device_token").
		Where("phone_number = ? AND is_active = ?", phone, true).
		First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("No user found")
	}
	return &user, nil
}
