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

// LockUser locks a user account
func (user User) LockUser() error {
	tx := DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Model(&models.User{}).Where("phone_number = ? AND is_active = ? AND quota = ?", user.PhoneNumber, true, 3).Update("locked", true).Error; err != nil {
		tx.Rollback()
		return errors.New("Failed to lock account due to an error")
	}

	if err := tx.Commit().Error; err != nil {
		return errors.New("Failed to lock account")
	}

	return nil
}

// ResetUserQuota resets the user quota
func (user User) ResetUserQuota() error {
	err := DB.Model(&models.User{}).Where("phone_number = ? AND is_active = ? AND locked = ?", user.PhoneNumber, true, false).Update("quota", 0).Error
	if err != nil {
		return errors.New("Failed to reset login attempts")
	}
	return nil
}

// CreateUserWithWallet creates both user and wallet in a single transaction
func (user User) CreateUserWithWallet() (int, *models.User, *models.Wallet, error) {
	tx := DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var response models.User
	var response_wallet models.Wallet

	// Create user
	if err := tx.Create(&user.User).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return http.StatusConflict, nil, nil, errors.New("Account already exists")
		}
		return http.StatusInternalServerError, nil, nil, errors.New("Failed to create user account")
	}

	// Get created user
	if err := tx.Select("id", "phone_number", "first_name", "last_name", "photo", "email", "device_token").First(&response, user.User.ID).Error; err != nil {
		tx.Rollback()
		return http.StatusInternalServerError, nil, nil, errors.New("Failed to retrieve created user")
	}

	// Create wallet
	wallet := models.Wallet{
		UserID:   response.ID,
		Currency: "XAF",
	}

	if err := tx.Create(&wallet).Error; err != nil {
		tx.Rollback()
		return http.StatusInternalServerError, nil, nil, errors.New("Failed to create user wallet")
	}

	// Get created wallet
	if err := tx.Select("id", "user_id", "currency", "balance").First(&response_wallet, wallet.ID).Error; err != nil {
		tx.Rollback()
		return http.StatusInternalServerError, nil, nil, errors.New("Failed to retrieve created wallet")
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return http.StatusInternalServerError, nil, nil, errors.New("Failed to complete user registration")
	}

	return http.StatusCreated, &response, &response_wallet, nil
}

// GetUserByPhoneNumber find user by phone number
func GetUserByPhoneNumber(phone string) (*models.User, error) {
	var user models.User
	err := DB.Select("id", "first_name", "last_name", "photo", "phone_number", "pin", "quota", "locked", "device_token", "email").
		Where("phone_number = ? AND is_active = ?", phone, true).
		First(&user).Error

	if err != nil {
		return nil, errors.New("No user found")
	}
	return &user, nil
}

// GetWalletByUserID find wallet by user ID
func GetWalletByUserID(userID int64) (*models.Wallet, error) {
	var wallet models.Wallet
	err := DB.Select("id", "user_id", "currency", "balance").Where("user_id = ? AND is_active = ?", userID, true).First(&wallet).Error
	if err != nil {
		return nil, errors.New("No wallet found")
	}
	return &wallet, nil
}
