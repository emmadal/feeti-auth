package controllers

import (
	"context"
	"fmt"
	"net/http"
	"sync"
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
		otp         models.Otp
		errChan     = make(chan error, 2) // Buffered channel to prevent blocking
		successChan = make(chan bool, 1)
	)

	// recover from panic
	defer func() {
		if r := recover(); r != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Internal server error", fmt.Errorf("recovered: %v", r))
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

	// search if user exists in DB
	if exist, err := models.CheckUserByPhone(ctx, body.PhoneNumber); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Error checking user", err)
		return
	} else if !exist {
		helpers.HandleError(c, http.StatusNotFound, "Unable to reset PIN, user not found", nil)
		return
	}

	// fetch user in a separate goroutine
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		response, err := models.GetUserByPhoneNumber(ctx, body.PhoneNumber)
		if err != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Error getting user", err)
			return
		}
		// check if user is locked
		if response.Locked || response.Quota >= 3 {
			helpers.HandleError(c, http.StatusForbidden, "Account locked, contact support", nil)
			return
		}
		user = *response
	}()

	// fetch OTP in a separate goroutine
	go func() {
		defer wg.Done()
		res, err := models.GetOTPByCodeAndUID(ctx, body.PhoneNumber, body.CodeOTP, body.KeyUID)
		if err != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Error getting OTP", err)
			return
		}
		// check if OTP is valid
		if res.IsUsed || time.Now().After(res.ExpiryAt) || res.KeyUID != body.KeyUID || res.Code != body.CodeOTP || res.PhoneNumber != body.PhoneNumber {
			helpers.HandleError(c, http.StatusUnauthorized, "Invalid or expired OTP", nil)
			return
		}
		otp = *res
	}()

	wg.Wait()

	// Hash PIN early to fail fast if invalid
	hashPin, err := helpers.HashPassword(body.Pin)
	if err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Failed to process PIN", err)
		return
	}
	// update otp and user's pin
	otp.IsUsed = true
	user.Pin = hashPin

	// Perform updates
	go func() {
		// update otp
		if err := otp.UpdateOTP(ctx); err != nil {
			errChan <- err
			return
		}

		// update user's pin
		if err := user.UpdateUserPin(ctx); err != nil {
			errChan <- err
			return
		}

		// Signal success after both updates complete
		successChan <- true
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
		close(successChan)
		close(errChan)
		// update user in cache asynchronously
		cacheKey := fmt.Sprintf("user:%s", user.PhoneNumber)
		go func() {
			_ = cache.SetRedisData(c, cacheKey, user, 0)
		}()
		helpers.HandleSuccess(c, "PIN reset successfully", nil)
	}
}
