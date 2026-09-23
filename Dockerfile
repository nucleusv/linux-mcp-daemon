# Build stage. Runs on the build machine's own platform ($BUILDPLATFORM) and
# cross-compiles for the target - Go does that natively, so multi-arch
# release builds never compile under slow QEMU emulation.
FROM --platform=$BUILDPLATFORM golang:alpine AS builder

# Set automatically by BuildKit to match the build's target platform (arm64
# on Docker Desktop/Apple Silicon, amd64 on a typical x86_64 Linux host) -
# this was previously hardcoded to arm64, which silently produced a binary
# that wouldn't run on an amd64 target.
ARG TARGETARCH
# Release builds pass these (see .github/workflows/release.yml); local
# builds report "dev".
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
# Download all dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags "-s -w -X github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/version.Version=${VERSION} -X github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/version.Commit=${COMMIT} -X github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/version.Date=${BUILD_DATE}" \
    -o mcpd ./cmd/mcpd
# linuxctl too: it's how users and tokens are created, even for a container
# (docker run ... linuxctl create mcpd user ...).
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags "-s -w -X github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/version.Version=${VERSION} -X github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/version.Commit=${COMMIT} -X github.com/nucleusv/linux-mcp-daemon-by-antigravity/internal/version.Date=${BUILD_DATE}" \
    -o linuxctl ./cmd/linuxctl

# Docs Build Stage - static output, so it too runs on the build platform.
FROM --platform=$BUILDPLATFORM node:20-alpine AS docs-builder
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
COPY --from=builder /app/mcpd /app/linuxctl /usr/local/bin/
# Copy configs. Defaults to the repo's configs/ (the local Kubernetes dev
# setup, with its test users). Release images pass
# CONFIG_DIR=packaging/configs-container: no users, so a public image never
# ships the test tokens that are published in this repo's docs.
ARG CONFIG_DIR=configs
COPY --from=builder /app/${CONFIG_DIR} ./configs
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
