package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func NewOTP(c *gin.Context) {
	var body models.OTP
	var errChan = make(chan error, 1)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Validate the phone number
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid data or bad request", err)
		return
	}

	// Generate OTP
	otpCode := helpers.GenerateOTPCode(5)
	body.KeyUID = uuid.NewString()
	body.Code = otpCode

	// Send OTP in a separate goroutine
	go helpers.SendOTP(body.PhoneNumber, otpCode)

	body.Code = otpCode
	go func() {
		if err := body.InsertOTP(); err != nil {
			errChan <- err
			close(errChan)
			return
		}
	}()

	select {
	case err := <-errChan:
		helpers.HandleError(c, http.StatusInternalServerError, err.Error(), err)
		return
	case <-ctx.Done():
		helpers.HandleError(c, http.StatusInternalServerError, "OTP creation timed out", ctx.Err())
		return
	default:
		helpers.HandleSuccessData(c, "OTP created successfully", body.KeyUID)
		return
	}
}
