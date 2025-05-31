package controllers

import (
	"github.com/emmadal/feeti-auth/helpers"
	"github.com/emmadal/feeti-auth/models"
	jwt "github.com/emmadal/feeti-module/auth"
	status "github.com/emmadal/feeti-module/status"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"
)

// UpdatePin handles user PIN update
func UpdatePin(c *gin.Context) {
	// Increment counter for HTTP requests total to prometheus
	helpers.HttpRequestsTotal.WithLabelValues(c.Request.URL.Path, c.Request.Method).Inc()

	body := models.UpdatePin{}

	// Validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		status.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	// Fetch user
	user, err := models.GetUserByPhoneNumber(body.PhoneNumber)
	if err != nil {
		status.HandleError(c, http.StatusNotFound, "request not found", err)
		return
	}

	// verify user identity with context data
	id, _ := jwt.GetUserIDFromGin(c)
	if user.ID != id {
		status.HandleError(c, http.StatusForbidden, "Unauthorized user", err)
		return
	}

	// Verify old PIN
	if !helpers.VerifyPassword(body.OldPin, user.Pin) {
		status.HandleError(c, http.StatusUnauthorized, "invalid password or phone number", err)
		return
	}

	// Hash new PIN
	hashedPin, err := helpers.HashPassword(body.ConfirmPin)
	if err != nil {
		status.HandleError(c, http.StatusInternalServerError, "Failed to process new PIN", err)
		return
	}

	// Update PIN
	user.Pin = hashedPin
	if err := user.UpdateUserPin(); err != nil {
		status.HandleError(c, http.StatusInternalServerError, "Failed to update PIN", err)
		return
	}

	//Generate JWT token
	token, err := jwt.GenerateToken(user.ID, []byte(os.Getenv("JWT_KEY")))
	if err != nil {
		status.HandleError(c, http.StatusInternalServerError, "unexpected token error", err)
		return
	}

	// Replace old token with new one
	jwt.SetSecureCookie(c, token, os.Getenv("DOMAIN"))

	// record auth log
	go func() {
		authLog := models.AuthLog{
			UserID:      user.ID,
			PhoneNumber: user.PhoneNumber,
			DeviceToken: user.DeviceToken,
			Activity:    "update_pin",
			Metadata:    `{"source": "update_pin"}`,
		}
		if err := authLog.CreateAuthLog(); err != nil {
			log.Printf("Error creating auth log: %v\n", err)
		}
	}()

	// Return success
	status.HandleSuccess(c, "Your PIN has been updated. Please, do not share your password")
}
