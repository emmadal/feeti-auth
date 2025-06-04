package controllers

import (
	"encoding/json"
	"github.com/emmadal/feeti-auth/helpers"
	"github.com/emmadal/feeti-auth/models"
	jwt "github.com/emmadal/feeti-module/auth"
	status "github.com/emmadal/feeti-module/status"
	"github.com/emmadal/feeti-module/subject"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"log"
	"net/http"
	"os"
)

// Register handles user registration
func Register(c *gin.Context) {
	// Increment counter for HTTP requests total to prometheus
	helpers.HttpRequestsTotal.WithLabelValues(c.Request.URL.Path, c.Request.Method).Inc()

	var body models.User
	var response helpers.ResponsePayload

	// Validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		status.HandleError(c, http.StatusBadRequest, "Invalid request data", err)
		return
	}

	// search if the user exists in DB
	if body.CheckUserByPhone() {
		status.HandleError(c, http.StatusConflict, "User already exist", nil)
		return
	}

	// Hash the user's PIN
	hashedPin, err := helpers.HashPassword(body.Pin)
	if err != nil {
		status.HandleError(c, http.StatusInternalServerError, "Unable to process PIN", err)
		return
	}

	// Create a user account
	body.Pin = hashedPin
	user, err := body.CreateUser()
	if err != nil {
		status.HandleError(c, http.StatusUnprocessableEntity, "Unable to process user", err)
		return
	}

	// create a user wallet
	pMessage := helpers.RequestPayload{
		Subject: subject.SubjectWalletCreate,
		Data:    user.ID.String(),
	}

	// Send the initial wallet creation request
	natsMsg, err := pMessage.PublishEvent()
	if err != nil {
		_ = user.RollbackUser()
		status.HandleError(c, http.StatusUnprocessableEntity, "Unable to request wallet creation", err)
		return
	}

	_ = json.Unmarshal(natsMsg.Data, &response)

	// Convert response.Data from map[string]interface{} to models.Wallet
	walletData := response.Data.(map[string]any)
	wallet := models.Wallet{
		ID:       uuid.MustParse(walletData["id"].(string)),
		Balance:  walletData["balance"].(float64),
		Currency: walletData["currency"].(string),
	}

	// Generate JWT token
	token, err := jwt.GenerateToken(user.ID, []byte(os.Getenv("JWT_KEY")))
	if err != nil {
		status.HandleError(c, http.StatusInternalServerError, "Unable to generate token", err)
		return
	}

	// Set cookie
	jwt.SetSecureCookie(c, token, os.Getenv("DOMAIN"))

	// record auth log
	go func() {
		authLog := models.AuthLog{
			UserID:      user.ID,
			PhoneNumber: user.PhoneNumber,
			DeviceToken: user.DeviceToken,
			Activity:    "register",
			Metadata:    `{"source": "register"}`,
		}
		if err := authLog.CreateAuthLog(); err != nil {
			log.Printf("Error creating auth log: %v\n", err)
		}
	}()

	// Send success response
	status.HandleSuccessData(
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
