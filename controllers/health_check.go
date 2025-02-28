package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthCheck handler to check health of the server
func HealthCheck(c *gin.Context) {
	c.SecureJSON(http.StatusOK, gin.H{"status": "ok", "message": "Server is healthy", "timestamp": time.Now().Unix(), "Date": time.Now().String()})
}
