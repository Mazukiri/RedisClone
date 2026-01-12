# Build Stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o redis-clone cmd/main.go

# Run Stage
FROM alpine:latest

WORKDIR /root/

# Copy binary from builder
COPY --from=builder /app/redis-clone .

# Expose ports
# 8082: TCP Server (Redis Protocol)
# 9090: HTTP Metrics & Dashboard
EXPOSE 8082 9090

# Run the binary
CMD ["./redis-clone"]
