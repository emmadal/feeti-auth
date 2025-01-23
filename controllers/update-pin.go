package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/emmadal/feeti-module/cache"
	"github.com/gin-gonic/gin"
)

func UpdatePin(c *gin.Context) {
	var body models.UpdatePin

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check request size
	if c.Request.ContentLength > (1024 * 1024) {
		helpers.HandleError(c, http.StatusRequestEntityTooLarge, "Request too large", nil)
		return
	}

	// Validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid data or bad request", err)
		return
	}

	// Create a single error channel and result channels
	errorChan := make(chan error, 1)
	userChan := make(chan *models.User, 1)
	otpChan := make(chan *models.OTP, 1)

	// Cleanup function to close channels
	defer func() {
		close(errorChan)
		close(userChan)
		close(otpChan)
	}()

	// Fetch user with goroutine
	go func() {
		user, err := models.GetUserByPhoneNumber(body.PhoneNumber)
		if err != nil {
			select {
			case errorChan <- err:
			case <-ctx.Done():
			}
			return
		}
		if !user.IsActive || user.Locked || user.Quota >= 3 {
			select {
			case errorChan <- fmt.Errorf("user is inactive or locked"):
			case <-ctx.Done():
			}
			return
		}
		select {
		case userChan <- &models.User{User: *user}:
		case <-ctx.Done():
		}
	}()

	// Fetch OTP with goroutine
	go func() {
		otp, err := models.GetOTPByCodeAndUID(body.PhoneNumber, body.CodeOTP, body.KeyUID)
		if err != nil {
			select {
			case errorChan <- err:
			case <-ctx.Done():
			}
			return
		}
		if !time.Now().Before(otp.ExpiryAt) {
			select {
			case errorChan <- fmt.Errorf("OTP has expired"):
			case <-ctx.Done():
			}
			return
		}
		if otp.IsUsed {
			select {
			case errorChan <- fmt.Errorf("OTP already used"):
			case <-ctx.Done():
			}
			return
		}
		select {
		case otpChan <- otp:
		case <-ctx.Done():
		}
	}()

	// Wait for results using select
	var user *models.User
	var otp *models.OTP

	// First result
	select {
	case err := <-errorChan:
		switch {
		case err.Error() == "user is inactive or locked":
			helpers.HandleError(c, http.StatusUnauthorized, err.Error(), err)
		case err.Error() == "OTP has expired":
			helpers.HandleError(c, http.StatusUnauthorized, err.Error(), err)
		case err.Error() == "OTP already used":
			helpers.HandleError(c, http.StatusUnauthorized, err.Error(), err)
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

	// Update user PIN synchronously to ensure consistency
	user.Pin = hashPin
	if err := user.UpdateUserPin(); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to update PIN", err)
		return
	}

	// Update cache asynchronously since it's not critical
	go func() {
		_ = cache.UpdateDataInCache(user.PhoneNumber, user, 0)
	}()

	helpers.HandleSuccess(c, "PIN updated successfully", nil)
}

// validateOTP checks if the OTP is valid for the user
func validateOTP(otp *models.OTP, body models.UpdatePin) bool {
	return otp != nil &&
		otp.Code == body.CodeOTP &&
		!otp.IsUsed &&
		otp.PhoneNumber == body.PhoneNumber &&
		time.Now().Before(otp.ExpiryAt) &&
		otp.KeyUID == body.KeyUID
}
