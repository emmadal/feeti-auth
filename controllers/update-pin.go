package controllers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/emmadal/feeti-module/cache"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func UpdatePin(c *gin.Context) {
	var body models.UpdatePin
	var user *models.User
	var otp *models.Otp

	ctx := c.Request.Context()
	g, _ := errgroup.WithContext(ctx)

	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	// Fetch user and OTP concurrently
	g.Go(func() error {
		localUser, err := models.GetUserByPhoneNumber(ctx, body.PhoneNumber)
		if err == nil {
			user = localUser
		}
		return err
	})

	g.Go(func() error {
		localOtp, err := models.GetOTPByCodeAndUID(ctx, body.PhoneNumber, body.CodeOTP, body.KeyUID)
		if err != nil {
			return err
		}
		if otp.Code != body.CodeOTP || body.PhoneNumber != otp.PhoneNumber {
			return fmt.Errorf("Invalid code or Phone Number")
		}
		if otp.IsUsed || time.Now().After(otp.ExpiryAt) || otp.KeyUID != body.KeyUID {
			return fmt.Errorf("Invalid or expired OTP")
		}
		otp = localOtp
		return nil
	})

	if err := g.Wait(); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to fetch data", err)
		return
	}

	// Verify old PIN
	if !helpers.VerifyPassword(body.OldPin, user.Pin) {
		helpers.HandleError(c, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}

	// Mark OTP as used
	if err := otp.UpdateOTP(ctx); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to validate OTP", err)
		return
	}

	// Hash new PIN
	hashedPin, err := helpers.HashPassword(body.NewPin)
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to process new PIN", err)
		return
	}

	// Update PIN
	user.Pin = hashedPin
	if err := user.UpdateUserPin(ctx); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to update PIN", err)
		return
	}

	// Update cache
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		cacheKey := fmt.Sprintf("user:%s", user.PhoneNumber)
		if err := cache.SetRedisData(ctx, cacheKey, user, 0); err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				logrus.WithField("cacheKey", cacheKey).Error("Redis timeout while updating cache")
			} else {
				logrus.WithError(err).WithField("cacheKey", cacheKey).Error("Failed to update cache")
			}
		}
	}()

	// Return success
	helpers.HandleSuccess(c, "PIN updated successfully", nil)
}
