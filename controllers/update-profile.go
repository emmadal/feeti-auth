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
)

// GetUser gets a user by phone number
func UpdateProfile(c *gin.Context) {
	var body models.Profile

	// Recover from panic
	defer func() {
		if r := recover(); r != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Internal server error", nil)
			return
		}
	}()

	// Create a context with timeout (default: 5s)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Validate request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	// search if user exists in DB
	if exist, err := models.CheckUserByPhone(ctx, body.PhoneNumber); err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, err.Error(), err)
		return
	} else if !exist {
		helpers.HandleError(c, http.StatusNotFound, "User not found", nil)
		return
	}

	// Fetch user from database
	updatedUser, err := body.UpdateProfile(ctx)
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	// update cache asynchronously
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		userKey := fmt.Sprintf("user:%s", body.PhoneNumber)

		marshalUser := marshalData(updatedUser)

		pipeline := cache.ExportRedisClient().Pipeline()
		pipeline.Set(cacheCtx, userKey, marshalUser, 0)

		if _, err := pipeline.Exec(cacheCtx); err != nil {
			logrus.WithFields(logrus.Fields{"error": err, "userKey": userKey}).
				Error(err.Error())
		}
	}()

	// Send success response
	helpers.HandleSuccessData(c, "User profile updated successfully", map[string]interface{}{
		"user": gin.H{
			"id":           updatedUser.ID,
			"phone_number": updatedUser.PhoneNumber,
			"first_name":   updatedUser.FirstName,
			"last_name":    updatedUser.LastName,
			"photo":        updatedUser.Photo,
			"face_id":      updatedUser.FaceID,
			"finger_print": updatedUser.FingerPrint,
			"premium":      updatedUser.Premium,
			"device_token": updatedUser.DeviceToken,
		},
	})
}
