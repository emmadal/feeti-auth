package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/emmadal/feeti-module/cache"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type bodyStruct struct {
	PhoneNumber string `json:"phone_number" binding:"required,e164,min=11,max=14"`
}

// GetUser gets a user by phone number
func GetUser(c *gin.Context) {
	var body bodyStruct
	ctx := c.Request.Context()

	// Validate request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	// Attempt to get user from cache
	user, wallet, err := getCacheData(ctx, body.PhoneNumber)
	if err == nil {
		helpers.HandleSuccessData(c, "User found", formatUserResponse(user, wallet))
		return
	}
	logrus.WithField("phone_number", body.PhoneNumber).Warn("User not found in cache, querying database")

	// Fetch user from database
	user, wallet, err = models.GetUserAndWalletByPhone(ctx, body.PhoneNumber)
	if err != nil {
		logrus.WithField("phone_number", body.PhoneNumber).WithError(err).Error("Failed to fetch user from database")
		helpers.HandleError(c, http.StatusNotFound, "User not found", nil)
		return
	}

	// Cache user asynchronously to prevent future DB queries
	go cacheUserData(user, wallet)

	// Send success response
	helpers.HandleSuccessData(c, "User found", formatUserResponse(user, wallet))
}

// formatUserResponse formats user and wallet data for response
func formatUserResponse(user *models.User, wallet *models.Wallet) map[string]any {
	return map[string]any{
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
	}
}

// cacheUserData stores user data in Redis asynchronously
func cacheUserData(user *models.User, wallet *models.Wallet) {
	cacheCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	userKey, walletKey := getCacheKeys(user.PhoneNumber)

	pipeline := cache.ExportRedisClient().Pipeline()
	pipeline.Set(cacheCtx, userKey, user, 0)
	pipeline.Set(cacheCtx, walletKey, wallet, 0)

	if _, err := pipeline.Exec(cacheCtx); err != nil {
		logrus.WithFields(logrus.Fields{"error": err, "userKey": userKey, "walletKey": walletKey}).
			Error("Failed to cache user data")
	}
}
