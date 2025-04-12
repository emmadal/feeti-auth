package controllers

import (
	"net/http"
	"time"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func NewOTP(c *gin.Context) {
	var body models.NewOtp
	var otp models.Otp
	ctx := c.Request.Context()

	// Validate request
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid data or bad request", err)
		return
	}

	// Generate OTP
	code := helpers.GenerateOTPCode(5)
	if len(code) != 5 || code == "00000" {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to generate OTP", nil)
		return
	}

	// Store OTP in database
	otp.KeyUID = uuid.NewString()
	otp.Code = code
	otp.PhoneNumber = body.PhoneNumber
	otp.CreatedAt = time.Now() // Ensure expiry logic works

	err := otp.InsertOTP(ctx)
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to create OTP", err)
		return
	}

	// Send OTP asynchronously
	go helpers.SendOTP(body.PhoneNumber, otp.Code)

	// Send response immediately
	helpers.HandleSuccessData(c, "OTP created successfully", otp.KeyUID)
}
