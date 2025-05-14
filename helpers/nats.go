package helpers

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

var (
	once sync.Once
	nc   *nats.Conn
)

type NatsConfig struct {
	URL           string
	MaxReconnects int
	ReconnectWait time.Duration
	Replicas      int
}

// RequestResponse is the response structure for NATS requests
type RequestResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// defaultNatsConfig returns the default NATS configuration
func defaultNatsConfig() NatsConfig {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	return NatsConfig{
		URL:           natsURL,
		MaxReconnects: 5,
		ReconnectWait: 5 * time.Second,
		Replicas:      1,
	}
}

func NatsConnect() error {
	var connectErr error

	once.Do(func() {
		// Load configuration from environment
		config := defaultNatsConfig()

		// Connect to NATS
		nc, connectErr = nats.Connect(
			config.URL,
			nats.RetryOnFailedConnect(true),
			nats.MaxReconnects(config.MaxReconnects),
			nats.ReconnectWait(config.ReconnectWait),
			nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
				log.Println("NATS disconnected")
			}),
			nats.ReconnectHandler(func(nc *nats.Conn) {
				log.Printf("NATS reconnection attempt")
			}),
			nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
				fmt.Printf("Nats error: %v\n", err)
			}),
			nats.ClosedHandler(func(nc *nats.Conn) {
				log.Println("NATS connection closed")
			}),
		)

		if connectErr != nil {
			fmt.Printf("Failed to connect to NATS: %v\n", connectErr)
			return
		}
		fmt.Println("Successfully connected to NATS")

		// Only start subscribers if everything is set up correctly
		if connectErr == nil {

		}
	})

	return connectErr
}

// DrainNatsConnection drains and closes the NATS connection
func DrainNatsConnection() error {
	if nc == nil {
		return nil
	}
	log.Println("Draining NATS connection...")
	return nc.Drain()
}

type ProducerMessage struct {
	Subject string `json:"subject"`
	Data    string `json:"data"`
}

// WalletEvent produces a wallet event
func (p *ProducerMessage) WalletEvent() (*nats.Msg, error) {
	msg, err := nc.Request(p.Subject, []byte(p.Data), time.Second)
	if err != nil {
		return nil, fmt.Errorf("Unable to publish message to create wallet: %v", err)
	}
	return msg, nil
}
