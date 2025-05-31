package controllers

import (
	"github.com/emmadal/feeti-auth/helpers"
	jwt "github.com/emmadal/feeti-module/auth"
	status "github.com/emmadal/feeti-module/status"
	"github.com/gin-gonic/gin"
)

// SignOut handles user sign out
func SignOut(c *gin.Context) {
	// Increment counter for HTTP requests total to prometheus
	helpers.HttpRequestsTotal.WithLabelValues(c.Request.URL.Path, c.Request.Method).Inc()

	// Delete cookie
	jwt.ClearAuthCookie(c, "")

	// Return success response
	status.HandleSuccess(c, "Successfully signed out")
}
