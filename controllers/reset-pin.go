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

func ResetPin(c *gin.Context) {
	var body models.ResetPin

	// validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid data or bad request", err)
		return
	}

	// Validate PIN format (add your PIN validation rules)
	if !isValidPinFormat(body.Pin) {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid PIN format: must be 4-6 digits", nil)
		return
	}

	// Hash PIN early to fail fast if invalid
	hashPin, err := helpers.HashPassword(body.Pin)
	if err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Failed to process PIN", err)
		return
	}

	// Create channels
	userChan := make(chan models.User, 1)
	otpChan := make(chan models.OTP, 1)
	errChan := make(chan error, 2)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer close(errChan)
	defer close(userChan)
	defer close(otpChan)

	// fetch user and OTP concurrently
	go func() {
		user, err := models.GetUserByPhoneNumber(body.PhoneNumber)
		if err != nil {
			errChan <- err
			return
		}
		if !user.IsActive {
			errChan <- errors.New("user is inactive")
			return
		}
		select {
		case userChan <- models.User{User: *user}:
		case <-ctx.Done():
		}
	}()

	go func() {
		otp, err := models.GetOTPByCodeAndUID(body.PhoneNumber, body.CodeOTP, body.KeyUID)
		if err != nil {
			errChan <- err
			return
		}
		if !time.Now().Before(otp.ExpiryAt) {
			errChan <- errors.New("OTP has expired")
			return
		}
		if otp.IsUsed {
			errChan <- errors.New("OTP already used")
			return
		}
		select {
		case otpChan <- *otp:
		case <-ctx.Done():
		}
	}()

	// Wait for results using select
	var user models.User
	var otp models.OTP

	// First result
	select {
	case err := <-errChan:
		switch {
		case err.Error() == "user is inactive":
			helpers.HandleError(c, http.StatusUnauthorized, "Unable to reset pin for inactive user", err)
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
		// Continue to next select
	}

	// Second result
	select {
	case err := <-errChan:
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to process request", err)
		return
	case <-ctx.Done():
		helpers.HandleError(c, http.StatusRequestTimeout, "Request timeout", ctx.Err())
		return
	case otp = <-otpChan:
		if !validateUserAndOTP(user, otp, body) {
			helpers.HandleError(c, http.StatusUnauthorized, "Invalid OTP", nil)
			return
		}
	}

	// Begin transaction
	tx := models.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update the user's pin within transaction
	user.Pin = hashPin
	if err := tx.Model(&user).Update("pin", hashPin).Error; err != nil {
		tx.Rollback()
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to reset user pin", err)
		return
	}

	// Mark OTP as used within the same transaction
	if err := tx.Model(&otp).Update("is_used", true).Error; err != nil {
		tx.Rollback()
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to update OTP state", err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to complete pin reset", err)
		return
	}

	helpers.HandleSuccess(c, "Reset pin successfully", nil)
}

// validateUserAndOTP checks if the user and OTP are valid
func validateUserAndOTP(user models.User, otp models.OTP, body models.ResetPin) bool {
	return user.PhoneNumber == body.PhoneNumber &&
		otp.Code == body.CodeOTP &&
		!otp.IsUsed &&
		otp.KeyUID == body.KeyUID
}

// isValidPinFormat validates the PIN format
func isValidPinFormat(pin string) bool {
	// Add your PIN validation logic here
	// Example: 4-6 digits
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
