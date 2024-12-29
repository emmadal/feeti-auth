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

	// Bind the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	// Try to get user from cache first
	cachedUser, err := cache.GetDataFromCache[models.User](body.PhoneNumber)
	if err == nil {
		// If user found in cache
		if cachedUser.Locked || cachedUser.Quota >= 3 {
			cachedUser.Locked = true
			go cache.UpdateDataInCache(cachedUser.PhoneNumber, cachedUser, 0)
			go cachedUser.LockUser()
			helpers.HandleError(c, http.StatusUnauthorized, "Account locked due to reached login attempts", nil)
			return
		}
		// Verify PIN
		if ok := helpers.VerifyPassword(body.Pin, cachedUser.Pin); !ok {
			handleFailedLogin(c, &cachedUser)
			return
		}
		// send success response
		handleSuccessfulLogin(c, &cachedUser)
		return
	}

	// Get user from database
	user, err := models.GetUserByPhoneNumber(body.PhoneNumber)
	if err != nil {
		helpers.HandleError(c, http.StatusNotFound, "Invalid phone number", err)
		return
	}

	// Cache the user for future requests
	cache.SetDataInCache(user.PhoneNumber, user, 0)

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
	if user.Quota < 3 {
		if err := user.UpdateUserQuota(); err != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Failed to update login attempts", err)
			return
		}
		user.Quota++
		go cache.UpdateDataInCache(user.PhoneNumber, user, 0)
		helpers.HandleError(c, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}

	// Lock account if quota exceeded
	user.Locked = true
	if err := user.LockUser(); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Internal server error", err)
		return
	}

	// Update user in cache after locking
	go cache.UpdateDataInCache(user.PhoneNumber, user, 0)
	go helpers.SmsAccountLocked(user.PhoneNumber)

	helpers.HandleError(c, http.StatusUnauthorized, "Account locked", nil)
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
		go func(u *models.User) {
			localUser := models.User{User: *&u.User}
			if err := localUser.ResetUserQuota(); err != nil {
				// Log error but don't fail the request
				helpers.HandleError(c, http.StatusInternalServerError, "Failed to reset quota", err)
			}
			localUser.Quota = 0
			cache.UpdateDataInCache(localUser.PhoneNumber, localUser, 0)
		}(user)
	}

	// Send success response
	helpers.HandleSuccessData(c, "Login successful", map[string]interface{}{
		"token": token,
		"user": map[string]interface{}{
			"id":           user.ID,
			"phone":        user.PhoneNumber,
			"first_name":   user.FirstName,
			"last_name":    user.LastName,
			"device_token": user.DeviceToken,
			"photo":        user.Photo,
			"email":        user.Email,
		},
	})
}
