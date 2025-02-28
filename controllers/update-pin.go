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
	"github.com/sirupsen/logrus"
)

func fetchUser(ctx context.Context, phoneNumber string) (*models.User, error) {
	return models.GetUserByPhoneNumber(ctx, phoneNumber)
}

func fetchOTP(ctx context.Context, body models.UpdatePin) (*models.Otp, error) {
	otp, err := models.GetOTPByCodeAndUID(ctx, body.PhoneNumber, body.CodeOTP, body.KeyUID)
	if err != nil {
		return nil, err
	}
	if otp.Code != body.CodeOTP {
		return nil, fmt.Errorf("invalid code")
	}
	if body.PhoneNumber != otp.PhoneNumber {
		return nil, fmt.Errorf("invalid phone number")
	}
	if otp.IsUsed || time.Now().After(otp.ExpiryAt) || otp.KeyUID != body.KeyUID {
		return nil, fmt.Errorf("invalid or expired OTP")
	}
	return otp, nil
}

func UpdatePin(c *gin.Context) {
	var body models.UpdatePin
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	var user *models.User
	var otp *models.Otp
	var userErr, otpErr error

	// Fetch user and OTP concurrently
	wg.Add(2)
	go func() {
		defer wg.Done()
		user, userErr = fetchUser(ctx, body.PhoneNumber)
	}()
	go func() {
		defer wg.Done()
		otp, otpErr = fetchOTP(ctx, body)
	}()
	wg.Wait()

	if userErr != nil {
		helpers.HandleError(c, http.StatusNotFound, "User not found", userErr)
		return
	}
	if otpErr != nil {
		helpers.HandleError(c, http.StatusBadRequest, otpErr.Error(), otpErr)
		return
	}

	// Verify old PIN
	if !helpers.VerifyPassword(body.OldPin, user.Pin) {
		helpers.HandleError(c, http.StatusUnauthorized, "Old PIN is incorrect", nil)
		return
	}

	// Mark OTP as used
	if err := otp.UpdateOTP(ctx); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to validate OTP", err)
		return
	}

	// Hash new PIN
	hashedPin, err := helpers.HashPassword(body.NewPin)
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to process new PIN", err)
		return
	}

	// Update PIN
	user.Pin = hashedPin
	if err := user.UpdateUserPin(ctx); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to update PIN", err)
		return
	}

	// Update cache
	cacheKey := fmt.Sprintf("user:%s", user.PhoneNumber)
	if err := cache.SetRedisData(ctx, cacheKey, user, 0); err != nil {
		logrus.WithError(err).WithField("cacheKey", cacheKey).Error("Failed to update cache")
	}

	// Return success
	helpers.HandleSuccess(c, "PIN updated successfully", nil)
}
