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
	body := models.NewOtp{}
	otp := models.Otp{}

	// Validate request
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid request data", err)
		return
	}

	// Generate OTP
	code, err := helpers.GenerateOTPCode(5)
	if err != nil || code == "00000" {
		helpers.HandleError(c, http.StatusInternalServerError, "Error generating OTP", err)
		return
	}

	// Store OTP in the database
	otp.KeyUID = uuid.NewString()
	otp.Code = code
	otp.PhoneNumber = body.PhoneNumber
	otp.ExpiryAt = time.Now().Add(2 * time.Minute)
	if err := otp.InsertOTP(); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to create OTP", err)
		return
	}

	// Send OTP asynchronously
	go helpers.SendOTP(body.PhoneNumber, otp.Code)

	helpers.HandleSuccessData(c, "Code OTP created successfully", otp.KeyUID)
}
