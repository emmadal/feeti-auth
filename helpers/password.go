package helpers

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2Params defines the parameters used by the Argon2id algorithm
type Argon2Params struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32
	KeyLength   uint32
}

// DefaultParams returns recommended parameters for the Argon2id algorithm
func DefaultParams() *Argon2Params {
	return &Argon2Params{
		Memory:      64 * 1024, // 64MB
		Iterations:  3,         // Number of iterations
		Parallelism: 4,         // Degree of parallelism
		SaltLength:  16,        // 16 bytes salt
		KeyLength:   32,        // 32 bytes key length
	}
}

// HashPassword hashes a password using Argon2id with the specified parameters
func HashPassword(password string) (string, error) {
	if len(password) == 0 {
		return "", fmt.Errorf("password cannot be empty")
	}

	params := DefaultParams()
	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	// Format: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	encodedHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		params.Memory,
		params.Iterations,
		params.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)

	return encodedHash, nil
}

// VerifyPassword verifies if a password matches the provided hash
func VerifyPassword(password, encodedHash string) bool {
	if len(password) == 0 || len(encodedHash) == 0 {
		return false
	}

	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false
	}

	var params Argon2Params
	_, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.Memory, &params.Iterations, &params.Parallelism)
	if err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	params.KeyLength = uint32(len(decodedHash))

	// Compute hash of the provided password with the same parameters
	computedHash := argon2.IDKey(
		[]byte(password),
		salt,
		params.Iterations,
		params.Memory,
		params.Parallelism,
		params.KeyLength,
	)

	// Constant-time comparison to prevent timing attacks
	return subtle.ConstantTimeCompare(decodedHash, computedHash) == 1
}
