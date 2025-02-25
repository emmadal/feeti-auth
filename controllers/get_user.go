package controllers

import (
	"net/http"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
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
