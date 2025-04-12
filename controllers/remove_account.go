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

// RemoveAccount remove user account
func RemoveAccount(c *gin.Context) {
	var body models.RemoveProfile
	ctx := c.Request.Context()

	g, _ := errgroup.WithContext(ctx)

	// Validate request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	// search if user exists in DB
	user, err := models.GetUserByPhoneNumber(ctx, body.PhoneNumber)
	if err != nil {
		helpers.HandleError(c, http.StatusNotFound, "User not found", err)
		return
	}

	// verify user password and OTP concurrently
	g.Go(func() error {
		if !helpers.VerifyPassword(body.Pin, user.Pin) {
			return fmt.Errorf("invalid credentials")
		}
		return nil
	})

	// verify OTP
	g.Go(func() error {
		if err := retrieveAndUpdateOTP(ctx, &body); err != nil {
			return err
		}
		return nil
	})

	// Wait for goroutines to finish
	if err := g.Wait(); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, err.Error(), err)
		return
	}

	// remove user and wallet from DB
	if err := user.RemoveUser(ctx); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Removing user failed", err)
		return
	}

	// remove user and wallet from cache
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		userKey := fmt.Sprintf("user:%s", body.PhoneNumber)
		walletKey := fmt.Sprintf("wallet:%s", body.PhoneNumber)

		pipeline := cache.ExportRedisClient().Pipeline()
		pipeline.Del(cacheCtx, userKey, walletKey)
		if _, err := pipeline.Exec(cacheCtx); err != nil {
			if cacheCtx.Err() == context.DeadlineExceeded {
				logrus.WithFields(logrus.Fields{"userKey": userKey, "walletKey": walletKey}).
					Error("Redis timeout while deleting user data")
			} else {
				logrus.WithFields(logrus.Fields{"error": err, "userKey": userKey, "walletKey": walletKey}).
					Error("Failed to remove user cache")
			}
		}
	}()

	// Send success response
	helpers.HandleSuccessData(c, "Account removed successfully", nil)
}

func retrieveAndUpdateOTP(ctx context.Context, body *models.RemoveProfile) error {
	otp, err := models.GetOTPByCodeAndUID(ctx, body.PhoneNumber, body.CodeOTP, body.KeyUID)
	if err != nil {
		return err
	}
	if otp.Code != body.CodeOTP || body.PhoneNumber != otp.PhoneNumber {
		return fmt.Errorf("Invalid OTP code or phone number")
	}

	if otp.IsUsed || time.Now().After(otp.ExpiryAt) || otp.KeyUID != body.KeyUID {
		return fmt.Errorf("Invalid or expired OTP")
	}
	if err := otp.UpdateOTP(ctx); err != nil {
		return err
	}
	return nil
}
