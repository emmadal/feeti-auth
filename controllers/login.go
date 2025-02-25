package controllers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/emmadal/feeti-module/cache"
	jwt "github.com/emmadal/feeti-module/jwt_module"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Login handler to sign in user
func Login(c *gin.Context) {
	var body models.UserLogin

	// recover from panic
	defer func() {
		if r := recover(); r != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Internal server error", fmt.Errorf("recovered: %v", r))
			return
		}
	}()

	// Bind the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	// Try to get user and wallet from cache first
	cachedUser, cachedWallet, err := getCacheData(c, body.PhoneNumber)

	// If user and wallet found in cache
	if err == nil {
		if cachedUser.DeviceToken != body.DeviceToken {
			cachedUser.DeviceToken = body.DeviceToken
		}
		// Check if account is locked or has exceeded quota
		if cachedUser.Locked && cachedUser.Quota >= 3 {
			helpers.HandleError(c, http.StatusUnauthorized, "Account locked. Login attempts exceeded", nil)
			return
		} else {
			// Verify PIN
			if ok := helpers.VerifyPassword(body.Pin, cachedUser.Pin); !ok {
				handleFailedLogin(c, cachedUser)
				return
			} else {
				// send success response
				handleSuccessfulLogin(c, cachedUser, cachedWallet)
				return
			}
		}
	}

	// query datbase if user is not found in the cache
	user, wallet, err := models.GetUserAndWalletByPhone(body.PhoneNumber)
	if err != nil {
		helpers.HandleError(c, http.StatusNotFound, err.Error(), err)
		return
	}

	// Verify PIN
	if ok := helpers.VerifyPassword(body.Pin, user.Pin); !ok {
		handleFailedLogin(c, user)
		return
	}

	// send success response
	if body.DeviceToken != user.DeviceToken {
		user.DeviceToken = body.DeviceToken
	}
	handleSuccessfulLogin(c, user, wallet)
}

// handleFailedLogin handles failed login
func handleFailedLogin(c *gin.Context, user *models.User) {
	userKey, _ := getCacheKeys(user.PhoneNumber)

	// Update login attempts in database and cache
	if user.Quota < 3 {
		if err := user.UpdateUserQuota(); err != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Failed to update login attempts", err)
		}
		user.Quota += 1
		go func() {
			_ = cache.SetRedisData(c, userKey, user, 0)
		}()
		helpers.HandleError(c, http.StatusForbidden, "Invalid credentials", nil)
		return
	}

	// Lock account if quota exceeded
	if user.Quota >= 3 && !user.Locked {
		if err := user.LockUser(); err != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Internal server error", err)
			return
		}

		// Send SMS to user account locked and update cache
		user.Locked = true
		go func() {
			_ = cache.SetRedisData(c, userKey, user, 0)
		}()
		go helpers.SmsAccountLocked(user.PhoneNumber)

		helpers.HandleError(c, http.StatusUnauthorized, "Account locked", nil)
		return
	}

	// Check if account is locked and quota is exceeded
	if user.Locked && user.Quota >= 3 {
		helpers.HandleError(c, http.StatusUnauthorized, "Your feeti account is locked. Please contact the helpdesk", nil)
		return
	}
	// send error response
	helpers.HandleError(c, http.StatusUnauthorized, "Invalid credentials", nil)
}

// handleSuccessfulLogin handles successful login
func handleSuccessfulLogin(c *gin.Context, user *models.User, wallet *models.Wallet) {
	userKey, walletKey := getCacheKeys(user.PhoneNumber)

	// Generate JWT token
	token, err := jwt.GenerateToken(user.ID, []byte(os.Getenv("JWT_KEY")))
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Authentication failed", err)
		return
	}

	// Reset quota asynchronously if needed
	if user.Quota > 0 && !user.Locked {
		go func() {
			if err := user.ResetUserQuota(); err != nil {
				// Log error but don't fail the request
				logrus.WithFields(logrus.Fields{"error": err}).Error("Failed to reset quota")
			}
		}()
	}

	// Cache user and wallet asynchronously
	user.Quota = 0
	go func() {
		_ = cache.SetRedisData(c, userKey, user, 0)
		_ = cache.SetRedisData(c, walletKey, wallet, 0)
		_ = user.UpdateDeviceToken()
	}()

	// Send success response
	helpers.HandleSuccessData(c, "Login successful", map[string]interface{}{
		"token": token,
		"user": gin.H{
			"id":           user.ID,
			"phone_number": user.PhoneNumber,
			"first_name":   user.FirstName,
			"last_name":    user.LastName,
			"photo":        user.Photo,
			"email":        user.Email,
			"face_id":      user.FaceID,
			"finger_print": user.FingerPrint,
			"premium":      user.Premium,
		},
		"wallet": gin.H{
			"id":       wallet.ID,
			"currency": wallet.Currency,
			"balance":  wallet.Balance,
		},
	})
}

// getCacheData retrieves user and wallet data from cache concurrently
func getCacheData(c *gin.Context, phoneNumber string) (*models.User, *models.Wallet, error) {
	cacheKey := fmt.Sprintf("user:%s", phoneNumber)
	walletKey := fmt.Sprintf("wallet:%s", phoneNumber)

	type result struct {
		data interface{}
		err  error
	}

	userCh := make(chan result, 1)
	walletCh := make(chan result, 1)

	// Fetch user data
	go func() {
		user, err := cache.GetRedisData[models.User](c, cacheKey)
		userCh <- result{data: user, err: err}
	}()

	// Fetch wallet data
	go func() {
		wallet, err := cache.GetRedisData[models.Wallet](c, walletKey)
		walletCh <- result{data: wallet, err: err}
	}()

	// Get results
	userResult := <-userCh
	walletResult := <-walletCh

	// Check for errors
	if userResult.err != nil {
		return nil, nil, userResult.err
	}
	if walletResult.err != nil {
		return nil, nil, walletResult.err
	}

	// Type assertions
	user, ok := userResult.data.(models.User)
	if !ok {
		return nil, nil, fmt.Errorf("invalid user data type")
	}

	wallet, ok := walletResult.data.(models.Wallet)
	if !ok {
		return nil, nil, fmt.Errorf("invalid wallet data type")
	}

	return &user, &wallet, nil
}

// getCacheKeys returns the cache keys for a given phone number
func getCacheKeys(phone string) (string, string) {
	return fmt.Sprintf("user:%s", phone), fmt.Sprintf("wallet:%s", phone)
}
