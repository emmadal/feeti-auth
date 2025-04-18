package helpers

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	var cost int
	if os.Getenv("GIN_MODE") == "release" {
		cost = bcrypt.DefaultCost
	} else {
		cost = bcrypt.MinCost
	}
	password = strings.TrimSpace(password)
	if strings.TrimSpace(password) == "" {
		return "", fmt.Errorf("password cannot be empty")
	}
	encodedHash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(encodedHash), nil
}

// VerifyPassword verifies if a password matches the provided hash
func VerifyPassword(password, encodedHash string) bool {
	password = strings.TrimSpace(password)
	encodedHash = strings.TrimSpace(encodedHash)

	if password == "" || encodedHash == "" {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(encodedHash), []byte(password))
	return err == nil
}
