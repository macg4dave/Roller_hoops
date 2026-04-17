# Combined Go + Node devtools image.
#
# Provides an interactive shell with all project toolchains inside Docker,
# so no local Go/Node/npm install is needed.
#
# Build:   docker build -f docker/devtools.Dockerfile -t roller-devtools .
# Run:     docker run --rm -it -v "$(pwd)":/workspace -w /workspace roller-devtools sh
# Or:      make devtools

# --- Stage 1: grab the Go toolchain and build Go-based tools ---
FROM golang:1.24-alpine AS go-tools

RUN apk add --no-cache git
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.17.1
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.28.0

# --- Stage 2: Node 20 base with everything merged ---
FROM node:20-alpine

# Copy the full Go toolchain from the golang image.
COPY --from=go-tools /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:/root/go/bin:${PATH}"
ENV GOPATH="/root/go"

# Copy pre-built Go tools (migrate, sqlc).
COPY --from=go-tools /go/bin/migrate /usr/local/bin/migrate
COPY --from=go-tools /go/bin/sqlc /usr/local/bin/sqlc

# Install common dev utilities.
RUN apk add --no-cache \
    git \
    curl \
    jq \
    bash \
    make \
    grep \
    postgresql16-client

# Install openapi-typescript globally so it can be used outside npm context.
RUN npm install -g openapi-typescript@7.9.1

# Warm Go module cache for core-go.
WORKDIR /cache/core-go
COPY core-go/go.mod core-go/go.sum ./
RUN go mod download

# Warm npm cache for ui-node.
WORKDIR /cache/ui-node
COPY ui-node/package.json ui-node/package-lock.json ./
RUN npm ci

WORKDIR /workspace

LABEL org.opencontainers.image.title="roller-devtools"
LABEL org.opencontainers.image.description="Dev shell with Go, Node, psql, migrate, sqlc for Roller_hoops"
