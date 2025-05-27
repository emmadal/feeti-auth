# Auth Service

This service is responsible for managing user accounts and authentication for the Feeti platform.

## Features

- User registration and login
- PIN-based authentication
- User account management (update PIN, reset PIN, deactivate account, etc.)

## Technology Stack

- Go as the programming language
- Gin as the web framework
- NATS for message queueing

## Running the Service

### Prerequisites

- Go 1.18+
- Docker
- Docker Compose
- A PostgreSQL database
- A NATS server for message queueing

## Development

### Running Tests

1. Run `go test ./...` to run all tests

### Building the Service

1. Run `go build main.go` to build the service
2. Run `docker build -t auth-service .` to build a Docker image of the service
