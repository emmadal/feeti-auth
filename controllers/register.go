package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/emmadal/feeti-module/cache"
	jwt "github.com/emmadal/feeti-module/jwt_module"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Register handles user registration
func Register(c *gin.Context) {
	var body models.User
	ctx := c.Request.Context()

	// Validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid request data", err)
		return
	}

	// search if user exists in DB
	if exists := models.CheckUserByPhone(ctx, body.PhoneNumber); exists {
		helpers.HandleError(c, http.StatusConflict, "User already exists", nil)
		return
	}

	// Hash the user's PIN
	hashedPin, err := helpers.HashPassword(body.Pin)
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Unable to process PIN", err)
		return
	}

	// Attempt to create user and wallet in a transaction
	body.Pin = hashedPin
	user, wallet, err := body.CreateUser(ctx)
	if err != nil {
		helpers.HandleError(c, http.StatusUnprocessableEntity, "Unable to process user", err)
		return
	}

	// Generate JWT token
	token, err := jwt.GenerateToken(user.ID, []byte(os.Getenv("JWT_KEY")))
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Unable to generate token", err)
		return
	}

	// Set cookie
	domain := os.Getenv("HOST")
	secure := os.Getenv("GIN_MODE") == "release"
	c.SetCookie("ftk", token, int(time.Now().Add(30*time.Minute).Unix()), "/", domain, secure, true)

	// Update cache asynchronously
	go cachedRegisterData(user, wallet)

	// Send success response
	helpers.HandleSuccessData(c, "User registered successfully", map[string]any{
		"user": gin.H{
			"id":           user.ID,
			"phone_number": user.PhoneNumber,
			"first_name":   user.FirstName,
			"last_name":    user.LastName,
			"photo":        user.Photo,
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

// cachedRegisterData caches user and wallet data in Redis
func cachedRegisterData(user *models.User, wallet *models.Wallet) {
	cacheCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	marshalUser := marshalData(user)
	marshalWallet := marshalData(wallet)

	userKey, walletKey := getCacheKeys(user.PhoneNumber)
	pipeline := cache.ExportRedisClient().Pipeline()
	pipeline.Set(cacheCtx, userKey, marshalUser, 0)
	pipeline.Set(cacheCtx, walletKey, marshalWallet, 0)
	if _, err := pipeline.Exec(cacheCtx); err != nil {
		if cacheCtx.Err() == context.DeadlineExceeded {
			logrus.WithFields(logrus.Fields{"userKey": userKey, "walletKey": walletKey}).
				Error("Redis timeout while setting user data")
		} else {
			logrus.WithFields(logrus.Fields{"error": err, "userKey": userKey, "walletKey": walletKey}).
				Error("Failed to set user cache")
		}
	}
}

// marshalData marshals data to JSON
func marshalData(data any) []byte {
	jsonData, err := json.Marshal(data)
	if err != nil {
		logrus.WithFields(logrus.Fields{"error": err}).Error("Failed to marshal data")
	}
	return jsonData
}
