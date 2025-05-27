# Stage 1: Build the Go binary
FROM golang:1.24.3-alpine AS builder

WORKDIR /app

# Define build arguments
ARG PORT
ARG JWT_KEY
ARG HOST_URL
ARG NATS_URL
ARG GIN_MODE
ARG DATABASE_URL

# Set environment variables for build
ENV PORT=$PORT \
    JWT_KEY=$JWT_KEY \
    HOST_URL=$HOST_URL \
    GIN_MODE=$GIN_MODE \
    NATS_URL=$NATS_URL \
    DATABASE_URL=$DATABASE_URL

# Copy go.mod and go.sum first to cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the application source code
COPY . .

# Build the Go binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o auth-service .

# Stage 2: Create a minimal runtime image
FROM alpine:latest

# Create a non-root user and group
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Set up the application directory
WORKDIR /app

# Copy only the necessary binary from the builder stage
COPY --from=builder /app/auth-service /app/

# Set ownership and permissions
RUN chown -R appuser:appgroup /app && \
    chmod +x /app/auth-service

# Expose the port for the application
EXPOSE 4000

# Switch to the non-root user
USER appuser

# Run the Go binary
CMD ["/app/auth-service"]
