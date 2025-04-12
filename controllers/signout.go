package controllers

import (
	"net/http"
	"os"

	"github.com/emmadal/feeti-backend-user/helpers"
	jwt "github.com/emmadal/feeti-module/jwt_module"
	"github.com/gin-gonic/gin"
)

func SignOut(c *gin.Context) {
	// Get user ID from token
	cookie, err := c.Request.Cookie("ftk")
	if err != nil || cookie.Value == "" {
		helpers.HandleError(c, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	_, err = jwt.VerifyToken(cookie.Value, []byte(os.Getenv("JWT_KEY")))
	if err != nil {
		helpers.HandleError(c, http.StatusUnauthorized, "Invalid token", err)
		return
	}

	// Delete cookie
	secure := os.Getenv("GIN_MODE") == "release"
	c.SetCookie("ftk", "", -1, "/", os.Getenv("HOST"), secure, true)

	// Return success response
	helpers.HandleSuccess(c, "Successfully signed out", nil)
}
