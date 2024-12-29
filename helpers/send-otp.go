package helpers

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

func SendOTP(phoneNumber string, otp string) {

	payload := []byte(fmt.Sprintf(
		`{"recipient": "%s", "sender_id": "%s", "type": "plain", "message": "Utilise le code %s pour te connecter à Feeti. Il Expire dans 2 minutes."}`, strings.Split(phoneNumber, "+")[1], os.Getenv("SMS_SENDER_ID"), otp))

	req, err := http.NewRequest("POST", os.Getenv("SMS_API_URL"), bytes.NewBuffer(payload))
	if err != nil {
		log.Println("Unable to send the contact sms provider ")
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+os.Getenv("SMS_JWT"))

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Println("Unable to send the OTP")
	}
	defer res.Body.Close()
}
