package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/emmadal/feeti-backend-user/controllers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/emmadal/feeti-module/cache"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	nrgin "github.com/newrelic/go-agent/v3/integrations/nrgin"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func main() {
	// Load environment variables
	mode := os.Getenv("GIN_MODE")
	if mode != "release" {
		gin.SetMode(gin.DebugMode)
		err := godotenv.Load()
		if err != nil {
			log.Fatalln("Error loading .env file")
		}
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize Gin server
	server := gin.Default()

	// Initialize New Relic
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("backend-user"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDebugLogger(os.Stdout),
		newrelic.ConfigCodeLevelMetricsEnabled(true),
	)
	if nil != err {
		log.Fatalln(err)
	}
	server.Use(nrgin.Middleware(app))

	// Database connection
	models.DBConnect()

	// Set api version group
	v1 := server.Group("/v1/api")

	// Redis connection
	err = cache.InitRedis()
	if err != nil {
		log.Fatalln(err)
	}

	// initialize server
	s := &http.Server{
		Handler:      server,
		Addr:         os.Getenv("PORT"),
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}

	// v1 routes
	v1.POST("/register", controllers.Register)
	v1.POST("/new-otp", controllers.NewOTP)
	v1.POST("/check-otp", controllers.CheckOTP)
	v1.POST("/login", controllers.Login)
	v1.POST("/reset-pin", controllers.ResetPin)
	v1.PUT("/update-pin", controllers.UpdatePin)
	v1.POST("/user", controllers.GetUser)

	// start server
	log.Printf("Server is running on port %s", os.Getenv("PORT"))
	log.Fatalln(s.ListenAndServe())
}
