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
	var passwordChan = make(chan string, 1)
	var errChan = make(chan error, 1)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parse and validate request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid data or bad request", err)
		return
	}

	// Hash user pin
	go func() {
		hashedPin, err := helpers.HashPassword(body.Pin)
		if err != nil {
			errChan <- err
			close(errChan)
			return
		}
		passwordChan <- hashedPin
		close(passwordChan)
	}()

	select {
	case err := <-errChan:
		// Handle error if hashing fails
		helpers.HandleError(c, http.StatusInternalServerError, err.Error(), nil)
		return
	case password := <-passwordChan:
		body.Pin = password
		// Create user
		code, user, err := body.CreateUser()
		if err != nil {
			helpers.HandleError(c, code, err.Error(), err)
			return
		}
		// Create user wallet
		var wallet models.Wallet
		wallet.UserID = user.ID
		err = wallet.CreateUserWallet()
		if err != nil {
			helpers.HandleError(c, http.StatusInternalServerError, err.Error(), err)
			return
		}

		// Return success response
		helpers.HandleSuccess(c, "User registered successfully", nil)
		return
	case <-ctx.Done():
		// Handle timeout if the context expires
		helpers.HandleError(c, http.StatusRequestTimeout, "Register timeout", nil)
		return
	}
}
