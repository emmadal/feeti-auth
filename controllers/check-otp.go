package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/gin-gonic/gin"
)

func CheckOTP(c *gin.Context) {
	var body models.CheckOtp

	// Recover from panic
	defer func() {
		if r := recover(); r != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Internal server error", nil)
		}
	}()

	// Create a context with timeout (5s)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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

	// Validate OTP
	switch {
	case otp.IsUsed:
		helpers.HandleError(c, http.StatusForbidden, "OTP has already been used", nil)
		return
	case time.Now().After(otp.ExpiryAt):
		helpers.HandleError(c, http.StatusForbidden, "OTP has expired", nil)
		return
	case otp.Code != body.Code:
		helpers.HandleError(c, http.StatusForbidden, "Invalid OTP code", nil)
		return
	case otp.KeyUID != body.KeyUID:
		helpers.HandleError(c, http.StatusForbidden, "Invalid OTP session", nil)
		return
	case otp.PhoneNumber != body.PhoneNumber:
		helpers.HandleError(c, http.StatusForbidden, "Phone number mismatch", nil)
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
