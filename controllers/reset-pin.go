package controllers

import (
	"fmt"
	"net/http"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/emmadal/feeti-module/cache"
	"github.com/gin-gonic/gin"
)

func ResetPin(c *gin.Context) {
	var (
		body        models.ResetPin
		user        models.User
		otp         models.Otp
		errChan     = make(chan error, 2) // Buffered channel to prevent blocking
		successChan = make(chan bool, 1)
	)
	ctx := c.Request.Context()
	g, _ := errgroup.WithContext(ctx)

	// validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	// search if user exists in DB
	response, err := models.GetUserByPhoneNumber(ctx, body.PhoneNumber)
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	// fetch user in a separate goroutine
	g.Go(func() error {
		// check if user is locked
		if response.Locked || response.Quota >= 3 {
			helpers.HandleError(c, http.StatusForbidden, "Account locked, contact support", nil)
			return fmt.Errorf("account locked")
		}
		user = *response
		return nil
	})

	// fetch OTP in a separate goroutine
	g.Go(func() error {
		res, err := models.GetOTPByCodeAndUID(ctx, body.PhoneNumber, body.CodeOTP, body.KeyUID)
		if err != nil {
			return err
		}
		// check if OTP is valid
		if res.IsUsed || time.Now().After(res.ExpiryAt) || res.KeyUID != body.KeyUID || res.Code != body.CodeOTP || res.PhoneNumber != body.PhoneNumber {
			return fmt.Errorf("invalid or expired OTP")
		}
		otp = *res
		return nil
	})

	// wait for both goroutines to complete
	if err := g.Wait(); err != nil {
		helpers.HandleError(c, http.StatusUnauthorized, "Unable to process data ", nil)
		return
	}

	// Hash PIN early to fail fast if invalid
	hashPin, err := helpers.HashPassword(body.Pin)
	if err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Failed to process PIN", err)
		return
	}
	// update otp and user's pin
	otp.IsUsed = true
	user.Pin = hashPin

	// Perform updates
	go func() {
		// update otp
		if err := otp.UpdateOTP(ctx); err != nil {
			errChan <- err
			return
		}

		// update user's pin
		if err := user.UpdateUserPin(ctx); err != nil {
			errChan <- err
			return
		}

		// Signal success after both updates complete
		successChan <- true
		close(successChan)
		close(errChan)
	}()

	// wait for either error or success
	select {
	case err := <-errChan:
		helpers.HandleError(c, http.StatusUnauthorized, err.Error(), err)
		return

	case <-successChan:
		// update user in cache asynchronously
		cacheKey := fmt.Sprintf("user:%s", user.PhoneNumber)
		go func() {
			_ = cache.SetRedisData(c, cacheKey, user, 0)
		}()
		helpers.HandleSuccess(c, "PIN reset successfully", nil)
	}
}
