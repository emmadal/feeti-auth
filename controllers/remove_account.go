package controllers

import (
	"encoding/json"
	"fmt"
	jwt "github.com/emmadal/feeti-module/auth"
	status "github.com/emmadal/feeti-module/status"
	"net/http"
	"os"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/gin-gonic/gin"
)

// RemoveAccount remove user account
func RemoveAccount(c *gin.Context) {
	body := models.RemoveUserAccount{}
	var response helpers.ResponsePayload

	// Validate request body
	if err := c.ShouldBindJSON(&body); err != nil {
		status.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	// search if a user exists in DB
	user, err := models.GetUserByPhoneNumber(body.PhoneNumber)
	if err != nil {
		status.HandleError(c, http.StatusNotFound, "Invalid phone number or user PIN", err)
		return
	}

	// verify user password
	if !helpers.VerifyPassword(body.Pin, user.Pin) {
		status.HandleError(c, http.StatusUnauthorized, "invalid password or phone number", err)
		return
	}

	// publish a request to get wallet data
	pMessage := helpers.RequestPayload{
		Subject: "wallet.lock",
		Data:    fmt.Sprintf("%d", user.ID),
	}
	resp, err := pMessage.PublishEvent()
	if err != nil {
		status.HandleError(c, http.StatusInternalServerError, "Unable to process wallet", err)
		return
	}

	// Unmarshal the wallet data
	_ = json.Unmarshal(resp.Data, &response)
	if !response.Success {
		status.HandleError(c, http.StatusUnprocessableEntity, response.Error, nil)
		return
	}

	// remove a user account
	if err := user.DeactivateUserAccount(); err != nil {
		status.HandleError(c, http.StatusInternalServerError, "Failed to remove account", err)
		return
	}

	// Send success response and delete cookie
	jwt.ClearAuthCookie(c, os.Getenv("HOST_URL"))

	// Send success response
	status.HandleSuccess(c, "Account removed successfully")
}
