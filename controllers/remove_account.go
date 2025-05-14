package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/gin-gonic/gin"
)

// RemoveAccount remove user account
func RemoveAccount(c *gin.Context) {
	body := models.RemoveUserAccount{}
	var response helpers.RequestResponse

	// Validate request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	// search if a user exists in DB
	user, err := models.GetUserByPhoneNumber(body.PhoneNumber)
	if err != nil {
		helpers.HandleError(c, http.StatusNotFound, "Invalid phone number or user PIN", err)
		return
	}

	// verify user password
	if !helpers.VerifyPassword(body.Pin, user.Pin) {
		helpers.HandleError(c, http.StatusUnauthorized, "invalid password or phone number", err)
		return
	}

	// publish a request to get wallet data
	pMessage := helpers.ProducerMessage{
		Subject: "wallet.lock",
		Data:    fmt.Sprintf("%d", user.ID),
	}
	resp, err := pMessage.WalletEvent()
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Unable to process wallet", err)
		return
	}

	// Unmarshal the wallet data
	_ = json.Unmarshal(resp.Data, &response)
	if !response.Success {
		helpers.HandleError(c, http.StatusUnprocessableEntity, response.Error, nil)
		return
	}

	// remove a user account
	if err := user.DeactivateUserAccount(); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to remove account", err)
		return
	}

	// Send success response and delete cookie
	c.SetCookie("ftk", "", -1, "/", os.Getenv("HOST"), false, true)

	// Send success response
	helpers.HandleSuccess(c, "Account removed successfully")
}
