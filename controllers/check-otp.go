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
	var body models.CheckOTP
	var errChan = make(chan error, 1)
	var otpChan = make(chan models.OTP, 1)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// bind the request body to the struct
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Bad request or invalid data", err)
		return
	}

	// Validate required fields
	if body.PhoneNumber == "" || body.KeyUID == "" || body.Code == "" {
		helpers.HandleError(c, http.StatusBadRequest, "Missing required fields", nil)
		return
	}

	// retrieve the OTP in a separate goroutine
	go func() {
		otp, err := body.GetOTP()
		if err != nil {
			errChan <- err
			close(errChan)
			return
		}
		otpChan <- *otp
		close(otpChan)
	}()

	select {
	case err := <-errChan:
		helpers.HandleError(c, http.StatusInternalServerError, err.Error(), err)
		return
	case otp := <-otpChan:
		// verify if the OTP is valid and not expired
		if otp.IsUsed {
			helpers.HandleError(c, http.StatusForbidden, "OTP has already been used", nil)
			return
		}
		if time.Now().After(otp.ExpiryAt) {
			helpers.HandleError(c, http.StatusForbidden, "OTP has expired", nil)
			return
		}
		// update the OTP status
		otp.KeyUID = body.KeyUID
		otp.PhoneNumber = body.PhoneNumber
		otp.Code = body.Code
		if err := otp.UpdateOTP(); err != nil {
			helpers.HandleError(c, http.StatusInternalServerError, err.Error(), err)
			return
		}
		helpers.HandleSuccess(c, "OTP validated successfully", nil)
	case <-ctx.Done():
		// handle context timeout
		helpers.HandleError(c, http.StatusInternalServerError, "Request timeout", ctx.Err())
		return
	}

}
