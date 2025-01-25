package controllers

import (
	"net/http"
	"os"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/emmadal/feeti-module/cache"
	jwt "github.com/emmadal/feeti-module/jwt_module"
	"github.com/gin-gonic/gin"
)

// Login handler to sign in user
func Login(c *gin.Context) {
	var body models.UserLogin

	// recover from panic
	defer func() {
		if r := recover(); r != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Internal server error", nil)
			return
		}
	}()

	// Bind the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	// Try to get user from cache first
	cachedUser, err := cache.GetDataFromCache[models.User](body.PhoneNumber)

	if err == nil {
		// If user found in cache
		if cachedUser.Locked && cachedUser.Quota >= 3 {
			helpers.HandleError(c, http.StatusUnauthorized, "Account locked. Login attempts exceeded", nil)
			return
		} else {
			// Verify PIN
			if ok := helpers.VerifyPassword(body.Pin, cachedUser.Pin); !ok {
				handleFailedLogin(c, &cachedUser)
				return
			}
		}
		// send success response
		handleSuccessfulLogin(c, &cachedUser)
		return
	}
	// Get user from database
	user, err := models.GetUserByPhoneNumber(body.PhoneNumber)
	if err != nil {
		helpers.HandleError(c, http.StatusNotFound, err.Error(), err)
		return
	}

	// Create local user object to avoid circular reference
	localUser := models.User{User: *user}

	// Verify PIN
	if ok := helpers.VerifyPassword(body.Pin, user.Pin); !ok {
		handleFailedLogin(c, &localUser)
		return
	}

	// send success response
	handleSuccessfulLogin(c, &localUser)
}

// handleFailedLogin handles failed login
func handleFailedLogin(c *gin.Context, user *models.User) {
	// Update login attempts in database and cache
	if user.Quota >= 0 && user.Quota < 3 {
		if err := user.UpdateUserQuota(); err != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Failed to update login attempts", err)
		}
		user.Quota++
		go cache.UpdateDataInCache(user.PhoneNumber, user, 0)
		helpers.HandleError(c, http.StatusUnauthorized, "Invalid credentials,", nil)
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
		go cache.UpdateDataInCache(user.PhoneNumber, user, 0)
		go helpers.SmsAccountLocked(user.PhoneNumber)

		helpers.HandleError(c, http.StatusUnauthorized, "Account locked", nil)
		return
	}

	// Check if account is locked and quota is exceeded
	if user.Locked && user.Quota >= 3 {
		helpers.HandleError(c, http.StatusUnauthorized, "Your feeti account is locked. Please contact the helpdesk", nil)
		return
	}

	helpers.HandleError(c, http.StatusUnauthorized, "Invalid credentials", nil)
	return
}

// handleSuccessfulLogin handles successful login
func handleSuccessfulLogin(c *gin.Context, user *models.User) {
	// Generate JWT token
	token, err := jwt.GenerateToken(user.ID, []byte(os.Getenv("JWT_KEY")))
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Authentication failed", err)
		return
	}

	// Reset quota asynchronously if needed
	if user.Quota > 0 {
		go func() {
			if err := user.ResetUserQuota(); err != nil {
				// Log error but don't fail the request
				helpers.HandleError(c, http.StatusInternalServerError, "Failed to reset quota", err)
				return
			}
			user.Quota = 0
			if err := cache.UpdateDataInCache(user.PhoneNumber, user, 0); err != nil {
				helpers.HandleError(c, http.StatusInternalServerError, "Failed to update cache", err)
				return
			}
		}()
	}

	// Get wallet
	wallet, err := user.GetWalletByUserID()
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to get wallet", err)
		return
	}

	// Cache the user for future requests
	go cache.SetDataInCache(user.PhoneNumber, user, 0)

	// Send success response
	helpers.HandleSuccessData(c, "Login successful", map[string]interface{}{
		"token": token,
		"user": gin.H{
			"id":           user.ID,
			"phone":        user.PhoneNumber,
			"first_name":   user.FirstName,
			"last_name":    user.LastName,
			"device_token": user.DeviceToken,
			"photo":        user.Photo,
			"email":        user.Email,
		},
		"wallet": gin.H{
			"id":       wallet.ID,
			"currency": wallet.Currency,
			"balance":  wallet.Balance,
		},
	})
}
