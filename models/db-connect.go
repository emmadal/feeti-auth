package models

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/emmadal/feeti-module/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// DBConnect connects to the database
func DBConnect() {
	once := sync.Once{}
	once.Do(func() {
		host := os.Getenv("DB_HOST")
		user := os.Getenv("DB_USER")
		password := os.Getenv("DB_PASSWORD")
		dbname := os.Getenv("DB_NAME")
		port := "5432"

		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=GMT", host, user, password, dbname, port)

		result, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			TranslateError:         true,
			SkipDefaultTransaction: true,
			PrepareStmt:            true,
		})

		if err != nil {
			log.Fatalln(err)
		}

		if err := result.AutoMigrate(&models.User{}, &models.Otp{}, &models.Wallet{}); err != nil {
			log.Fatalln(err)
		}
		DB = result
	})

}
