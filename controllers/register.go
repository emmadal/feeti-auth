package controllers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/emmadal/feeti-module/cache"
	jwt "github.com/emmadal/feeti-module/jwt_module"
	"github.com/gin-gonic/gin"
)

// Register handles user registration
func Register(c *gin.Context) {
	var body models.User

	// recover from panic to avoid server crash
	defer func() {
		if r := recover(); r != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Internal server error", nil)
		}
	}()

	// Validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "bad request", err)
		return
	}

	// search if user exists in DB
	if models.CheckUserByPhone(body.PhoneNumber) {
		helpers.HandleError(c, http.StatusConflict, "User already exists", nil)
		return
	}

	// Hash the user's PIN
	hashedPin, err := helpers.HashPassword(body.Pin)
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Unable to process PIN", err)
		return
	}

	// Attempt to create user and wallet in a transaction
	body.Pin = hashedPin
	user, wallet, err := body.CreateUser()
	if err != nil {
		helpers.HandleError(c, http.StatusUnprocessableEntity, "Unable to process user", err)
		return
	}

	// Generate JWT token
	token, err := jwt.GenerateToken(user.ID, []byte(os.Getenv("JWT_KEY")))
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to generate token", err)
		return
	}

	// Store user data in the cache asynchronously
	cacheKey := fmt.Sprintf("user:%s", user.PhoneNumber)
	go cache.SetRedisData(c, cacheKey, user.User, 0)

	// Send success response
	helpers.HandleSuccessData(c, "User registered successfully", gin.H{
		"user": gin.H{
			"id":           user.ID,
			"first_name":   user.FirstName,
			"last_name":    user.LastName,
			"email":        user.Email,
			"phone_number": user.PhoneNumber,
			"photo":        user.Photo,
		},
		"wallet": gin.H{
			"id":       wallet.ID,
			"currency": wallet.Currency,
			"balance":  wallet.Balance,
		},
		"token": token,
	})
}
