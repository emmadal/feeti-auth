package controllers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/emmadal/feeti-module/cache"
	jwt "github.com/emmadal/feeti-module/jwt_module"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const MaxLoginAttempts = 3

// Login handler to sign in user
func Login(c *gin.Context) {
	var body models.UserLogin

	// Recover from panic
	defer func() {
		if r := recover(); r != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Internal server error", nil)
			return
		}
	}()

	// Create a context with timeout (5s)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	// Retrieve user and wallet from cache first
	cachedUser, cachedWallet, err := getCacheData(ctx, body.PhoneNumber)
	if err == nil {
		if cachedUser.DeviceToken != body.DeviceToken {
			cachedUser.DeviceToken = body.DeviceToken
		}
		// Check if the user is locked and quota exceeded
		if cachedUser.Locked && cachedUser.Quota >= MaxLoginAttempts {
			helpers.HandleError(c, http.StatusUnauthorized, "Account locked. Login attempts exceeded. Please contact support", nil)
			return
		}
		// Verify the PIN
		if !helpers.VerifyPassword(body.Pin, cachedUser.Pin) {
			handleFailedLogin(c, ctx, cachedUser, cachedWallet)
			return
		}
		handleSuccessfulLogin(c, cachedUser, cachedWallet, body.DeviceToken)
		return
	}

	// Otherwise, fetch user and wallet from database
	user, wallet, err := models.GetUserAndWalletByPhone(ctx, body.PhoneNumber)
	if err != nil {
		helpers.HandleError(c, http.StatusNotFound, err.Error(), err)
		return
	}

	// Check if the user is locked and quota exceeded
	if user.Locked && user.Quota >= MaxLoginAttempts {
		helpers.HandleError(c, http.StatusUnauthorized, "Your account has been locked due to many failed login attempts", nil)
		return
	}

	// Verify the PIN
	if !helpers.VerifyPassword(body.Pin, user.Pin) {
		handleFailedLogin(c, ctx, user, wallet)
		return
	}

	// Send response
	handleSuccessfulLogin(c, user, wallet, body.DeviceToken)
}

// Optimized getCacheData using WaitGroup
func getCacheData(ctx context.Context, phoneNumber string) (*models.User, *models.Wallet, error) {
	userKey, walletKey := getCacheKeys(phoneNumber)

	var userData models.User
	var walletData models.Wallet
	var errUser, errWallet error

	var wg sync.WaitGroup
	wg.Add(2)

	// Fetch user data
	go func() {
		defer wg.Done()
		u, err := cache.GetRedisData[models.User](ctx, userKey)
		if err == nil {
			userData = u
		} else {
			errUser = err
		}
	}()

	// Fetch wallet data
	go func() {
		defer wg.Done()
		w, err := cache.GetRedisData[models.Wallet](ctx, walletKey)
		if err == nil {
			walletData = w
		} else {
			errWallet = err
		}
	}()

	wg.Wait()

	if errUser != nil && errWallet != nil {
		return nil, nil, fmt.Errorf("failed to fetch cache data: user: %v, wallet: %v", errUser, errWallet)
	}

	return &userData, &walletData, nil
}

// Optimized handleSuccessfulLogin with Redis Pipelining
func handleSuccessfulLogin(c *gin.Context, user *models.User, wallet *models.Wallet, deviceToken string) {
	// Generate JWT token
	token, err := jwt.GenerateToken(user.ID, []byte(os.Getenv("JWT_KEY")))
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Failed to verify token", err)
		return
	}

	// Reset user quota if needed to update database
	if user.Quota > 0 {
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			if err := user.ResetUserQuota(bgCtx); err != nil {
				logrus.WithFields(logrus.Fields{"error": err, "user_id": user.ID}).Error("Failed to reset user quota")
			}
		}()
	}

	// Update device token only if changed
	if user.DeviceToken != deviceToken {
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			if err := user.UpdateDeviceToken(bgCtx); err != nil {
				logrus.WithFields(logrus.Fields{"error": err}).Error("Failed to update device token")
			}
		}()
		user.DeviceToken = deviceToken
	}

	// Update cache asynchronously
	user.Quota = 0
	go cachedLoginData(user, wallet)

	// Return success response
	helpers.HandleSuccessData(c, "Login successful", map[string]interface{}{
		"token": token,
		"user": gin.H{
			"id":           user.ID,
			"phone_number": user.PhoneNumber,
			"first_name":   user.FirstName,
			"last_name":    user.LastName,
			"photo":        user.Photo,
			"email":        user.Email,
			"face_id":      user.FaceID,
			"finger_print": user.FingerPrint,
			"premium":      user.Premium,
			"device_token": user.DeviceToken,
		},
		"wallet": gin.H{
			"id":       wallet.ID,
			"currency": wallet.Currency,
			"balance":  wallet.Balance,
		},
	})
}

func handleFailedLogin(c *gin.Context, ctx context.Context, user *models.User, wallet *models.Wallet) {
	// Increment quota
	if user.Quota < MaxLoginAttempts {
		user.Quota++
		if err := user.UpdateUserQuota(ctx); err != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Failed to update user quota", err)
			return
		}
	}

	// Lock the account if quota exceeds the limit
	if user.Quota == MaxLoginAttempts && !user.Locked {
		user.Locked = true
		go helpers.SmsAccountLocked(user.PhoneNumber) // Send SMS to the user
		if err := user.LockUser(ctx); err != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Failed to lock user account", err)
			return
		}
	}

	// Set user data in cache
	go cachedLoginData(user, wallet)

	// Return error
	helpers.HandleError(c, http.StatusUnauthorized, "Invalid phone number or PIN", nil)
}

func getCacheKeys(phone string) (string, string) {
	return fmt.Sprintf("user:%s", phone), fmt.Sprintf("wallet:%s", phone)
}

// cachedLoginData caches user and wallet data in Redis
func cachedLoginData(user *models.User, wallet *models.Wallet) {
	cacheCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	marshalUser := marshalData(user)
	marshalWallet := marshalData(wallet)

	userKey, walletKey := getCacheKeys(user.PhoneNumber)
	pipeline := cache.ExportRedisClient().Pipeline()
	pipeline.Set(cacheCtx, userKey, marshalUser, 0)
	pipeline.Set(cacheCtx, walletKey, marshalWallet, 0)
	if _, err := pipeline.Exec(cacheCtx); err != nil {
		logrus.WithFields(logrus.Fields{"error": err, "userKey": userKey, "walletKey": walletKey}).Error("Failed to set data in pipeline")
	}
}
