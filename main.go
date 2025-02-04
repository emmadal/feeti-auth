package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/emmadal/feeti-backend-user/controllers"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/emmadal/feeti-module/cache"
	jwt "github.com/emmadal/feeti-module/jwt_module"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	mode := os.Getenv("GIN_MODE")
	if mode == "debug" || mode == "" {
		gin.SetMode(gin.DebugMode)
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize Gin server
	server := gin.Default()

	// Database connection
	models.DBConnect()

	// Set api version group
	v1 := server.Group("/v1/api")

	// Redis connection
	err := cache.InitRedis()
	if err != nil {
		log.Fatal(err)
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
	v1.PUT("/update-pin", jwt.AuthAuthorization([]byte(os.Getenv("JWT_KEY"))), controllers.UpdatePin)

	// start server
	log.Printf("Server is running on port %s", os.Getenv("PORT"))
	log.Fatalln(s.ListenAndServe())
}
