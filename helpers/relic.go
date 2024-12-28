package helpers

import (
	"os"

	"github.com/newrelic/go-agent/v3/newrelic"
)

func NewRelicConfig() (*newrelic.Application, error) {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(os.Getenv("RELIC_APP_NAME")),
		newrelic.ConfigLicense(os.Getenv("RELIC_LICENSE_KEY")),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	return app, err
}
