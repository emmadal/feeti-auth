package controllers

import (
	"net/http"
	"time"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/gin-gonic/gin"
)

func CheckOTP(c *gin.Context) {
	var body models.Otp

	// Bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	if err := body.GetOTP(); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "OTP not found or invalid", err)
		return
	}

	if time.Now().After(body.ExpiryAt) || body.IsUsed {
		helpers.HandleError(c, http.StatusForbidden, "Expired OTP code", nil)
		return
	}

	// Mark OTP as used BEFORE returning success
	if err := body.UpdateOTP(); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to update OTP", err)
		return
	}

	helpers.HandleSuccess(c, "OTP validated successfully")
}
