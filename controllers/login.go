package controllers

import (
	"encoding/json"
	"fmt"
	jwt "github.com/emmadal/feeti-module/auth"
	status "github.com/emmadal/feeti-module/status"
	"log"

	"golang.org/x/sync/errgroup"
	"net/http"
	"os"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/gin-gonic/gin"
	"github.com/nats-io/nats.go"
)

const MaxLoginAttempts = 3

// Login handler to sign in a user
func Login(c *gin.Context) {
	// Increment counter for HTTP requests total to prometheus
	helpers.HttpRequestsTotal.WithLabelValues(c.Request.URL.Path, c.Request.Method).Inc()

	body := models.UserLogin{}
	var response helpers.ResponsePayload
	natsMsg := make(chan *nats.Msg, 1)

	// Validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		status.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	// Otherwise, fetch user and wallet from a database
	userStruct := &models.User{PhoneNumber: body.PhoneNumber}
	user, err := userStruct.GetUserByPhone()
	if err != nil {
		status.HandleError(c, http.StatusNotFound, "request not found", err)
		return
	}

	// Check if the user is locked and the quota exceeded
	if user.Locked && user.Quota >= MaxLoginAttempts {
		status.HandleError(c, http.StatusLocked, "Your account has been locked. Please contact support", nil)
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
				pMessage := helpers.RequestPayload{
					Subject: "wallet.lock",
					Data:    fmt.Sprintf("%d", userStruct.ID),
				}
				resp, err := pMessage.PublishEvent()
				if err != nil {
					return err
				}
				natsMsg <- resp
				return nil
			},
		)

		if err := group.Wait(); err != nil {
			status.HandleError(c, http.StatusInternalServerError, "Unable to lock user and wallet", err)
			return
		}
		result := <-natsMsg
		_ = json.Unmarshal(result.Data, &response)
		if response.Success && response.Data == nil {
			status.HandleError(
				c, http.StatusLocked, "Maximum login attempts reached. Your account has been locked", nil,
			)
			return
		}
		status.HandleError(c, http.StatusInternalServerError, "Something went wrong while locking user data", nil)
		return
	}

	// Verify the PIN and increment quota
	if !helpers.VerifyPassword(body.Pin, user.Pin) {
		if user.Quota < MaxLoginAttempts {
			if err := user.UpdateUserQuota(); err != nil {
				status.HandleError(c, http.StatusInternalServerError, "Failed to update user quota", err)
				return
			}
		}
		// Return error
		status.HandleError(c, http.StatusUnauthorized, "phone number or pin incorrect", nil)
		return
	}

	// publish a request to get wallet data
	pMessage := helpers.RequestPayload{
		Subject: "wallet.balance",
		Data:    fmt.Sprintf("%d", user.ID),
	}
	resp, err := pMessage.PublishEvent()
	if err != nil {
		status.HandleError(c, http.StatusUnprocessableEntity, "Unable to process wallet", err)
		return
	}

	// Unmarshal the wallet data
	_ = json.Unmarshal(resp.Data, &response)
	if !response.Success {
		status.HandleError(c, http.StatusUnprocessableEntity, response.Error, nil)
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
		status.HandleError(c, http.StatusInternalServerError, "Unable to generate token", err)
		return
	}

	// Reset user quota if needed to update database
	if user.Quota > 0 {
		if err := user.ResetUserQuota(); err != nil {
			status.HandleError(c, http.StatusInternalServerError, "Unable to reset quota", err)
			return
		}
	}

	// Update device token only if changed
	if user.DeviceToken != body.DeviceToken {
		if err := user.UpdateDeviceToken(); err != nil {
			status.HandleError(c, http.StatusInternalServerError, "Unable to update device token", err)
			return
		}
	}

	// Set cookie
	jwt.SetSecureCookie(c, token, os.Getenv("HOST_URL"), true)

	// record auth log
	go func() {
		authLog := models.AuthLog{
			UserID:      user.ID,
			PhoneNumber: user.PhoneNumber,
			DeviceToken: user.DeviceToken,
			Activity:    "login",
			Metadata:    `{"source": "login"}`,
		}
		if err := authLog.CreateAuthLog(); err != nil {
			log.Printf("Error creating auth log: %v\n", err)
		}
	}()

	// Return success response
	status.HandleSuccessData(
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
