# Stage 1: Build the Go binary
FROM golang:1.24.3-alpine AS builder

WORKDIR /app

# Define build arguments
ARG PORT
ARG JWT_SECRET
ARG NEW_RELIC_LICENSE_KEY
ARG HOST
ARG GIN_MODE
ARG DATABASE_URL
ARG TWILIO_ACCOUNT_SID
ARG TWILIO_AUTH_TOKEN

# Set environment variables for build
ENV PORT=$PORT \
    JWT_KEY=$JWT_SECRET \
    NEW_RELIC_LICENSE_KEY=$NEW_RELIC_LICENSE_KEY \
    HOST=$HOST \
    GIN_MODE=$GIN_MODE \
    DATABASE_URL=$DATABASE_URL \
    TWILIO_ACCOUNT_SID=$TWILIO_ACCOUNT_SID \
    TWILIO_AUTH_TOKEN=$TWILIO_AUTH_TOKEN

# Copy go.mod and go.sum first to cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the application source code
COPY . .

# Build the Go binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o feeti-auth-service .

# Stage 2: Create a minimal runtime image
FROM alpine:latest

WORKDIR /app

# Copy only the necessary binary from the builder stage
COPY --from=builder /app/feeti-auth-service /app/

# Expose the port for the application
EXPOSE 4000

# Run the Go binary
CMD ["/app/feeti-auth-service"]
