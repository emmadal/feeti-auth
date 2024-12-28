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

func UpdatePin(c *gin.Context) {
	var body models.UserUpdatePin
	var wg sync.WaitGroup
	var userOTPChan = make(chan map[string]interface{}, 1)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid data or bad request", err)
		return
	}

	// Fetch OTP and User in a separate goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		otp, err := models.GetOTPByParams(body.KeyUID, body.PhoneNumber, body.CodeOTP)
		if err != nil {
			helpers.HandleError(c, http.StatusNotFound, err.Error(), err)
			return
		}
		response, err := models.GetUserByPhone(body.PhoneNumber)
		if err != nil {
			helpers.HandleError(c, http.StatusNotFound, err.Error(), err)
			return
		}

		userOTPChan <- map[string]interface{}{
			"otp":  otp,
			"user": response,
		}
	}()

	// Wait for all goroutines to finish
	wg.Wait()

	// Validate OTP and user old pin
	var result = <-userOTPChan

	// convert OTP to valid struct and compare
	otp, _ := result["otp"].(*models.OTP)
	if !OTPValidation(otp, body) {
		helpers.HandleError(c, http.StatusForbidden, "Invalid or expired OTP", nil)
		return
	}

	// convert user to valid struct and compare the password
	user, _ := result["user"].(*models.User)
	if !user.IsActive || !helpers.VerifyPassword(user.Pin, body.OldPin) {
		helpers.HandleError(c, http.StatusForbidden, "Invalid user or pin", nil)
		return
	}

	// Hash new PIN
	hashPin, err := helpers.HashPassword(body.ConfirmPin)
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	// Update the user's pin
	user.Pin = hashPin
	if err := user.UpdateUserPin(); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	// Success response
	helpers.HandleSuccess(c, "User pin updated successfully", nil)

	// Check if the context has timed out
	if ctx.Err() != nil {
		helpers.HandleError(c, http.StatusRequestTimeout, "pin update timed out", nil)
		return
	}
}

func OTPValidation(otp *models.OTP, body models.UserUpdatePin) bool {
	return otp.Code == body.CodeOTP && otp.IsUsed && otp.PhoneNumber == body.PhoneNumber && time.Now().Before(otp.ExpiryAt) && otp.KeyUID == body.KeyUID
}
