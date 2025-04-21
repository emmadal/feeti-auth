package controllers

import (
	"golang.org/x/sync/errgroup"
	"net/http"
	"os"
	"time"

	"github.com/emmadal/feeti-backend-user/helpers"
	"github.com/emmadal/feeti-backend-user/models"
	jwt "github.com/emmadal/feeti-module/jwt_module"
	"github.com/gin-gonic/gin"
)

const MaxLoginAttempts = 3

// Login handler to sign in user
func Login(c *gin.Context) {
	var body models.UserLogin

	// Validate the request body
	if err := c.ShouldBindJSON(&body); err != nil {
		helpers.HandleError(c, http.StatusBadRequest, "Bad request", err)
		return
	}

	// Otherwise, fetch user and wallet from a database
	var user models.User
	var wallet models.Wallet
	if err := models.GetUserAndWalletByPhone(body.PhoneNumber, &user, &wallet); err != nil {
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
				if err := user.LockWallet(); err != nil {
					return err
				}
				return nil
			},
		)

		if err := group.Wait(); err != nil {
			helpers.HandleError(c, http.StatusInternalServerError, "Something went wrong", err)
		}

		helpers.HandleError(
			c, http.StatusLocked, "Maximum login attempts reached. Your account has been locked", nil,
		)
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

	//Generate JWT token
	token, err := jwt.GenerateToken(user.ID, []byte(os.Getenv("JWT_KEY")))
	if err != nil {
		helpers.HandleError(c, http.StatusInternalServerError, err.Error(), err)
		return
	}

	// Reset user quota if needed to update database
	if user.Quota > 0 {
		if err := user.ResetUserQuota(); err != nil {
			helpers.HandleError(c, http.StatusInternalServerError, err.Error(), err)
			return
		}
	}

	// Update device token only if changed
	if user.DeviceToken != body.DeviceToken {
		if err := user.UpdateDeviceToken(); err != nil {
			helpers.HandleError(c, http.StatusInternalServerError, err.Error(), err)
			return
		}
	}

	// Set cookie
	domain := os.Getenv("HOST")
	secure := os.Getenv("GIN_MODE") == "release"
	c.SetCookie("ftk", token, int(time.Now().Add(30*time.Minute).Unix()), "/", domain, secure, true)

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
			Wallet: models.WalletResponse{
				ID:       wallet.ID,
				Balance:  wallet.Balance,
				Currency: wallet.Currency,
			},
		},
	)
}
