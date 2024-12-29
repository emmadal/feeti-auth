package controllers

import (
	"context"
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
	var body models.User
	var hashedPinChan = make(chan string, 1)
	var errChan = make(chan error, 1)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer close(hashedPinChan)
	defer close(errChan)

	// Validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid data or bad request", err)
		return
	}

	// Hash user pin in goroutine
	go func() {
		hashedPin, err := helpers.HashPassword(body.Pin)
		if err != nil {
			errChan <- err
			return
		}
		select {
		case hashedPinChan <- hashedPin:
		case <-ctx.Done():
		}
	}()

	// Wait for PIN hashing result
	select {
	case err := <-errChan:
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to process PIN", err)
		return
	case <-ctx.Done():
		helpers.HandleError(c, http.StatusRequestTimeout, "Request timeout", ctx.Err())
		return
	case hashedPin := <-hashedPinChan:
		body.Pin = hashedPin

		// Create user and wallet in a single transaction
		code, user, wallet, err := body.CreateUserWithWallet()
		if err != nil {
			helpers.HandleError(c, code, err.Error(), err)
			return
		}
		// Generate JWT token
		token, err := jwt.GenerateToken(user.ID, []byte(os.Getenv("JWT_KEY")))
		if err != nil {
			helpers.HandleError(c, http.StatusInternalServerError, err.Error(), err)
			return
		}
		// set user in cache
		user.Pin = hashedPin
		go cache.SetDataInCache(user.PhoneNumber, user, 0)
		// Send success response
		helpers.HandleSuccessData(c, "User registered successfully", map[string]interface{}{
			"user": map[string]any{
				"id":           user.ID,
				"first_name":   user.FirstName,
				"last_name":    user.LastName,
				"email":        user.Email,
				"phone_number": user.PhoneNumber,
				"device_token": user.DeviceToken,
				"photo":        user.Photo,
			},
			"wallet": map[string]any{
				"id":       wallet.ID,
				"user_id":  wallet.UserID,
				"balance":  wallet.Balance,
				"currency": wallet.Currency,
			},
			"token": token,
		})
	}
}
