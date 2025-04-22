package controllers

import (
	"net/http"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/gin-gonic/gin"
)

// RemoveAccount remove user account
func RemoveAccount(c *gin.Context) {
	body := models.RemoveUserAccount{}

	// Validate request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	// search if user exists in DB
	user, err := models.GetUserByPhoneNumber(body.PhoneNumber)
	if err != nil {
		helpers.HandleError(c, http.StatusNotFound, "request not found", err)
		return
	}

	// verify user password
	if !helpers.VerifyPassword(body.Pin, user.Pin) {
		helpers.HandleError(c, http.StatusUnauthorized, "invalid password or phone number", err)
		return
	}

	// remove a user account
	if err := user.RemoveUserAndWallet(); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "request failed", err)
		return
	}

	// Send success response
	helpers.HandleSuccess(c, "Account removed successfully")
}
