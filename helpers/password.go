package helpers

import (
	"fmt"
	"strings"

	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"

	"golang.org/x/crypto/argon2"
)

// HashPassword hashes user password
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return "unexpected error generating salt", err
	}
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	encodedHash := fmt.Sprintf("$argon2id$v=%d$m=65536,t=4,p=4$%s$%s", argon2.Version, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash))
	return encodedHash, nil
}

// VerifyPassword verifies a password against a given hash.
func VerifyPassword(password, encodedHash string) bool {
	splitHash := strings.Split(encodedHash, "$")
	if len(splitHash) != 6 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(splitHash[3])
	if err != nil {
		return false
	}
	hash, err := base64.RawStdEncoding.DecodeString(splitHash[4])
	if err != nil {
		return false
	}
	otherHash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return subtle.ConstantTimeCompare(hash, otherHash) == 1

}
