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

# Use ubuntu instead of alpine for a full environment
FROM ubuntu:24.04

WORKDIR /root/

# Install some basic tools and certificates, and create testuser
RUN apt-get update && apt-get install -y \
    ca-certificates \
    sudo \
    curl \
    man-db \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -m -s /bin/bash testuser

# Copy the binary from the builder stage
COPY --from=builder /app/mcpd .
# Copy configs
COPY --from=builder /app/configs ./configs
# Copy man pages
COPY --from=builder /app/docs/man/linuxctl.1 /usr/local/share/man/man1/
COPY --from=builder /app/docs/man/mcpd.8 /usr/local/share/man/man8/
RUN mandb

# Expose the default port
EXPOSE 9090

# Command to run the executable
CMD ["./mcpd"]
