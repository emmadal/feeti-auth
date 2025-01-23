package controllers

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/emmadal/feeti-module/cache"
	jwt "github.com/emmadal/feeti-module/jwt_module"
	"github.com/gin-gonic/gin"
)

// Register handles user registration
func Register(c *gin.Context) {
	var (
		body          models.User
		hashedPinChan = make(chan string, 1)
		errChan       = make(chan error, 1)
	)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "bad request", err)
		return
	}

	// Goroutine to hash the user's PIN
	go func() {
		defer close(hashedPinChan)
		defer close(errChan)
		hashedPin, err := helpers.HashPassword(body.Pin)
		if err != nil {
			errChan <- err
			return
		}
		select {
		case hashedPinChan <- hashedPin:
		case <-ctx.Done():
			// Context timeout occurred, stop processing
			helpers.HandleError(c, http.StatusRequestTimeout, "Request timeout", ctx.Err())
			return
		}
	}()

	// Wait for either PIN hashing result or an error
	select {
	case err := <-errChan:
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to process PIN", err)
		return
	case <-ctx.Done():
		helpers.HandleError(c, http.StatusRequestTimeout, "Request timeout", ctx.Err())
		return
	case hashedPin := <-hashedPinChan:
		body.Pin = hashedPin
	}

	// Attempt to create user and wallet in a transaction
	code, user, wallet, err := body.CreateUserWithWallet()
	if err != nil {
		helpers.HandleError(c, code, err.Error(), err)
		return
	}

	// Generate JWT token
	token, err := jwt.GenerateToken(user.ID, []byte(os.Getenv("JWT_KEY")))
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to generate token", err)
		return
	}

	// Store user data in the cache asynchronously
	go func() {
		if cacheErr := cache.SetDataInCache(user.PhoneNumber, user, 0); cacheErr != nil {
			log.Printf("Failed to cache user data: %v", cacheErr)
		}
	}()

	// Send success response
	helpers.HandleSuccessData(c, "User registered successfully", gin.H{
		"user": gin.H{
			"id":           user.ID,
			"first_name":   user.FirstName,
			"last_name":    user.LastName,
			"email":        user.Email,
			"phone_number": user.PhoneNumber,
			"device_token": user.DeviceToken,
			"photo":        user.Photo,
		},
		"wallet": gin.H{
			"id":       wallet.ID,
			"user_id":  wallet.UserID,
			"balance":  wallet.Balance,
			"currency": wallet.Currency,
		},
		"token": token,
	})
}
