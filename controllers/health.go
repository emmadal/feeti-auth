package controllers

import (
	"github.com/emmadal/feeti-backend-user/helpers"
	status "github.com/emmadal/feeti-module/status"
	"github.com/gin-gonic/gin"
	"time"
)

// HealthCheck check is a health check endpoint for kubernetes
func HealthCheck(c *gin.Context) {
	// Increment counter for HTTP requests total to prometheus
	helpers.HttpRequestsTotal.WithLabelValues(c.Request.URL.Path, c.Request.Method).Inc()

	// Return success response
	status.HandleSuccessData(
		c, "OK", gin.H{
			"status":  "up",
			"time":    time.Now().Format(time.RFC3339),
			"service": "Auth Service",
		},
	)
}
