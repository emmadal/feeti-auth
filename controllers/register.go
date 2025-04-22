package controllers

import (
	"net/http"
	"os"
	"time"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	jwt "github.com/emmadal/feeti-module/jwt_module"
	"github.com/gin-gonic/gin"
)

// Register handles user registration
func Register(c *gin.Context) {
	body := models.User{}

	// Validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid request data", err)
		return
	}

	// search if the user exists in DB
	if models.CheckUserByPhone(body.PhoneNumber) {
		helpers.HandleError(c, http.StatusConflict, "User already exist", nil)
		return
	}

	// Hash the user's PIN
	hashedPin, err := helpers.HashPassword(body.Pin)
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Unable to process PIN", err)
		return
	}
	body.Pin = hashedPin

	// Create a user account
	err = body.CreateUser()
	if err != nil {
		helpers.HandleError(c, http.StatusUnprocessableEntity, "Unable to process user", err)
		return
	}

	// create a user wallet
	wallet, err := body.CreateWallet()
	if err != nil {
		helpers.HandleError(c, http.StatusUnprocessableEntity, "Unable to process wallet", err)
		return
	}

	// Generate JWT token
	token, err := jwt.GenerateToken(body.ID, []byte(os.Getenv("JWT_KEY")))
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Unable to generate token", err)
		return
	}

	// Set cookie
	domain := os.Getenv("HOST")
	//secure := os.Getenv("GIN_MODE") == "release"
	c.SetCookie("ftk", token, int(time.Now().Add(30*time.Minute).Unix()), "/", domain, false, true)

	// Send success response
	helpers.HandleSuccessData(
		c, "User registered successfully", models.AuthResponse{
			User: models.UserResponse{
				ID:          body.ID,
				PhoneNumber: body.PhoneNumber,
				FirstName:   body.FirstName,
				LastName:    body.LastName,
				Photo:       "",
				DeviceToken: body.DeviceToken,
			},
			Wallet: models.WalletResponse{
				ID:       wallet.ID,
				Balance:  wallet.Balance,
				Currency: wallet.Currency,
			},
		},
	)
}
