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

func ResetPin(c *gin.Context) {
	var (
		body        models.ResetPin
		user        models.User
		otp         models.OTP
		errChan     = make(chan error, 2) // Buffered channel to prevent blocking
		successChan = make(chan bool, 1)
	)

	// recover from panic
	defer func() {
		if r := recover(); r != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Internal server error", nil)
			return
		}
	}()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	// fetch user in a separate goroutine
	go func() {
		response, err := models.GetUserByPhoneNumber(body.PhoneNumber)
		if err != nil {
			errChan <- err
			return
		}
		// check if user is locked
		if response.Locked || response.Quota >= 3 {
			errChan <- fmt.Errorf("Feeti account locked, contact support")
			return
		}
		user = models.User{User: *response}
	}()

	// fetch OTP in a separate goroutine
	go func() {
		res, err := models.GetOTPByCodeAndUID(body.PhoneNumber, body.CodeOTP, body.KeyUID)
		if err != nil {
			errChan <- err
			return
		}
		// check if OTP is valid
		if res.IsUsed || time.Now().After(res.ExpiryAt) || res.KeyUID != body.KeyUID || res.Code != body.CodeOTP || res.PhoneNumber != body.PhoneNumber {
			errChan <- fmt.Errorf("invalid or expired OTP")
			return
		}
		otp = *res
	}()

	// Handle errors from goroutines
	select {
	case err := <-errChan:
		helpers.HandleError(c, http.StatusUnauthorized, err.Error(), err)
		return
	case <-ctx.Done():
		helpers.HandleError(c, http.StatusRequestTimeout, "Request timed out", nil)
		return
	default:
		// Continue if no errors
		// Hash PIN early to fail fast if invalid
		hashPin, err := helpers.HashPassword(body.Pin)
		if err != nil {
			helpers.HandleError(c, http.StatusBadRequest, "Failed to process PIN", err)
			return
		}
		// update otp and user's pin
		otp.IsUsed = true
		user.Pin = hashPin
	}

	// Perform updates
	go func() {
		// update otp
		if err := otp.UpdateOTP(); err != nil {
			errChan <- err
			return
		}

		// update user's pin
		if err := user.UpdateUserPin(); err != nil {
			errChan <- err
			return
		}

		// Signal success after both updates complete
		successChan <- true
		close(successChan)
		close(errChan)
	}()

	// wait for either error or success
	select {
	case err := <-errChan:
		helpers.HandleError(c, http.StatusUnauthorized, err.Error(), err)
		return
	case <-ctx.Done():
		helpers.HandleError(c, http.StatusRequestTimeout, "Request timed out", nil)
		return
	case <-successChan:
		// update user in cache asynchronously
		go cache.UpdateDataInCache(user.PhoneNumber, user, 0)
		helpers.HandleSuccess(c, "PIN reset successfully", nil)
	}
}
