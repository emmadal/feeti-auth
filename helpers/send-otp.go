package helpers

import (
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

func SendOTP(c *gin.Context, phoneNumber string, otp string) {
	client := twilio.NewRestClientWithParams(twilio.ClientParams{
		Username: os.Getenv("TWILIO_ACCOUNT_SID"),
		Password: os.Getenv("TWILIO_AUTH_TOKEN"),
	})

	message := fmt.Sprintf("Féeti: Votre code de vérification est %s. Ne partagez ce code avec personne. Il expire dans 2 minutes.", otp)

	params := &twilioApi.CreateMessageParams{}
	params.SetTo(phoneNumber)
	params.SetFrom("+15202264216")
	params.SetBody(message)

	_, err := client.Api.CreateMessage(params)
	if err != nil {
		logrus.WithFields(logrus.Fields{"error": err}).Error("Error sending SMS message to " + phoneNumber)
	} else {
		logrus.WithFields(logrus.Fields{"info": "SMS message sent to " + phoneNumber}).Info("SMS message sent")
	}
}
