// package controllers

// import (
// 	"context"
// 	"fmt"
// 	"net/http"
// 	"sync"
// 	"time"

// 	"github.com/emmadal/feeti-backend-user/helpers"
// 	"github.com/emmadal/feeti-backend-user/models"
// 	"github.com/emmadal/feeti-module/cache"
// 	"github.com/gin-gonic/gin"
// )

// // handleGoroutineErrors collects errors from a channel and returns the first one received.
// func handleGoroutineErrors(ctx context.Context, errorChan <-chan error) (error, int) {
// 	select {
// 	case err := <-errorChan:
// 		return err, http.StatusUnauthorized
// 	case <-ctx.Done():
// 		return fmt.Errorf("Request timed out"), http.StatusRequestTimeout
// 	}
// }

// // fetchUser fetches a user based on the phone number and performs basic validations.
// func fetchUser(ctx context.Context, phoneNumber string, errorChan chan<- error, userChan chan<- *models.User, wg *sync.WaitGroup) {
// 	defer wg.Done()
// 	select {
// 	case <-ctx.Done():
// 		errorChan <- ctx.Err()
// 		return
// 	default:
// 		user, err := models.GetUserByPhoneNumber(phoneNumber)
// 		if err != nil {
// 			errorChan <- err
// 			return
// 		}
// 		if user.Locked || user.Quota >= 3 {
// 			errorChan <- fmt.Errorf("Feeti account is locked")
// 			return
// 		}
// 		userChan <- &models.User{User: *user}
// 	}
// }

// // fetchOTP validates the OTP details asynchronously.
// func fetchOTP(ctx context.Context, body models.UpdatePin, errorChan chan<- error, wg *sync.WaitGroup) {
// 	defer wg.Done()
// 	select {
// 	case <-ctx.Done():
// 		errorChan <- ctx.Err()
// 		return
// 	default:
// 		otp, err := models.GetOTPByCodeAndUID(body.PhoneNumber, body.CodeOTP, body.KeyUID)
// 		if err != nil || otp.IsUsed || time.Now().After(otp.ExpiryAt) {
// 			errorChan <- fmt.Errorf("Invalid or expired OTP")
// 			return
// 		}
// 		if otp.KeyUID != body.KeyUID || otp.Code != body.CodeOTP || otp.PhoneNumber != body.PhoneNumber {
// 			errorChan <- fmt.Errorf("Invalid OTP")
// 			return
// 		}
// 	}
// }

// func UpdatePin(c *gin.Context) {
// 	// recover from panic to avoid server crash
// 	defer func() {
// 		if r := recover(); r != nil {
// 			helpers.HandleError(c, http.StatusInternalServerError, "Internal server error", nil)
// 		}
// 	}()

// 	// body request
// 	var body models.UpdatePin
// 	var wg sync.WaitGroup // WaitGroup to wait for goroutines

// 	// Channels for error and user
// 	errorChan := make(chan error, 1)
// 	userChan := make(chan *models.User, 1)

// 	// Validate request body
// 	if err := c.ShouldBindJSON(&body); err != nil {
// 		helpers.HandleError(c, http.StatusBadRequest, "Bad request", err)
// 		return
// 	}

