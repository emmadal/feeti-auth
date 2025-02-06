package helpers

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

func SmsAccountLocked(phoneNumber string) {

	payload := []byte(fmt.Sprintf(
		`{"recipient": "%s", "sender_id": "%s", "type": "plain", "message": "Votre compte Feeti a été temporairement bloqué. Ceci est due au nombre de tentatives de connexion erronées. Veuillez contacter le support 124."}`, strings.Split(phoneNumber, "+")[1], "Feeti"))

	req, err := http.NewRequest("POST", os.Getenv("SMS_API_URL"), bytes.NewBuffer(payload))
	if err != nil {
		log.Println("Unable to send the contact sms provider")
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+os.Getenv("SMS_JWT"))

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Println("Unable to send the contact sms provider ")
	}
	defer res.Body.Close()
}
