package controllers

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/gin-gonic/gin"
)

func ResetPin(c *gin.Context) {
	var body models.UserResetPin
	var wg sync.WaitGroup
	var userOTPChan = make(chan map[string]interface{}, 1)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid data or bad request", err)
		return
	}

	// Fetch user and OTP concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		user, err := models.GetUserByPhone(body.PhoneNumber)
		if err != nil {
			helpers.HandleError(c, http.StatusNotFound, err.Error(), err)
			return
		}

		otp, err := models.CheckOTP{
			Code:        body.CodeOTP,
			PhoneNumber: body.PhoneNumber,
			KeyUID:      body.KeyUID,
		}.GetOTP()
		if err != nil {
			helpers.HandleError(c, http.StatusNotFound, err.Error(), err)
			return
		}

		userOTPChan <- map[string]interface{}{
			"user": user,
			"otp":  otp,
		}
	}()

	// Wait for all goroutines to finish or context timeout
	wg.Wait()

	var result = <-userOTPChan
	user, _ := result["user"].(*models.User)
	otp, _ := result["otp"].(*models.OTP)

	if valid := validateUserAndOTP(*user, *otp, body); !valid {
		helpers.HandleError(c, http.StatusUnauthorized, "Invalid or expired OTP", nil)
		return
	}

	// Hash new PIN
	hashPin, err := helpers.HashPassword(body.Pin)
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to hash pin", err)
		return
	}

	// Update the user's pin
	user.Pin = hashPin
	if err := user.UpdateUserPin(); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to reset user pin", err)
		return
	}

	helpers.HandleSuccess(c, "Reset pin successfully", nil)

	// Check if the context has timed out
	if ctx.Err() != nil {
		helpers.HandleError(c, http.StatusRequestTimeout, "register timeout", nil)
		return
	}
}

// validateUserAndOTP checks if the user and OTP are valid
func validateUserAndOTP(user models.User, otp models.OTP, body models.UserResetPin) bool {
	return user.PhoneNumber == body.PhoneNumber && user.IsActive && otp.Code == body.CodeOTP && otp.IsUsed && time.Now().Before(otp.ExpiryAt) && otp.KeyUID == body.KeyUID
}