// 	// Context with timeout
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer func() {
// 		cancel()
// 		close(errorChan)
// 		close(userChan)
// 	}()

// 	// Fetch user and OTP concurrently
// 	wg.Add(2)
// 	go fetchUser(ctx, body.PhoneNumber, errorChan, userChan, &wg)
// 	go fetchOTP(ctx, body, errorChan, &wg)

// 	// Handle error or timeout
// 	err, statusCode := handleGoroutineErrors(ctx, errorChan)
// 	if err != nil {
// 		helpers.HandleError(c, statusCode, err.Error(), err)
// 		return
// 	}

// 	// Get user result
// 	user := <-userChan

// 	// Verify old PIN
// 	if !helpers.VerifyPassword(user.Pin, body.OldPin) {
// 		helpers.HandleError(c, http.StatusUnauthorized, "Old PIN is incorrect", nil)
// 		return
// 	}

// 	// Hash new PIN
// 	hashPin, err := helpers.HashPassword(body.NewPin)
// 	if err != nil {
// 		helpers.HandleError(c, http.StatusInternalServerError, "Failed to process PIN", err)
// 		return
// 	}

// 	// Update user PIN
// 	user.Pin = hashPin
// 	if err := user.UpdateUserPin(); err != nil {
// 		helpers.HandleError(c, http.StatusInternalServerError, "Failed to update PIN", err)
// 		return
// 	}

// 	// Update cache asynchronously
// 	go cache.UpdateDataInCache(user.PhoneNumber, user, 0)

//		// Send response
//		helpers.HandleSuccess(c, "PIN updated successfully", nil)
//	}
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
func fetchUserWithValidation(ctx context.Context, wg *sync.WaitGroup, phoneNumber string, errorChan chan<- error, userChan chan<- *models.User) {
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
		if user.Locked || user.Quota >= 3 {
			errorChan <- fmt.Errorf("Feeti account is locked")
			return
		}
		userChan <- &models.User{User: *user}
	}
}

// fetchOTPWithValidation fetches and validates the OTP.
func fetchOTPWithValidation(ctx context.Context, wg *sync.WaitGroup, body models.UpdatePin, errorChan chan<- error, otpChan chan<- models.OTP) {
	defer wg.Done()

	select {
	case <-ctx.Done():
		errorChan <- ctx.Err()
		return
	default:
		otp, err := models.GetOTPByCodeAndUID(body.PhoneNumber, body.CodeOTP, body.KeyUID)
		if err != nil || otp.IsUsed || time.Now().After(otp.ExpiryAt) {
			errorChan <- fmt.Errorf("Invalid or expired OTP")
			return
		}
		if otp.KeyUID != body.KeyUID || otp.Code != body.CodeOTP || otp.PhoneNumber != body.PhoneNumber {
			errorChan <- fmt.Errorf("Invalid OTP")
			return
		}
		otpChan <- *otp
	}
}

func UpdatePin(c *gin.Context) {
	// recover from panic to avoid server crash
	defer func() {
		if r := recover(); r != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Internal server error", nil)
		}
	}()

	var body models.UpdatePin

	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errorChan := make(chan error, 2)
	userChan := make(chan *models.User, 1)
	otpChan := make(chan models.OTP, 1)

	var wg sync.WaitGroup

	// Fetch user and OTP concurrently
	wg.Add(2)
	go fetchUserWithValidation(ctx, &wg, body.PhoneNumber, errorChan, userChan)
	go fetchOTPWithValidation(ctx, &wg, body, errorChan, otpChan)

	// Wait for both goroutines to complete
	go func() {
		wg.Wait()
		close(errorChan)
		close(userChan)
		close(otpChan)
	}()

	select {
	case err := <-errorChan:
		if err != nil {
			helpers.HandleError(c, http.StatusUnauthorized, err.Error(), err)
			return
		}
	case <-ctx.Done():
		helpers.HandleError(c, http.StatusRequestTimeout, "Request timeout", ctx.Err())
		return
	}

	// Process user
	user := <-userChan
	otp := <-otpChan

	// Verify old PIN
	if !helpers.VerifyPassword(body.OldPin, user.Pin) {
		helpers.HandleError(c, http.StatusUnauthorized, "Old PIN is incorrect", nil)
		return
	}

	// Hash new PIN
	hashPin, err := helpers.HashPassword(body.NewPin)
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to process PIN", err)
		return
	}

	// Update user PIN
	user.Pin = hashPin
	if err := user.UpdateUserPin(); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	// update otp as used
	if err := otp.UpdateOTP(); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	// update cache
	go cache.UpdateDataInCache(user.PhoneNumber, user, 0)

	// send response
	helpers.HandleSuccess(c, "PIN updated successfully", nil)
}
