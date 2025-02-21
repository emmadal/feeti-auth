package models

import (
	"errors"
	"fmt"

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
	DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&User{}).Where("phone_number = ? AND is_active = ? AND locked = ? AND quota = ?", user.PhoneNumber, true, false, 0).Update("pin", user.Pin).Error

		if err != nil {
			// return any error will rollback
			return fmt.Errorf("Failed to update user pin")
		}
		// return nil will commit the whole transaction
		return nil
	})
	return nil
}

// UpdateUserQuota updates the user data
func (user User) UpdateUserQuota() error {
	DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&User{}).Where("phone_number = ? AND is_active = ? AND locked = ? AND quota < ?", user.PhoneNumber, true, false, 3).Update("quota", gorm.Expr("quota + ?", 1)).Error
		if err != nil {
			// return any error will rollback
			return fmt.Errorf("Failed to update quota")
		}
		return nil
	})
	return nil
}

// LockUser locks a user account
func (user User) LockUser() error {
	DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&User{}).Where("phone_number = ? AND is_active = ? AND quota >= ?", user.PhoneNumber, true, 3).Update("locked", true).Error

		if err != nil {
			// return any error will rollback
			return fmt.Errorf("Failed to lock account")
		}
		// return nil will commit the whole transaction
		return nil
	})
	return nil
}

// ResetUserQuota resets the user quota
func (user User) ResetUserQuota() error {
	DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&User{}).Where("phone_number = ? AND is_active = ? AND locked = ?", user.PhoneNumber, true, false).Update("quota", 0).Error

		if err != nil {
			// return any error will rollback
			return fmt.Errorf("Failed to reset login attempts")
		}

		// return nil will commit the whole transaction
		return nil
	})
	return nil
}

// CreateUser creates a new user
func (user User) CreateUser() (*User, *Wallet, error) {
	var createdUser User
	var wallet Wallet
	DB.Transaction(func(tx *gorm.DB) error {
		// Create the user
		if err := tx.Create(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return fmt.Errorf("phone number already registered")
			}
			return fmt.Errorf("database error: %v", err)
		}

		// Retrieve the created user within the same transaction
		if err := tx.Model(&User{}).Where("id = ?", user.ID).First(&createdUser).Error; err != nil {
			return fmt.Errorf("failed to retrieve created user: %v", err)
		}

		// create wallet for the user
		wallet.UserID = createdUser.ID
		if err := tx.Create(&wallet).Error; err != nil {
			return fmt.Errorf("failed to create wallet: %v", err)
		}

		// retrieve created wallet
		if err := tx.Select("id", "currency", "balance").First(&wallet, "user_id = ?", createdUser.ID).Error; err != nil {
			return fmt.Errorf("failed to retrieve created wallet: %v", err)
		}

		// return nil will commit the whole transaction
		return nil
	})

	return &createdUser, &wallet, nil
}

// GetUserByPhoneNumber find user by phone number
func GetUserByPhoneNumber(phone string) (*User, error) {
	var user User
	err := DB.Select("id", "first_name", "last_name", "photo", "phone_number", "pin", "quota", "locked", "device_token", "email", "face_id", "finger_print").
		Where("phone_number = ? AND is_active = ? AND locked = ?", phone, true, false).
		First(&user).Error

	if err != nil {
		return nil, fmt.Errorf("No user found")
	}
	return &user, nil
}

// CheckUserByPhone verify if a phone number exist
func CheckUserByPhone(phone string) bool {
	var user User
	err := DB.Select("id", "phone_number").
		Where("phone_number = ?", phone).
		First(&user).Error

	if err != nil {
		return false
	}
	return true
}

// GetUserAndWalletByPhone find user and wallet by phone number
func GetUserAndWalletByPhone(phone string) (*User, *Wallet, error) {
	var (
		user   User
		wallet Wallet
	)

	err := DB.Transaction(func(tx *gorm.DB) error {
		// Fetch user data
		if err := tx.Select("id", "first_name", "last_name", "photo", "phone_number", "email", "pin", "quota", "locked", "device_token", "face_id", "finger_print").
			Where("phone_number = ? AND is_active = ? AND locked = ?", phone, true, false).
			First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("user not found")
			}
			return fmt.Errorf("failed to fetch user")
		}

		// Fetch wallet data
		if err := tx.Select("id", "currency", "balance").
			Where("user_id = ?", user.ID).
			First(&wallet).Error; err != nil {
			return fmt.Errorf("failed to fetch wallet")
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return &user, &wallet, nil
}

// UpdateDeviceToken update user device token
func (user User) UpdateDeviceToken() error {
	DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&User{}).Where("phone_number = ? AND is_active = ?", user.PhoneNumber, true).Update("device_token", user.DeviceToken).Error
		if err != nil {
			// return any error will rollback
			return fmt.Errorf("Failed to update device uid")
		}
		// return nil will commit the whole transaction
		return nil
	})
	return nil
}
