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

// fetchUserWithValidation fetches and validates the user.
func fetchUserWithValidation(ctx context.Context, wg *sync.WaitGroup, phoneNumber string, userChan chan<- *models.User, errorChan chan<- error) {
	defer wg.Done()

	select {
	case <-ctx.Done():
		errorChan <- ctx.Err()
		return
	default:
		user, err := models.GetUserByPhoneNumber(phoneNumber)
		if err != nil {
			errorChan <- err
			return
		}
		select {
		case userChan <- user:
		case <-ctx.Done():
			errorChan <- ctx.Err()
		}
	}
}

// fetchOTPWithValidation fetches and validates the OTP.
func fetchOTPWithValidation(ctx context.Context, wg *sync.WaitGroup, body models.UpdatePin, otpChan chan<- models.OTP, errorChan chan<- error) {
	defer wg.Done()

	select {
	case <-ctx.Done():
		errorChan <- ctx.Err()
		return
	default:
		otp, err := models.GetOTPByCodeAndUID(body.PhoneNumber, body.CodeOTP, body.KeyUID)
		if err != nil {
			errorChan <- err
			return
		}

		if otp.IsUsed {
			errorChan <- fmt.Errorf("OTP has already been used")
			return
		}

		if time.Now().After(otp.ExpiryAt) {
			errorChan <- fmt.Errorf("OTP has expired")
			return
		}

		if otp.KeyUID != body.KeyUID || otp.Code != body.CodeOTP || otp.PhoneNumber != body.PhoneNumber {
			errorChan <- fmt.Errorf("invalid OTP")
			return
		}

		select {
		case otpChan <- *otp:
		case <-ctx.Done():
			errorChan <- ctx.Err()
		}
	}
}

func UpdatePin(c *gin.Context) {
	var body models.UpdatePin

	// recover from panic
	defer func() {
		if r := recover(); r != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Internal server error", fmt.Errorf("recovered: %v", r))
			return
		}
	}()

	// validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// search if user exists in DB
	if !models.CheckUserByPhone(body.PhoneNumber) {
		helpers.HandleError(c, http.StatusNotFound, "User not found", nil)
		return
	}

	errChan := make(chan error, 2)
	userChan := make(chan *models.User, 1)
	otpChan := make(chan models.OTP, 1)

	// Create a wait group to fetch user and OTP in parallel
	var wg sync.WaitGroup
	wg.Add(2)
	go fetchUserWithValidation(ctx, &wg, body.PhoneNumber, userChan, errChan)
	go fetchOTPWithValidation(ctx, &wg, body, otpChan, errChan)

	// Create a goroutine to wait for completion and close channels
	go func() {
		wg.Wait()
		close(userChan)
		close(otpChan)
		close(errChan)
	}()

	// Error handling
	var user *models.User
	var otp models.OTP
	var fetchErr error

	for i := 0; i < 2; i++ {
		select {
		case err := <-errChan:
			if fetchErr == nil { // Capture the first error
				fetchErr = err
			}
		case fetchedUser, ok := <-userChan:
			if ok {
				user = fetchedUser
			}
		case fetchedOtp, ok := <-otpChan:
			if ok {
				otp = fetchedOtp
			}
		case <-ctx.Done():
			fetchErr = ctx.Err()
		}
	}

	if fetchErr != nil {
		helpers.HandleError(c, http.StatusUnauthorized, fetchErr.Error(), fetchErr)
		return
	}

	// Verify old PIN
	if !helpers.VerifyPassword(body.OldPin, user.Pin) {
		helpers.HandleError(c, http.StatusUnauthorized, "Old PIN is incorrect", nil)
		return
	}

	// update OTP isUsed to true
	if err := otp.UpdateOTP(); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	// Hash new PIN
	hashPin, err := helpers.HashPassword(body.NewPin)
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to process PIN", err)
		return
	}

	// update new user's PIN
	user.Pin = hashPin
	if err := user.UpdateUserPin(); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	// update user cache
	cacheKey := fmt.Sprintf("user:%s", user.PhoneNumber)
	go cache.SetRedisData(c, cacheKey, user, 0)

	// send response
	helpers.HandleSuccess(c, "PIN updated successfully", nil)
}
