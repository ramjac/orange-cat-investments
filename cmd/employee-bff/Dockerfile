# Multi-stage build for Orange Cat Investments - Employee BFF
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

# Cache Go module dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source tree
COPY . .

# Build statically-linked stripped binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/oci-employee-bff ./cmd/employee-bff

# Minimal production runtime
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S oci && adduser -S oci -G oci

WORKDIR /app

COPY --from=builder /app/bin/oci-employee-bff /usr/local/bin/oci-employee-bff
COPY config/ /app/config/

USER oci

EXPOSE 8081

ENTRYPOINT ["/usr/local/bin/oci-employee-bff"]
