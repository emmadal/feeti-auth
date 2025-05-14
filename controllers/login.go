package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	jwt "github.com/emmadal/feeti-module/jwt_module"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

const MaxLoginAttempts = 3

// Login handler to sign in a user
func Login(c *gin.Context) {
	body := models.UserLogin{}
	var response helpers.RequestResponse
	natsMsg := make(chan *nats.Msg, 1)

	// Validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	// Otherwise, fetch user and wallet from a database
	userStruct := &models.User{PhoneNumber: body.PhoneNumber}
	user, err := userStruct.GetUserByPhone()
	if err != nil {
		helpers.HandleError(c, http.StatusNotFound, "request not found", err)
		return
	}

	// Check if the user is locked and the quota exceeded
	if user.Locked && user.Quota >= MaxLoginAttempts {
		helpers.HandleError(c, http.StatusLocked, "Your account has been locked. Please contact support", nil)
		return
	}

	// Lock the account if the quota exceeds the limit
	if user.Quota == MaxLoginAttempts && !user.Locked {
		// lock user and wallet concurrently
		group := errgroup.Group{}
		group.Go(
			func() error {
				if err := user.LockUser(); err != nil {
					return err
				}
				return nil
			},
		)

		group.Go(
			func() error {
				// send a nats message to lock wallet
				pMessage := &helpers.ProducerMessage{
					Subject: "wallet.lock",
					Data:    fmt.Sprintf("%d", userStruct.ID),
				}
				resp, err := pMessage.WalletEvent()
				if err != nil {
					return err
				}
				natsMsg <- resp
				return nil
			},
		)

		if err := group.Wait(); err != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Unable to lock user and wallet", err)
			return
		}
		result := <-natsMsg
		_ = json.Unmarshal(result.Data, &response)
		if response.Success && response.Data == nil {
			helpers.HandleError(
				c, http.StatusLocked, "Maximum login attempts reached. Your account has been locked", nil,
			)
			return
		}
		helpers.HandleError(c, http.StatusInternalServerError, "Something went wrong while locking user data", nil)
		return
	}

	// Verify the PIN and increment quota
	if !helpers.VerifyPassword(body.Pin, user.Pin) {
		if user.Quota < MaxLoginAttempts {
			if err := user.UpdateUserQuota(); err != nil {
				helpers.HandleError(c, http.StatusInternalServerError, "Failed to update user quota", err)
				return
			}
		}
		// Return error
		helpers.HandleError(c, http.StatusUnauthorized, "phone number or pin incorrect", nil)
		return
	}

	// publish a request to get wallet data
	pMessage := helpers.ProducerMessage{
		Subject: "wallet.balance",
		Data:    fmt.Sprintf("%d", user.ID),
	}
	resp, err := pMessage.WalletEvent()
	if err != nil {
		helpers.HandleError(c, http.StatusUnprocessableEntity, "Unable to process wallet", err)
		return
	}

	// Unmarshal the wallet data
	_ = json.Unmarshal(resp.Data, &response)
	if !response.Success {
		helpers.HandleError(c, http.StatusUnprocessableEntity, response.Error, nil)
		return
	}

	// Convert response.Data from map[string]interface{} to models.Wallet
	walletData := response.Data.(map[string]any)
	wallet := models.Wallet{
		ID:       int64(walletData["id"].(float64)),
		Balance:  walletData["balance"].(float64),
		Currency: walletData["currency"].(string),
	}

	//Generate JWT token
	token, err := jwt.GenerateToken(user.ID, []byte(os.Getenv("JWT_KEY")))
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, "Unable to generate token", err)
		return
	}

	// Reset user quota if needed to update database
	if user.Quota > 0 {
		if err := user.ResetUserQuota(); err != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Unable to reset quota", err)
			return
		}
	}

	// Update device token only if changed
	if user.DeviceToken != body.DeviceToken {
		if err := user.UpdateDeviceToken(); err != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Unable to update device token", err)
			return
		}
	}

	// Set cookie
	domain := os.Getenv("HOST")
	//secure := os.Getenv("GIN_MODE") == "release"
	c.SetCookie("ftk", token, int(time.Now().Add(30*time.Minute).Unix()), "/", domain, false, true)

	// Return success response
	helpers.HandleSuccessData(
		c, "Login successfully", models.AuthResponse{
			User: models.UserResponse{
				ID:          user.ID,
				PhoneNumber: user.PhoneNumber,
				FirstName:   user.FirstName,
				LastName:    user.LastName,
				Photo:       user.Photo,
				DeviceToken: user.DeviceToken,
			},
			Wallet: wallet,
		},
	)
}
