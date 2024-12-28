package controllers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/gin-gonic/gin"
)

func UpdatePin(c *gin.Context) {
	var body models.UpdatePin

	// Validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid data or bad request", err)
		return
	}

	// Validate new PIN format
	if !validPinFormat(body.NewPin) {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid new PIN format: must be 4-6 digits", nil)
		return
	}

	// Create channels with proper buffer sizes
	errorChan := make(chan error, 2) // Buffer for both goroutines
	userChan := make(chan models.User, 1)
	otpChan := make(chan models.OTP, 1)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer close(errorChan)
	defer close(userChan)
	defer close(otpChan)

	// Fetch user with goroutine
	go func() {
		user, err := models.GetUserByPhoneNumber(body.PhoneNumber)
		if err != nil {
			errorChan <- err
			return
		}
		if !user.IsActive {
			errorChan <- errors.New("user is inactive")
			return
		}
		select {
		case userChan <- models.User{User: *user}:
		case <-ctx.Done():
		}
	}()

	// Fetch OTP with goroutine
	go func() {
		otp, err := models.GetOTPByCodeAndUID(body.PhoneNumber, body.CodeOTP, body.KeyUID)
		if err != nil {
			errorChan <- err
			return
		}
		if !time.Now().Before(otp.ExpiryAt) {
			errorChan <- errors.New("OTP has expired")
			return
		}
		if otp.IsUsed {
			errorChan <- errors.New("OTP already used")
			return
		}
		select {
		case otpChan <- *otp:
		case <-ctx.Done():
		}
	}()

	// Wait for results
	var user models.User
	var otp models.OTP

	// First result
	select {
	case err := <-errorChan:
		switch {
		case err.Error() == "user is inactive":
			helpers.HandleError(c, http.StatusUnauthorized, "Unable to update pin for inactive user", err)
		case err.Error() == "OTP has expired":
			helpers.HandleError(c, http.StatusUnauthorized, "OTP has expired", err)
		case err.Error() == "OTP already used":
			helpers.HandleError(c, http.StatusUnauthorized, "OTP has already been used", err)
		default:
			helpers.HandleError(c, http.StatusInternalServerError, "Failed to process request", err)
		}
		return
	case <-ctx.Done():
		helpers.HandleError(c, http.StatusRequestTimeout, "Request timeout", ctx.Err())
		return
	case user = <-userChan:
		if !helpers.VerifyPassword(user.Pin, body.OldPin) {
			helpers.HandleError(c, http.StatusUnauthorized, "Old PIN is incorrect", nil)
			return
		}
	}

	// Second result
	select {
	case err := <-errorChan:
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to process request", err)
		return
	case <-ctx.Done():
		helpers.HandleError(c, http.StatusRequestTimeout, "Request timeout", ctx.Err())
		return
	case otp = <-otpChan:
		if !validateOTP(otp, body) {
			helpers.HandleError(c, http.StatusUnauthorized, "Invalid OTP", nil)
			return
		}
	}

	// Hash new PIN
	hashPin, err := helpers.HashPassword(body.NewPin)
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to hash new PIN", err)
		return
	}

	// Begin transaction
	tx := models.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update user's PIN within transaction
	if err := tx.Model(&user).Update("pin", hashPin).Error; err != nil {
		tx.Rollback()
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to update PIN", err)
		return
	}

	// Mark OTP as used within transaction
	if err := tx.Model(&otp).Update("is_used", true).Error; err != nil {
		tx.Rollback()
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to update OTP state", err)
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to complete PIN update", err)
		return
	}

	helpers.HandleSuccess(c, "PIN updated successfully", nil)
}

// validateOTP checks if the OTP is valid for the user
func validateOTP(otp models.OTP, body models.UpdatePin) bool {
	return otp.Code == body.CodeOTP &&
		!otp.IsUsed &&
		otp.PhoneNumber == body.PhoneNumber &&
		time.Now().Before(otp.ExpiryAt) &&
		otp.KeyUID == body.KeyUID
}

// isValidPinFormat validates the PIN format
func validPinFormat(pin string) bool {
	if len(pin) < 4 || len(pin) > 6 {
		return false
	}
	for _, c := range pin {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
