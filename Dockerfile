# Build stage
FROM golang:alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
# Download all dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o mcpd ./cmd/mcpd

# Final stage
# Use alpine instead of scratch to have some basic tools like sh if we need to exec in,
# but it still keeps it very small.
FROM alpine:latest

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/mcpd .
# Copy configs
COPY --from=builder /app/configs ./configs

# Expose the default port
EXPOSE 9090

# Command to run the executable
CMD ["./mcpd"]
