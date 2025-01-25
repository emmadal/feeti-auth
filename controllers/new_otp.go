package controllers

import (
	"net/http"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func NewOTP(c *gin.Context) {
	var body models.NewOTP
	var otp models.OTP

	// recover from panic
	defer func() {
		if r := recover(); r != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Internal server error", nil)
			return
		}
	}()

	// Validate the phone number
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid data or bad request", err)
		return
	}

	// Generate OTP
	otp.KeyUID = uuid.NewString()
	otp.Code = helpers.GenerateOTPCode(6)
	otp.PhoneNumber = body.PhoneNumber

	err := otp.InsertOTP()
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to create OTP", err)
		return
	}

	// Send OTP in a separate goroutine
	go helpers.SendOTP(body.PhoneNumber, otp.Code)

	// send success response
	helpers.HandleSuccessData(c, "OTP created successfully", otp.KeyUID)
}
