ARG GO_VERSION=1.26
# Multi-stage build for Orange Cat Investments - Employee BFF
FROM golang:${GO_VERSION}-alpine AS builder

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
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /app/bin/oci-employee-bff /usr/local/bin/oci-employee-bff
COPY config/ /app/config/

USER nonroot:nonroot

EXPOSE 8081

ENTRYPOINT ["/usr/local/bin/oci-employee-bff"]
