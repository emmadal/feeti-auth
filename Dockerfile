# Stage 1: Build the Go binary
FROM golang:1.24.0-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first to cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the application source code
COPY . .

# Build the Go binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o backend-user .

# Stage 2: Create a minimal runtime image
FROM scratch  

WORKDIR /app

# Copy only the necessary binary from the builder stage
COPY --from=builder /app/backend-user /app/

# Expose the port for the application
EXPOSE 4000

# Run the Go binary
CMD ["/app/backend-user"]
