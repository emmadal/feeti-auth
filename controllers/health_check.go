package controllers

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthCheck handler to check health of the server
func HealthCheck(c *gin.Context) {
	start := time.Now()
	c.SecureJSON(http.StatusOK, gin.H{"status": "ok", "message": "Server is healthy", "timestamp": time.Now().Unix(), "uptime": time.Since(start).Seconds(), "memoryUsage": runtime.MemStats{}.Alloc, "Environment": "Production"})
}
