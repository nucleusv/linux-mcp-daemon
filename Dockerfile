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

# Docs Build Stage
FROM node:20-alpine AS docs-builder
WORKDIR /app/docs/website
# Copy only package files first for better caching
COPY docs/website/package.json docs/website/package-lock.json* ./
RUN npm ci || npm install
# Copy the rest of the documentation files
COPY docs/website .
# Build the Docusaurus site
RUN npm run build

# Use ubuntu instead of alpine for a full environment
FROM ubuntu:24.04

WORKDIR /root/

# Install some basic tools and certificates, and create OS accounts matching
# configs/daemon.yaml's users (SpawnWorker does user.Lookup() against the OS
# passwd db for every tool call, privileged or not, to resolve a UID)
RUN apt-get update && apt-get install -y \
    ca-certificates \
    sudo \
    curl \
    man-db \
    iproute2 \
    dnsutils \
    iputils-ping \
    net-tools \
    smartmontools \
    traceroute \
    file \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -m -s /bin/bash testuser \
    && useradd -m -s /bin/bash unpriviliged \
    && useradd -m -s /bin/bash privileged

# Copy the binary from the builder stage
COPY --from=builder /app/mcpd /usr/local/bin/
# Copy configs
COPY --from=builder /app/configs ./configs
# Copy compiled documentation website
COPY --from=docs-builder /app/docs/website/build ./docs/website/build
# Copy man pages
COPY --from=builder /app/docs/man/linuxctl.1 /usr/local/share/man/man1/
COPY --from=builder /app/docs/man/mcpd.8 /usr/local/share/man/man8/
RUN mandb

# Expose the default port
EXPOSE 9091

# Command to run the executable
CMD ["/usr/local/bin/mcpd"]
