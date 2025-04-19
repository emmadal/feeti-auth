package helpers

import (
	"fmt"
	"os"

	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

func SendOTP(phoneNumber string, otp string) {
	client := twilio.NewRestClientWithParams(
		twilio.ClientParams{
			Username: os.Getenv("TWILIO_ACCOUNT_SID"),
			Password: os.Getenv("TWILIO_AUTH_TOKEN"),
		},
	)

	message := fmt.Sprintf(
		"Féeti: Votre code de vérification est %s. Ne partagez ce code avec personne. Il expire dans 2 minutes.", otp,
	)

	params := &twilioApi.CreateMessageParams{}
	params.SetTo(phoneNumber)
	params.SetFrom("+15202264216")
	params.SetBody(message)

	_, err := client.Api.CreateMessage(params)
	if err != nil {
		Logger.Error().Time().Err("err", err).Send()
	}
}
