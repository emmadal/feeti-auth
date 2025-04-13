package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthCheck handler to check health of the server
func HealthCheck(c *gin.Context) {
	startTime := time.Now()
	c.SecureJSON(http.StatusOK, map[string]any{
		"status":    "ok",
		"message":   "Server is healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"uptime":    time.Since(startTime).String(),
	})
}
