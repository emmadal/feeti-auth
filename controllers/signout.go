package controllers

import (
	"os"

	"github.com/emmadal/feeti-backend-user/helpers"
	jwt "github.com/emmadal/feeti-module/jwt_module"
	"github.com/gin-gonic/gin"
)

func SignOut(c *gin.Context) {
	// Delete cookie
	jwt.ClearAuthCookie(c, os.Getenv("HOST"))

	// Return success response
	helpers.HandleSuccess(c, "Successfully signed out")
}
