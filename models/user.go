package models

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// UserLogin is the struct for user login
type UserLogin struct {
	PhoneNumber string `json:"phone_number" binding:"required,e164,min=11,max=14"`
	Pin         string `json:"pin" binding:"required,len=4,numeric,min=4,max=4"`
	DeviceToken string `json:"device_token" binding:"required,min=10,max=100"`
}

// User is the struct for a user
type User struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	FirstName   string    `json:"first_name" gorm:"type:varchar(150);not null" binding:"required,alpha,min=3,max=150"`
	LastName    string    `json:"last_name" gorm:"type:varchar(150);not null" binding:"required,alpha,min=3,max=150"`
	Email       string    `json:"email" gorm:"type:varchar(150)"`
	PhoneNumber string    `json:"phone_number" gorm:"type:varchar(15);uniqueIndex;not null" binding:"required,e164,min=11,max=14"`
	DeviceToken string    `json:"device_token" gorm:"type:varchar(150);not null" binding:"required,min=10,max=100"`
	Pin         string    `json:"pin" gorm:"type:varchar(150);not null" binding:"required,len=4,numeric,min=4,max=4"`
	Quota       uint      `json:"quota" gorm:"type:bigint;default:0;not null"`
	Locked      bool      `json:"locked" gorm:"type:boolean;default:false;not null"`
	FaceID      bool      `json:"face_id" gorm:"type:boolean;default:false;not null"`
	Premium     bool      `json:"premium" gorm:"type:boolean;default:false;not null"`
	FingerPrint bool      `json:"finger_print" gorm:"type:boolean;default:false;not null"`
	Photo       string    `json:"photo" gorm:"type:varchar(250)"`
	IsActive    bool      `json:"is_active" gorm:"type:boolean;default:true;index;not null"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// Wallet is the struct for a wallet
type Wallet struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    int64     `json:"user_id" gorm:"type:bigint;not null;index" binding:"required,number,gt=0"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Balance   int64     `json:"balance" gorm:"type:bigint;default:0;not null"`
	Currency  string    `json:"currency" gorm:"type:varchar(3);default:XAF;not null" binding:"alpha,oneof=XOF XAF"`
	IsActive  bool      `json:"is_active" gorm:"type:boolean;default:true"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// Login is the struct for login
type Login struct {
	PhoneNumber string `json:"phone_number" binding:"required,e164,min=11,max=14"`
	Pin         string `json:"pin" binding:"required,len=4,numeric,min=4,max=4"`
	DeviceToken string `json:"device_token" binding:"required,min=10,max=100"`
}

// ResetPin is the struct for resetting the pin
type ResetPin struct {
	PhoneNumber string `json:"phone_number" binding:"required,e164,min=11,max=14"`
	Pin         string `json:"pin" binding:"required,len=4,numeric,min=4,max=4"`
	CodeOTP     string `json:"code_otp" binding:"required,len=5,numeric,min=5,max=5"`
	KeyUID      string `json:"key_uid" binding:"required,uuid"`
}

// UpdatePin is the struct for updating the pin
type UpdatePin struct {
	PhoneNumber string `json:"phone_number" binding:"required,e164,min=11,max=14"`
	OldPin      string `json:"old_pin" binding:"required,len=4,numeric,min=4,max=4"`
	NewPin      string `json:"new_pin" binding:"required,len=4,numeric,min=4,max=4"`
	CodeOTP     string `json:"code_otp" binding:"required,len=5,numeric,min=5,max=5"`
	KeyUID      string `json:"key_uid" binding:"required,uuid"`
}

// UpdateProfile is the struct for updating the profile
type Profile struct {
	PhoneNumber string `json:"phone_number" binding:"required,e164,min=11,max=14"`
	Email       string `json:"email"`
	Photo       string `json:"photo"`
	FaceID      bool   `json:"face_id"`
	FingerPrint bool   `json:"finger_print"`
}

// RemoveAccount is the struct for removing the account
type RemoveProfile struct {
	PhoneNumber string `json:"phone_number" binding:"required,e164,min=11,max=14"`
	Pin         string `json:"pin" binding:"required,len=4,numeric,min=4,max=4"`
	CodeOTP     string `json:"code_otp" binding:"required,len=5,numeric,min=5,max=5"`
	KeyUID      string `json:"key_uid" binding:"required,uuid"`
}

// UpdateUserPin updates the pin of a user
func (user User) UpdateUserPin(ctx context.Context) error {
	err := DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&User{}).Where("phone_number = ? AND is_active = ? AND locked = ? AND quota = ?", user.PhoneNumber, true, false, 0).Update("pin", user.Pin).Error

		if err != nil {
			// return any error will rollback
			return fmt.Errorf("Failed to update user pin")
		}
		// return nil will commit the whole transaction
		return nil
	})
	// Return the transaction error
	if err != nil {
		logrus.WithFields(logrus.Fields{"error": err}).Error(err)
		return fmt.Errorf("Something went wrong while updating user pin")
	}
	return nil
}

// UpdateUserQuota updates the user data
func (user User) UpdateUserQuota(ctx context.Context) error {
	err := DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&User{}).Where("phone_number = ? AND is_active = ? AND locked = ? AND quota < ?", user.PhoneNumber, true, false, 3).Update("quota", gorm.Expr("quota + ?", 1)).Error
		if err != nil {
			// return any error will rollback
			return fmt.Errorf("Failed to update quota")
		}
		// return nil will commit the whole transaction
		return nil
	})
	// Return the transaction error
	if err != nil {
		logrus.WithFields(logrus.Fields{"error": err}).Error(err)
		return fmt.Errorf("Something went wrong while updating user quota")
	}
	return nil
}

// LockUser locks a user account
func (user User) LockUser(ctx context.Context) error {
	err := DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock user account
		if err := tx.Model(&User{}).Where("phone_number = ? AND is_active = ? AND quota >= ?", user.PhoneNumber, true, 3).Update("locked", true).Error; err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error(err)
			return fmt.Errorf("Failed to lock account")
		}

		// Lock user wallet
		if err := tx.Model(&Wallet{}).Where("user_id = ? AND is_active = ?", user.ID, true).Update("is_active", false).Error; err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error(err)
			return fmt.Errorf("Unable to lock user wallet")
		}
		// return nil will commit the whole transaction
		return nil
	})
	// Return the transaction error
	if err != nil {
		logrus.WithFields(logrus.Fields{"error": err}).Error(err)
		return fmt.Errorf("Something went wrong while locking user account")
	}
	return nil
}

// ResetUserQuota resets the user quota
func (user User) ResetUserQuota(ctx context.Context) error {
	err := DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&User{}).Where("phone_number = ? AND is_active = ? AND locked = ?", user.PhoneNumber, true, false).Update("quota", 0).Error

		if err != nil {
			// return any error will rollback
			return fmt.Errorf("Failed to reset login attempts")
		}

		// return nil will commit the whole transaction
		return nil
	})
	// Return the transaction error
	if err != nil {
		logrus.WithFields(logrus.Fields{"error": err}).Error(err)
		return fmt.Errorf("Something went wrong while resetting user quota")
	}
	return nil
}

// CreateUser creates a new user
func (user User) CreateUser(ctx context.Context) (*User, *Wallet, error) {
	var createdUser User
	var wallet Wallet
	err := DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create the user
		if err := tx.Create(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return fmt.Errorf("phone number already registered")
			}
			logrus.WithFields(logrus.Fields{"error": err}).Error(err)
			return fmt.Errorf("failed to create user")
		}

		// Retrieve the created user within the same transaction
		if err := tx.Model(&User{}).Where("id = ?", user.ID).First(&createdUser).Error; err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error(err)
			return fmt.Errorf("failed to retrieve created user")
		}

		// create wallet for the user
		wallet.UserID = createdUser.ID
		if err := tx.Create(&wallet).Error; err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error(err)
			return fmt.Errorf("failed to create wallet")
		}

		// retrieve created wallet
		if err := tx.Select("id", "currency", "balance", "is_active").First(&wallet, "user_id = ?", createdUser.ID).Error; err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error(err)
			return fmt.Errorf("failed to retrieve created wallet")
		}

		// return nil will commit the whole transaction
		return nil
	})

	if err != nil {
		// Return the transaction error
		logrus.WithFields(logrus.Fields{"error": err}).Error(err)
		return nil, nil, fmt.Errorf("Something went wrong while creating user account")
	}
	return &createdUser, &wallet, nil
}

// CreateUser creates a new user
func (user User) RemoveUser(ctx context.Context) error {
	err := DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		err := tx.Model(&user).Where("phone_number = ? AND is_active = ?", user.PhoneNumber, true).Select("IsActive", "Locked", "Quota").Updates(User{IsActive: false, Locked: true, Quota: 3}).Error
		if err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error(err)
			return fmt.Errorf("Unable to remove user account")
		}

		if err := tx.Model(&Wallet{}).Where("user_id = ? AND is_active = ?", user.ID, true).Update("is_active", false).Error; err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error(err)
			return fmt.Errorf("Unable to remove user wallet")
		}

		// return nil will commit the whole transaction
		return nil
	})

	// Return the transaction error
	if err != nil {
		logrus.WithFields(logrus.Fields{"error": err}).Error(err)
		return fmt.Errorf("Something went wrong while removing user account")
	}

	return nil
}

// GetUserByPhoneNumber find user by phone number
func GetUserByPhoneNumber(ctx context.Context, phone string) (*User, error) {
	var user User
	err := DB.WithContext(ctx).Select("id", "first_name", "last_name", "photo", "phone_number", "pin", "quota", "locked", "device_token", "face_id", "finger_print", "premium", "is_active").
		Where("phone_number = ? AND is_active = ? AND locked = ?", phone, true, false).
		First(&user).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("No user found")
		}
		return nil, fmt.Errorf("Unable to retrieve user data")
	}
	return &user, nil
}

// CheckUserByPhone verify if a phone number exist
func CheckUserByPhone(ctx context.Context, phone string) bool {
	var user User
	err := DB.WithContext(ctx).Where("phone_number = ?", phone).First(&user).Error

	return err == nil
}

// GetUserAndWalletByPhone find user and wallet by phone number
func GetUserAndWalletByPhone(ctx context.Context, phone string) (*User, *Wallet, error) {
	var (
		user   User
		wallet Wallet
	)

	err := DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Fetch user data
		if err := tx.Select("id", "first_name", "last_name", "photo", "phone_number", "pin", "quota", "locked", "device_token", "face_id", "finger_print", "premium", "is_active").
			Where("phone_number = ? AND is_active = ?", phone, true).
			First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("user not found")
			}
			return fmt.Errorf("failed to fetch user")
		}

		// Fetch wallet data
		if err := tx.Select("id", "currency", "balance", "is_active").
			Where("user_id = ? AND is_active = ?", user.ID, true).
			First(&wallet).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("wallet not found")
			}
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
func (user User) UpdateDeviceToken(ctx context.Context) error {
	err := DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&User{}).Where("phone_number = ? AND is_active = ?", user.PhoneNumber, true).Update("device_token", user.DeviceToken).Error
		if err != nil {
			// return any error will rollback
			return fmt.Errorf("Failed to update device token")
		}
		// return nil will commit the whole transaction
		return nil
	})
	if err != nil {
		// Return the transaction error
		logrus.WithFields(logrus.Fields{"error": err}).Error(err)
		return fmt.Errorf("Something went wrong while updating user device token")
	}
	return nil
}

// UpdateProfile updates user profile
func (user Profile) UpdateProfile(ctx context.Context) (*User, error) {
	var updatedUser User

	// Start the transaction
	err := DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Find the user and lock for update
		if err := tx.Where("phone_number = ? AND is_active = ?", user.PhoneNumber, true).
			First(&updatedUser).Error; err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error(err)
			return fmt.Errorf("User not found or inactive")
		}

		// Update only specific fields
		err := tx.Model(&updatedUser).Select("email", "photo", "face_id", "finger_print").Updates(user).Error
		if err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error(err)
			return fmt.Errorf("Failed to update user profile")
		}
		// return nil will commit the whole transaction
		return nil
	})

	if err != nil {
		// Return the transaction error
		logrus.WithFields(logrus.Fields{"error": err}).Error(err)
		return nil, fmt.Errorf("Something went wrong while updating user profile")
	}

	// Return updated user
	return &updatedUser, nil
}
