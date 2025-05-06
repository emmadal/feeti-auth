package helpers

import (
	"crypto/rand"
	"fmt"
)

// GenerateOTPCode generates a secure OTP of a given length
func GenerateOTPCode(length int) (string, error) {
	if length <= 0 {
		return "00000", fmt.Errorf("invalid OTP length: must be greater than 0")
	}
	otp := make([]byte, length)
	_, err := rand.Read(otp)
	if err != nil {
		return "00000", fmt.Errorf("failed to generate secure random OTP: %w", err)
	}
	for i := range otp {
		otp[i] = (otp[i] % 10) + '0'
	}
	return string(otp), nil
}
