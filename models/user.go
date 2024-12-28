package models

import (
	"errors"
	"net/http"
	"time"

	"gorm.io/gorm"
)

// User is the struct for a user
type User struct {
	ID          int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	FirstName   string    `json:"first_name" gorm:"type:varchar(100);not null" binding:"required,alpha"`
	LastName    string    `json:"last_name" gorm:"type:varchar(100);not null" binding:"required,alpha"`
	Email       string    `json:"email" gorm:"type:varchar(200)"`
	PhoneNumber string    `json:"phone_number" gorm:"type:varchar(14);uniqueIndex;unique;not null" binding:"required,e164,min=11,max=14"`
	DeviceToken string    `json:"device_token" gorm:"type:varchar(200);not null" binding:"required,alpha"`
	Pin         string    `json:"pin" gorm:"type:varchar(200);not null" binding:"required,len=4,numeric"`
	Quota       uint      `json:"quota" gorm:"type:bigint;default:0;not null"`
	Locked      bool      `json:"locked" gorm:"type:boolean;default:false"`
	Photo       string    `json:"photo" gorm:"type:text"`
	IsActive    bool      `json:"is_active" gorm:"type:boolean;default:true;index"`
	CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type Wallet struct {
	ID        int64     `json:"id" gorm:"primaryKey;unique"`
	UserID    int64     `json:"user_id" gorm:"type:bigint;not null" binding:"required,number,gt=0"`
	User      User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Balance   uint      `json:"balance" gorm:"type:bigint;default:0;not null" binding:"required,number,min=0,max=10000000"`
	Currency  string    `json:"currency" gorm:"type:varchar(3);default:XOF;not null" binding:"alpha,oneof=XOF GHS XAF GNH EUR USD"`
	IsActive  bool      `json:"is_active" gorm:"type:boolean;default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserLogin is the struct for login
type UserLogin struct {
	PhoneNumber string `json:"phone_number" binding:"required,e164,min=11,max=14"`
	Pin         string `json:"pin" binding:"required,len=4,numeric,min=4,max=4"`
}

// UserResetPin is the struct for resetting the pin
type UserResetPin struct {
	PhoneNumber string `json:"phone_number" binding:"required,e164,min=11,max=14"`
	Pin         string `json:"pin" binding:"required,len=4,numeric,min=4,max=4"`
	CodeOTP     string `json:"code_otp" binding:"required,len=6,numeric,min=6,max=6"`
	KeyUID      string `json:"key_uid" binding:"required,uuid"`
}

type UserUpdatePin struct {
	PhoneNumber string `json:"phone_number" binding:"required,e164,min=11,max=14"`
	OldPin      string `json:"old_pin" binding:"required,len=4,numeric,min=4,max=4"`
	NewPin      string `json:"new_pin" binding:"required,len=4,numeric,min=4,max=4"`
	ConfirmPin  string `json:"confirm_pin" binding:"required,len=4,numeric,min=4,max=4"`
	CodeOTP     string `json:"code_otp" binding:"required,len=6,numeric,min=6,max=6"`
	KeyUID      string `json:"key_uid" binding:"required,uuid"`
}

// UpdateUserPin updates the pin of a user
func (user User) UpdateUserPin() error {
	err := DB.Model(&User{}).Where("phone_number = ? AND is_active = ?", user.PhoneNumber, true).Update("pin", user.Pin).Error

	if err != nil {
		return errors.New("Failed to update user pin")
	}
	return nil
}

// UpdateUserQuota updates the user data
func (user User) UpdateUserQuota() error {
	err := DB.Model(&User{}).Where("phone_number = ? AND is_active = ? AND locked = ?", user.PhoneNumber, true, false).Update("quota", gorm.Expr("quota + ?", 1)).Error
	if err != nil {
		return errors.New("Failed to update login attempts")
	}
	return nil
}

// LockUser locks a user
func (user User) LockUser() error {
	err := DB.Model(&User{}).Where("phone_number = ? AND is_active = ? AND quota = ?", user.PhoneNumber, true, user.Quota).Update("locked", true).Error
	if err != nil {
		return errors.New("Failed to lock account")
	}
	return nil
}

// CreateUser insert user data in database
func (user User) CreateUser() (int, *User, error) {
	var response User
	result := DB.Create(&user).Select("id", "phone_number").First(&response)
	if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
		return http.StatusConflict, nil, errors.New("Account already exist")
	}

	if errors.Is(result.Error, gorm.ErrRegistered) {
		return http.StatusInternalServerError, nil, errors.New("Unable to register user")
	}
	return 0, &user, nil
}

// CreateUser insert user data in database
func (wallet Wallet) CreateUserWallet() error {
	err := DB.Create(&wallet).Error
	if err != nil {
		return errors.New("Failed to create user wallet")
	}

	return nil
}

// GetUserByPhone get a user by phone number
func GetUserByPhone(phone string) (*User, error) {
	var user User
	err := DB.Select("id", "first_name", "last_name", "photo", "phone_number", "pin", "quota", "locked", "device_token").
		Where("phone_number = ? AND is_active = ?", phone, true).
		First(&user).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("No user found")
	}
	return &user, nil
}
