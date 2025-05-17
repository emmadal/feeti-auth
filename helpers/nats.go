package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/emmadal/feeti-backend-user/models"
	"github.com/nats-io/nats.go"
	"log"
	"os"
	"sync"
	"time"
)

var (
	nc            *nats.Conn
	once          sync.Once
	subscriptions []*nats.Subscription
	subsMutex     sync.Mutex
	initDone      sync.WaitGroup
)

// NatsConfig holds the configuration options for NATS
type NatsConfig struct {
	URL           string
	MaxReconnects int
	ReconnectWait time.Duration
	Replicas      int
}

// ResponsePayload represents the standard response structure
type ResponsePayload struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

// RequestPayload represents the standard request structure
type RequestPayload struct {
	Data    string `json:"data"`
	Subject string `json:"subject"`
}

// defaultNatsConfig returns default configuration for NATS
func defaultNatsConfig() NatsConfig {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}
	return NatsConfig{
		URL:           natsURL,
		MaxReconnects: 60,
		ReconnectWait: 5 * time.Second,
		Replicas:      1,
	}
}

// NatsConnect initializes the NATS connection
func NatsConnect() error {
	var connectErr error

	// Signal that initialization is starting
	initDone.Add(1)

	once.Do(func() {
		// Load configuration from environment
		config := defaultNatsConfig()

		log.Println("Connecting to NATS server...")

		// Connect to NATS
		nc, connectErr = nats.Connect(
			config.URL,
			nats.RetryOnFailedConnect(true),
			nats.MaxReconnects(config.MaxReconnects),
			nats.ReconnectWait(config.ReconnectWait),
			nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
				log.Printf("NATS disconnected: %v\n", err)
			}),
			nats.ReconnectHandler(func(nc *nats.Conn) {
				log.Println("NATS reconnection attempt...")
			}),
			nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
				if sub != nil {
					log.Printf("NATS error on subject %s: %v\n", sub.Subject, err)
				} else {
					log.Printf("NATS error: %v\n", err)
				}
			}),
			nats.ClosedHandler(func(nc *nats.Conn) {
				log.Println("NATS connection closed")
			}),
		)

		if connectErr != nil {
			log.Printf("Failed to connect to NATS: %v\n", connectErr)
			// Signal that initialization has completed (with error)
			initDone.Done()
			return
		}
		log.Println("Successfully connected to NATS")

		// Only start subscribers if everything is set up correctly
		// Use a WaitGroup to track when all subscriptions are ready
		var subWg sync.WaitGroup
		subWg.Add(1) // We have 1 subscription

		go func() {
			// Catch subscription panics to prevent goroutine crashes
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Recovered from panic in NATS subscriptions: %v\n", r)
				}

				// Signal that initialization has completed
				initDone.Done()
			}()

			// Start all subscription handlers
			err1 := subscribeToGetUser(&subWg)

			// Check for errors
			for i, err := range []error{err1} {
				if err != nil {
					subject := ""
					switch i {
					case 0:
						subject = "auth.get.user"
					}
					log.Printf("Failed to subscribe to %s: %v\n", subject, err)
				}
			}

			// Wait for all subscriptions to be ready
			subWg.Wait()
			log.Println("All NATS subscriptions established")
		}()
	})

	return connectErr
}

// DrainNatsConnection drains and closes the NATS connection
func DrainNatsConnection(ctx context.Context) error {
	if nc == nil {
		return nil
	}

	// Create a channel to signal when draining is done
	done := make(chan error, 1)

	go func() {
		log.Println("Draining NATS connection and unsubscribing from all subjects...")

		// Lock the subscription list
		subsMutex.Lock()

		// Unsubscribe from each subscription
		for _, sub := range subscriptions {
			if sub != nil {
				if err := sub.Unsubscribe(); err != nil {
					log.Printf("Error unsubscribing from %s: %v", sub.Subject, err)
				} else {
					log.Printf("Unsubscribed from %s", sub.Subject)
				}
			}
		}

		// Clear the subscription list
		subscriptions = nil
		subsMutex.Unlock()

		// Drain the connection
		done <- nc.Drain()
	}()

	// Wait for drain to complete or context to be canceled
	select {
	case err := <-done:
		log.Println("NATS connection drained successfully")
		return err
	case <-ctx.Done():
		log.Println("NATS drain timeout, forcing close")
		nc.Close()
		return nil
	}
}

