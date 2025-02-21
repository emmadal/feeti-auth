package controllers

import (
	"fmt"
	"net/http"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/emmadal/feeti-module/cache"
	"github.com/gin-gonic/gin"
)

type bodyStruct struct {
	PhoneNumber string `json:"phone_number" binding:"required,e164,min=11,max=14"`
}

// GetUser gets a user by id
func GetUser(c *gin.Context) {
	var body bodyStruct

	// recover from panic
	defer func() {
		if r := recover(); r != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Internal server error", nil)
			return
		}
	}()

	// validate request or type
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	// check if user exist in cache
	user, wallet, err := getCacheData(c, body.PhoneNumber)
	if err == nil {
		// Send success response
		helpers.HandleSuccessData(c, "User found", map[string]interface{}{
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
			},
			"wallet": gin.H{
				"id":       wallet.ID,
				"currency": wallet.Currency,
				"balance":  wallet.Balance,
			},
		})
		return
	}

	// User not found in cache, continue with database call
	user, wallet, err = models.GetUserAndWalletByPhone(body.PhoneNumber)
	if err != nil {
		helpers.HandleError(c, http.StatusNotFound, err.Error(), err)
		return
	}

	// Send success response
	helpers.HandleSuccessData(c, "User found", map[string]interface{}{
		"user": gin.H{
			"id":           user.ID,
			"phone_number": user.PhoneNumber,
			"first_name":   user.FirstName,
			"last_name":    user.LastName,
			"photo":        user.Photo,
			"email":        user.Email,
		},
		"wallet": gin.H{
			"id":       wallet.ID,
			"currency": wallet.Currency,
			"balance":  wallet.Balance,
		},
	})

}

// getCacheData retrieves user and wallet data from cache concurrently
func retrieveUserFromCache(c *gin.Context, phoneNumber string) (*models.User, *models.Wallet, error) {
	cacheKey := fmt.Sprintf("user:%s", phoneNumber)
	walletKey := fmt.Sprintf("wallet:%s", phoneNumber)

	type result struct {
		data interface{}
		err  error
	}

	userCh := make(chan result, 1)
	walletCh := make(chan result, 1)

	// Fetch user data
	go func() {
		user, err := cache.GetRedisData[models.User](c, cacheKey)
		userCh <- result{data: user, err: err}
	}()

	// Fetch wallet data
	go func() {
		wallet, err := cache.GetRedisData[models.Wallet](c, walletKey)
		walletCh <- result{data: wallet, err: err}
	}()

	// Get results
	userResult := <-userCh
	walletResult := <-walletCh

	// Check for errors
	if userResult.err != nil {
		return nil, nil, userResult.err
	}
	if walletResult.err != nil {
		return nil, nil, walletResult.err
	}

	// Type assertions
	user, ok := userResult.data.(models.User)
	if !ok {
		return nil, nil, fmt.Errorf("invalid user data type")
	}

	wallet, ok := walletResult.data.(models.Wallet)
	if !ok {
		return nil, nil, fmt.Errorf("invalid wallet data type")
	}

	return &user, &wallet, nil
}

// getCacheKeys returns the cache keys for a given phone number
func fetchCacheKeys(phone string) (string, string) {
	return fmt.Sprintf("user:%s", phone), fmt.Sprintf("wallet:%s", phone)
}
