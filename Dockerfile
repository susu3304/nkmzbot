# Build stage
FROM golang:1.24.0 AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o nkmzbot cmd/nkmzbot/main.go

# Runtime stage
FROM ubuntu:24.04

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/nkmzbot .

# Copy migrations
COPY migrations ./migrations

# docker-compose.yml mounts the host IMM binary here.
ENV IMM_BINARY=/usr/local/bin/imm

# Run the application
CMD ["./nkmzbot"]