// RegisterSubscription adds a subscription to the tracked list
func RegisterSubscription(sub *nats.Subscription) {
	if sub == nil {
		return
	}

	subsMutex.Lock()
	defer subsMutex.Unlock()

	subscriptions = append(subscriptions, sub)
	log.Printf("Registered subscription on subject: %s", sub.Subject)
}

// subscribeToGetUser subscribes to the "wallet.get.user" subject
func subscribeToGetUser(wg *sync.WaitGroup) error {
	defer wg.Done()

	// Subscribe to the "auth.get.user" subject
	sub, err := nc.Subscribe("auth.get.user", func(msg *nats.Msg) {
		startTime := time.Now()
		log.Printf("Received message [%s] on subject %s\n", string(msg.Data), msg.Subject)

		// Add recovery to prevent crashes
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in wallet.create handler: %v\n", r)
				sendResponse(msg, ResponsePayload{
					Success: false,
					Error:   fmt.Sprintf("Internal server error: %v", r),
				})
			}
		}()

		// Get user by phone number
		phoneNumber := string(msg.Data)
		modelUser := models.User{PhoneNumber: phoneNumber}
		user, err := modelUser.GetUserByPhone()
		if err != nil {
			log.Printf("Failed to get user by phone number [%s]: %v\n", phoneNumber, err)
			sendResponse(msg, ResponsePayload{
				Success: false,
				Error:   fmt.Sprintf("Failed to get user by phone number [%s]: %v", phoneNumber, err),
			})
			return
		}
		log.Printf("Got user by phone number [%s] in %v\n", phoneNumber, time.Since(startTime))

		// Send success response
		sendResponse(msg, ResponsePayload{
			Success: true,
			Data:    user,
		})
	})

	if err != nil {
		return fmt.Errorf("failed to subscribe to auth.get.user: %w", err)
	}

	// Keep subscription active - don't auto-unsubscribe
	if err := sub.SetPendingLimits(-1, -1); err != nil {
		log.Printf("Failed to set pending limits for auth.get.user: %v\n", err)
	}

	// Register this subscription for cleanup
	RegisterSubscription(sub)

	return nil
}

// sendResponse sends a structured response to the NATS message
func sendResponse(msg *nats.Msg, payload ResponsePayload) {
	// If there's no reply subject, we can't respond
	if msg.Reply == "" {
		log.Println("No reply subject in message, cannot respond")
		return
	}

	// Marshal the response payload to JSON
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshaling response: %v\n", err)
		// Try to send a simplified error message
		errorMsg := []byte(`{"success":false,"error":"Failed to marshal response"}`)
		if pubErr := nc.Publish(msg.Reply, errorMsg); pubErr != nil {
			log.Printf("Failed to publish error response: %v\n", pubErr)
		}
		return
	}

	// Publish the response
	if err := nc.Publish(msg.Reply, response); err != nil {
		log.Printf("Failed to publish response: %v\n", err)
	} else {
		log.Printf("Response sent to %s: %t\n", msg.Reply, payload.Success)
	}
}

// PublishEvent sends a request to the NATS server
func (r *RequestPayload) PublishEvent() (*nats.Msg, error) {
	msg, err := nc.Request(r.Subject, []byte(r.Data), time.Second)
	if err != nil {
		return nil, fmt.Errorf("unable to publish message to create wallet: %v", err)
	}
	return msg, nil
}
