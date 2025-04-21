package controllers

import (
	jwt "github.com/emmadal/feeti-module/jwt_module"
	"net/http"
	"os"
	"time"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/gin-gonic/gin"
)

func UpdatePin(c *gin.Context) {
	var body models.UpdatePin

	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	// Fetch user
	user, err := models.GetUserByPhoneNumber(body.PhoneNumber)
	if err != nil {
		helpers.HandleError(c, http.StatusNotFound, "request not found", err)
		return
	}

	// Verify old PIN
	if !helpers.VerifyPassword(body.OldPin, user.Pin) {
		helpers.HandleError(c, http.StatusUnauthorized, "invalid password or phone number", err)
		return
	}

	// Hash new PIN
	hashedPin, err := helpers.HashPassword(body.ConfirmPin)
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to process new PIN", err)
		return
	}

	// Update PIN
	user.Pin = hashedPin
	if err := user.UpdateUserPin(); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to update PIN", err)
		return
	}

	//Generate JWT token
	token, err := jwt.GenerateToken(user.ID, []byte(os.Getenv("JWT_KEY")))
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "unexpected token error", err)
		return
	}

	// Replace old token with new one
	domain := os.Getenv("HOST")
	secure := os.Getenv("GIN_MODE") == "release"
	c.SetCookie("ftk", token, int(time.Now().Add(30*time.Minute).Unix()), "/", domain, secure, true)

	go helpers.SendPinMessage(body.PhoneNumber)

	// Return success
	helpers.HandleSuccess(c, "Your PIN has been updated. Please, do not share your password")
}
