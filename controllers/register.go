package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	jwt "github.com/emmadal/feeti-module/jwt_module"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
)

// Register handles user registration
func Register(c *gin.Context) {
	body := models.User{}
	var response helpers.ResponsePayload

	// Validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid request data", err)
		return
	}

	// search if the user exists in DB
	if body.CheckUserByPhone() {
		helpers.HandleError(c, http.StatusConflict, "User already exist", nil)
		return
	}

	// Hash the user's PIN
	hashedPin, err := helpers.HashPassword(body.Pin)
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Unable to process PIN", err)
		return
	}

	// Create a user account
	body.Pin = hashedPin
	user, err := body.CreateUser()
	if err != nil {
		helpers.HandleError(c, http.StatusUnprocessableEntity, "Unable to process user", err)
		return
	}

	// create a user wallet
	pMessage := helpers.RequestPayload{
		Subject: "wallet.create",
		Data:    fmt.Sprintf("%d", user.ID),
	}

	// Send the initial wallet creation request
	natsMsg, err := pMessage.PublishEvent()
	if err != nil {
		_ = user.RollbackUser()
		helpers.HandleError(c, http.StatusUnprocessableEntity, "Unable to request wallet creation", err)
		return
	}

	_ = json.Unmarshal(natsMsg.Data, &response)

	// Convert response.Data from map[string]interface{} to models.Wallet
	walletData := response.Data.(map[string]any)
	wallet := models.Wallet{
		ID:       int64(walletData["id"].(float64)),
		Balance:  walletData["balance"].(float64),
		Currency: walletData["currency"].(string),
	}

	// Generate JWT token
	token, err := jwt.GenerateToken(user.ID, []byte(os.Getenv("JWT_KEY")))
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Unable to generate token", err)
		return
	}

	// Set cookie
	jwt.SetSecureCookie(c, token, os.Getenv("HOST"), false)

	// Send success response
	helpers.HandleSuccessData(
		c, "User registered successfully", models.AuthResponse{
			User: models.UserResponse{
				ID:          user.ID,
				PhoneNumber: user.PhoneNumber,
				FirstName:   user.FirstName,
				LastName:    user.LastName,
				Photo:       user.Photo,
				DeviceToken: user.DeviceToken,
			},
			Wallet: wallet,
		},
	)
}
