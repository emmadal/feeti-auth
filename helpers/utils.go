package helpers

import (
	"crypto/rand"
	"encoding/hex"
)

// SecureRandomText generates a secure random hexadecimal string of length n.
// Returns an error if random data generation fails.
func SecureRandomText(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
