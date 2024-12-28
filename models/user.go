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

// CreateUserWithWallet creates both user and wallet in a single transaction
func (user User) CreateUserWithWallet() (int, *models.User, error) {
	tx := DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var response models.User

	// Create user
	if err := tx.Create(&user.User).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return http.StatusConflict, nil, errors.New("Account already exists")
		}
		return http.StatusInternalServerError, nil, errors.New("Failed to create user account")
	}

	// Get created user
	if err := tx.Select("id", "phone_number", "first_name", "last_name", "photo").First(&response, user.User.ID).Error; err != nil {
		tx.Rollback()
		return http.StatusInternalServerError, nil, errors.New("Failed to retrieve created user")
	}

	// Create wallet
	wallet := models.Wallet{
		UserID:   response.ID,
		Currency: "XAF",
	}
	if err := tx.Create(&wallet).Error; err != nil {
		tx.Rollback()
		return http.StatusInternalServerError, nil, errors.New("Failed to create user wallet")
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return http.StatusInternalServerError, nil, errors.New("Failed to complete user registration")
	}

	return http.StatusCreated, &response, nil
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

// GetWalletByUserID find wallet by user ID
func GetWalletByUserID(userID uint) (*models.Wallet, error) {
	var wallet models.Wallet
	err := DB.Where("user_id = ?", userID).First(&wallet).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("No wallet found")
	}
	return &wallet, nil
}
