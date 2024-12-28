package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
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
		code, user, err := body.CreateUserWithWallet()
		if err != nil {
			helpers.HandleError(c, code, err.Error(), err)
			return
		}
		helpers.HandleSuccessData(c, "User registered successfully", user)
	}
}
