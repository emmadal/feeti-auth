package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/emmadal/feeti-backend-user/controllers"
	"github.com/emmadal/feeti-backend-user/middleware"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/emmadal/feeti-module/cache"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	nrgin "github.com/newrelic/go-agent/v3/integrations/nrgin"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		logrus.WithFields(logrus.Fields{"warning": err}).Error("No .env file found")
	}

	mode := os.Getenv("GIN_MODE")
	if mode != "release" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize Gin server
	server := gin.Default()

	// middleware
	server.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowOrigins:     []string{fmt.Sprintf("http://%s", os.Getenv("HOST"))},
		AllowFiles:       false,
		AllowWildcard:    false,
		AllowCredentials: true,
	}))
	server.Use(middleware.Helmet())
	server.Use(gzip.Gzip(gzip.BestCompression))
	server.Use(middleware.Timeout(5 * time.Second))
	server.Use(middleware.Recover())

	// Initialize New Relic
	app, err := newrelic.NewApplication(
		newrelic.ConfigEnabled(true),
		newrelic.ConfigAppName("backend-user"),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigCodeLevelMetricsEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	if nil != err {
		logrus.WithFields(logrus.Fields{"warning": err}).Warning("No newrelic instance found")
	}
	server.Use(nrgin.Middleware(app))

	// Database connection
	models.DBConnect()

	// Set api version group
	v1 := server.Group("/v1/api")

	// Redis connection
	if err := cache.InitRedis(); err != nil {
		logrus.WithFields(logrus.Fields{"warning": err}).Warning("No redis instance found")
	}

	// initialize server
	s := &http.Server{
		Handler:        server,
		Addr:           os.Getenv("PORT"),
		WriteTimeout:   10 * time.Second,
		ReadTimeout:    10 * time.Second,
		IdleTimeout:    30 * time.Second,
		MaxHeaderBytes: 1 << 5,
	}

	// v1 routes
	v1.POST("/register", controllers.Register)
	v1.POST("/new-otp", controllers.NewOTP)
	v1.POST("/check-otp", controllers.CheckOTP)
	v1.POST("/login", controllers.Login)
	v1.POST("/reset-pin", controllers.ResetPin)
	v1.PUT("/update-pin", controllers.UpdatePin)
	v1.POST("/user", controllers.GetUser)
	v1.GET("/health", controllers.HealthCheck)
	v1.PUT("/update-profile", controllers.UpdateProfile)
	v1.DELETE("/remove-account", controllers.RemoveAccount)
	v1.DELETE("/signout", controllers.SignOut)

	// start server
	fmt.Fprintf(os.Stdout, "Server is running on port %s", os.Getenv("PORT"))
	log.Fatalln(s.ListenAndServe())
}
