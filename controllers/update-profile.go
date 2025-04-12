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
	ctx := c.Request.Context()

	// Validate request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Invalid request format", err)
		return
	}

	// search if user exists in DB
	if exist := models.CheckUserByPhone(ctx, body.PhoneNumber); !exist {
		helpers.HandleError(c, http.StatusNotFound, "User data not found", nil)
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
			if cacheCtx.Err() == context.DeadlineExceeded {
				logrus.WithFields(logrus.Fields{"userKey": userKey}).
					Error("Redis timeout while updating user data")
			} else {
				logrus.WithFields(logrus.Fields{"error": err, "userKey": userKey}).
					Error("Failed to update user cache")
			}
		}
	}()

	// Send success response
	helpers.HandleSuccessData(c, "User profile updated successfully", map[string]any{
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
