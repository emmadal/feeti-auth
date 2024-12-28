package controllers

import (
	"net/http"
	"os"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	jwt "github.com/emmadal/feeti-module/jwt_module"
	"github.com/gin-gonic/gin"
)

// Login handler to sign in user
func Login(c *gin.Context) {
	var body models.UserLogin

	// Bind the request body to the UserLogin struct
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid data or bad request", err)
		return
	}

	// Check if the user exists
	user, err := models.GetUserByPhone(body.PhoneNumber)
	if err != nil {
		helpers.HandleError(c, http.StatusNotFound, "Invalid phone number", err)
		return
	}

	// Check if the user is locked or has reached the maximum login attempts
	if user.Quota >= 2 || user.Locked {
		go helpers.SmsAccountLocked(user.PhoneNumber)
		helpers.HandleError(c, http.StatusUnauthorized, "Account locked.", nil)
		return
	}

	// Verify the user's pin
	if ok := helpers.VerifyPassword(body.Pin, user.Pin); !ok {
		if user.Quota < 2 {
			// Increment user quota
			if err := user.UpdateUserQuota(); err != nil {
				helpers.HandleError(c, http.StatusInternalServerError, err.Error(), err)
				return
			}
			helpers.HandleError(c, http.StatusUnauthorized, "Invalid phone number or pin", nil)
			return
		} else {
			// Lock user
			if err := user.LockUser(); err != nil {
				helpers.HandleError(c, http.StatusInternalServerError, err.Error(), err)
				return
			}
			go helpers.SmsAccountLocked(user.PhoneNumber)
			helpers.HandleError(c, http.StatusUnauthorized, "Account locked", nil)
			return
		}
	}

	// Generate JWT synchronously
	token, err := jwt.GenerateToken(user.ID, []byte(os.Getenv("JWT_KEY")))
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to generate token", err)
		return
	}

	// Return successful login response
	helpers.HandleSuccessData(c, "Login successful", gin.H{
		"token": token,
		"user": map[string]interface{}{
			"id":           user.ID,
			"phone":        user.PhoneNumber,
			"first_name":   user.FirstName,
			"last_name":    user.LastName,
			"device_token": user.DeviceToken,
			"photo":        user.Photo,
		},
	})
}
