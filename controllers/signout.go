package controllers

import (
	"os"

	jwt "github.com/emmadal/feeti-module/auth"
	status "github.com/emmadal/feeti-module/status"
	"github.com/gin-gonic/gin"
)

func SignOut(c *gin.Context) {
	// Delete cookie
	jwt.ClearAuthCookie(c, os.Getenv("HOST_URL"))

	// Return success response
	status.HandleSuccess(c, "Successfully signed out")
}
