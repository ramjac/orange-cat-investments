# Multi-stage build for Orange Cat Investments - Unified Domain API Server
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

# Cache Go module dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source tree
COPY . .

# Build statically-linked stripped binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/oci-server ./cmd/server

# Minimal production runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S oci && adduser -S oci -G oci

WORKDIR /app

COPY --from=builder /app/bin/oci-server /usr/local/bin/oci-server
COPY config/ /app/config/

USER oci

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/oci-server"]
