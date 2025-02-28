package helpers

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func SendOTP(c *gin.Context, phoneNumber string, otp string) error {
	payload := []byte(fmt.Sprintf(
		`{"recipient": "%s", "sender_id": "%s", "type": "plain", "message": "Utilise le code %s pour te connecter à Feeti. Il Expire dans 2 minutes."}`, strings.Split(phoneNumber, "+")[1], "Feeti", otp))

	req, err := http.NewRequestWithContext(c, "POST", os.Getenv("SMS_API_URL"), bytes.NewBuffer(payload))
	if err != nil {
		return errors.New("Unable to contact the sms provider")
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+os.Getenv("SMS_JWT"))

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return errors.New("Unable to send the OTP")
	}
	defer res.Body.Close()
	return nil
}
