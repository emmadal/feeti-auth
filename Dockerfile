# Stage 1: Build the Go binary
FROM golang:1.23.6-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Create a limited-permissions user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Change the ownership of the working directory to the new user
RUN chown -R appuser:appgroup /app

# Switch to the non-root user
USER appuser

# Copy the Go modules files and download the dependencies first (this will help with caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o backend-user .

# Stage 2: Create a smaller image to run the Go app
FROM alpine:latest

# Set working directory in runtime image
WORKDIR /app

# Copy the binary from the builder
COPY --from=builder /app/backend-user /app/

# Expose the port on which the Gin API will run
EXPOSE 4000

# Run the Go binary
CMD ["./backend-user"]
