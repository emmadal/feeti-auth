# Feeti User Service

This service is responsible for managing user accounts and authentication for the Feeti platform.

## Features

- User registration and login
- PIN-based authentication
- Two-factor authentication using one-time passwords (OTPs) sent via SMS
- User account management (update PIN, reset PIN)

## Technology Stack

- Go as the programming language
- Gin as the web framework
- New Relic for application performance monitoring
- Twilio for SMS-based two-factor authentication
- NATS for message queueing

## Running the Service

### Prerequisites

- Go 1.18+
- Docker
- Docker Compose
- A PostgreSQL database
- A Twilio account for SMS-based two-factor authentication
- A New Relic account for application performance monitoring
- A NATS server for message queueing

## Development

### Running Tests

1. Run `go test ./...` to run all tests

### Building the Service

1. Run `go build main.go` to build the service
2. Run `docker build -t feeti-user-service .` to build a Docker image of the service

## Contributing

Contributions are welcome! Please open a pull request with your changes.