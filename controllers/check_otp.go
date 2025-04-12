package controllers

import (
	"net/http"
	"time"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/gin-gonic/gin"
)

func CheckOTP(c *gin.Context) {
	var body models.CheckOtp
	ctx := c.Request.Context()

	// Bind request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	// Fetch OTP (Pass context for timeout)
	otp, err := body.GetOTP(ctx)
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "OTP not found or invalid", err)
		return
	}

	if otp.IsUsed {
		helpers.HandleError(c, http.StatusForbidden, "OTP has already been used", nil)
		return
	}

	if time.Now().After(otp.ExpiryAt) || otp.Code != body.Code || otp.KeyUID != body.KeyUID || otp.PhoneNumber != body.PhoneNumber {
		helpers.HandleError(c, http.StatusForbidden, "Invalid OTP code", nil)
		return
	}

	// Mark OTP as used BEFORE returning success
	otp.IsUsed = true
	if err := otp.UpdateOTP(ctx); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to update OTP", err)
		return
	}

	helpers.HandleSuccess(c, "OTP validated successfully", nil)
}
