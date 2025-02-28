package helpers

import (
	"crypto/rand"

	"github.com/sirupsen/logrus"
)

// GenerateOTPCode generates a secure OTP of given length
func GenerateOTPCode(length int) string {
	if length <= 0 {
		logrus.WithFields(logrus.Fields{"length": length}).Error("Invalid OTP length")
		return "00000"
	}

	otp := make([]byte, length)
	_, err := rand.Read(otp)
	if err != nil {
		logrus.WithFields(logrus.Fields{"error": err}).Error("Random read failed")
		return "00000"
	}

	for i := range otp {
		otp[i] = (otp[i] % 10) + '0'
	}

	return string(otp)
}
