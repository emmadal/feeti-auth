package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/emmadal/feeti-module/cache"
	"github.com/gin-gonic/gin"
)

func ResetPin(c *gin.Context) {
	var (
		body     models.ResetPin
		userChan = make(chan models.User, 1)
		otpChan  = make(chan models.OTP, 1)
		errChan  = make(chan error, 1)
	)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	// fetch user and otp in parallel
	go func() {
		moduleUser, err := models.GetUserByPhoneNumber(body.PhoneNumber)
		if err != nil {
			errChan <- err
		}
		user := &models.User{User: *moduleUser}
		userChan <- *user

		// fetch OTP
		otp, err := models.GetOTPByCodeAndUID(body.PhoneNumber, body.CodeOTP, body.KeyUID)
		if err != nil {
			errChan <- err
		}

		otpChan <- *otp
	}()

	// wait for user and otp
	select {
	case err := <-errChan:
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to reset pin", err)
		return
	case <-ctx.Done():
		helpers.HandleError(c, http.StatusRequestTimeout, "Request timed out", nil)
		return
	case otp := <-otpChan:
		if otp.IsUsed || time.Now().After(otp.ExpiryAt) || otp.KeyUID != body.KeyUID || otp.Code != body.CodeOTP || otp.PhoneNumber != body.PhoneNumber {
			helpers.HandleError(c, http.StatusUnauthorized, "OTP already used", nil)
			return
		}
	case user := <-userChan:
		if user.Locked || user.Quota >= 3 || !user.IsActive {
			helpers.HandleError(c, http.StatusUnauthorized, "Account locked", nil)
			return
		}
	}

	// Hash PIN early to fail fast if invalid
	hashPin, err := helpers.HashPassword(body.Pin)
	if err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Failed to process PIN", err)
		return
	}

	// Update the user's pin
	user := <-userChan
	otp := <-otpChan
	user.Pin = hashPin

	// update otp and user's pin parallelly
	go func() {
		otp.IsUsed = true
		if err := otp.UpdateOTP(); err != nil {
			errChan <- err
			return
		}
		otpChan <- otp
		if err := user.UpdateUserPin(); err != nil {
			errChan <- err
			return
		}
		userChan <- user
	}()

	// wait for both updates
	select {
	case err := <-errChan:
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to reset pin", err)
		return
	case <-ctx.Done():
		helpers.HandleError(c, http.StatusRequestTimeout, "Request timed out", nil)
		return
	default:
		user = <-userChan
		otp = <-otpChan

		// update user in cache
		go cache.UpdateDataInCache(user.PhoneNumber, user, 0)
		helpers.HandleSuccess(c, "Reset pin successfully", nil)
	}

}
