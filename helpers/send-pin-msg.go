package helpers

import (
	"os"

	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

// SendPinMessage sends a message to the user when their PIN is updated
func SendPinMessage(phoneNumber string) {
	client := twilio.NewRestClientWithParams(
		twilio.ClientParams{
			Username: os.Getenv("TWILIO_ACCOUNT_SID"),
			Password: os.Getenv("TWILIO_AUTH_TOKEN"),
		},
	)
	params := &twilioApi.CreateMessageParams{}
	params.SetTo(phoneNumber)
	params.SetFrom("+15202264216")
	params.SetBody(
		`Le mot de passe de votre compte Feeti a été modifié. Si ce n'est pas vous,
		veuillez contacter le support client au 1313.`,
	)

	_, err := client.Api.CreateMessage(params)
	if err != nil {
		Logger.Error().Time().Err("err", err).Send()
	}
}
