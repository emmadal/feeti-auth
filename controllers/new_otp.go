package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func NewOTP(c *gin.Context) {
	var body models.NewOtp
	var otp models.Otp

	// Recover from panic
	defer func() {
		if r := recover(); r != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Internal server error", nil)
		}
	}()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

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
	go func() {
		if err := helpers.SendOTP(c, body.PhoneNumber, otp.Code); err != nil {
			logrus.WithFields(logrus.Fields{"phone": body.PhoneNumber, "error": err}).Error("Failed to send OTP")
		}
	}()

	// Send response immediately
	helpers.HandleSuccessData(c, "OTP created successfully", otp.KeyUID)
}
