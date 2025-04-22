package main

import (
	"fmt"
	"github.com/emmadal/feeti-backend-user/middleware"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/emmadal/feeti-backend-user/controllers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Fatalln("Error loading .env file")
	}

	mode := strings.TrimSpace(os.Getenv("MODE"))
	if mode != "release" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = ":4000"
	}

	// Initialize Gin server
	server := gin.Default()

	// middleware
	server.Use(
		cors.New(
			cors.Config{
				AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
				AllowOrigins:     []string{"*"},
				AllowFiles:       false,
				AllowWildcard:    false,
				AllowCredentials: true,
			},
		),
	)
	server.Use(middleware.Helmet())
	server.Use(gzip.Gzip(gzip.BestCompression))
	server.Use(middleware.Timeout(5 * time.Second))
	server.Use(middleware.Recover())

	// Database connection
	models.DBConnect()

	// Set api version group
	v1 := server.Group("/v1/api")

	// initialize server
	s := &http.Server{
		Handler:        server,
		Addr:           port,
		WriteTimeout:   10 * time.Second,
		ReadTimeout:    10 * time.Second,
		IdleTimeout:    20 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// v1 routes
	v1.POST("/register", controllers.Register)
	v1.POST("/new-otp", controllers.NewOTP)
	v1.POST("/check-otp", controllers.CheckOTP)
	v1.POST("/login", controllers.Login)
	v1.PUT("/update-pin", controllers.UpdatePin)
	v1.DELETE("/remove-account", controllers.RemoveAccount)
	v1.DELETE("/sign-out", controllers.SignOut)

	// start server
	_, err := fmt.Fprintf(os.Stdout, "Server started on port %s\n", port)
	if err != nil {
		log.Fatalln("Error writing to stdout")
	}
	log.Fatalln(s.ListenAndServe())
}
