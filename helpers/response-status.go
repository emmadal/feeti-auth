package helpers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var logging = logrus.New()

func init() {
	logging.SetFormatter(&logrus.JSONFormatter{})
	logging.SetLevel(logrus.InfoLevel)
}

// HandleError is a helper function to handle an error
func HandleError(c *gin.Context, status int, message string, err error) {
	logging.WithFields(logrus.Fields{"error": err}).Error(message)
	c.SecureJSON(status, gin.H{
		"message": message,
		"success": false,
	})
}

// HandleSuccess is a helper function to handle a success
func HandleSuccess(c *gin.Context, message string, err error) {
	logging.WithFields(logrus.Fields{"data": "empty"}).Info(message)
	c.SecureJSON(http.StatusOK, gin.H{
		"message": message,
		"success": true,
	})
}

// HandleSuccessData is a helper function to handle a success and data
func HandleSuccessData(c *gin.Context, message string, data interface{}) {
	logging.WithFields(logrus.Fields{"data": "empty"}).Info(message)
	c.SecureJSON(http.StatusOK, gin.H{
		"message": message,
		"success": true,
		"data":    data,
	})
}

// HandleDebug is a helper function to handle a debug
func HandleDebug(c *gin.Context, message string, data interface{}) {
	logging.WithFields(logrus.Fields{"data": data}).Debug(message)
	c.SecureJSON(http.StatusOK, gin.H{
		"message": message,
		"success": true,
		"data":    data,
	})
}

// HandleConflict is a helper function to handle a conflict
func HandleConflict(c *gin.Context, message string, data interface{}) {
	logging.WithFields(logrus.Fields{"data": "empty"}).Warn(message)
	c.SecureJSON(http.StatusConflict, gin.H{
		"message": message,
		"success": false,
	})
}
