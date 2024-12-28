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
	// create a new CheckOTP struct
	var body models.CheckOTP
	var errChan = make(chan error)
	var otpChan = make(chan models.OTP)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	defer close(errChan)
	defer close(otpChan)

	// bind the request body to the struct
	if err := c.ShouldBindJSON(&body); err != nil {
		// log and handle the error
		helpers.HandleError(c, http.StatusBadRequest, "Bad request or invalid data", err)
		return
	}

	// retrieve the OTP in a separate goroutine
	go func() {
		otp, err := body.GetOTP()
		if err != nil {
			errChan <- err
			return
		}
		otpChan <- *otp
	}()

	select {
	case err := <-errChan:
		// log and handle the error
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to retrieve OTP", err)
		return
	case otp := <-otpChan:
		// verify if the OTP is valid and not expired
		if otp.IsUsed || time.Now().After(otp.ExpiryAt) {
			helpers.HandleError(c, http.StatusForbidden, "Invalid or expired OTP", nil)
			return
		}
		// update the OTP status
		if err := models.CheckOTP.UpdateOTP(models.CheckOTP{
			Code:        body.Code,
			PhoneNumber: body.PhoneNumber,
			KeyUID:      body.KeyUID,
		}); err != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Failed to validate OTP", err)
			return
		}
		// return a success response
		helpers.HandleSuccess(c, "OTP validated successfully", nil)
	case <-ctx.Done():
		helpers.HandleError(c, http.StatusInternalServerError, "OTP retrieval timed out", ctx.Err())
		return
	}

}
