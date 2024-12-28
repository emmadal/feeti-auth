package helpers

import (
	"math/rand"
)

// GenerateOTPCode generate otp code for authentication
func GenerateOTPCode(maxNum int) string {
	otpCode := make([]byte, maxNum)
	for i := range otpCode {
		otpCode[i] = byte(rand.Intn(10) + '0')
	}
	return string(otpCode)
}
