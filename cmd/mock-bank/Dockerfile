ARG GO_VERSION=1.26
FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/mock-bank ./cmd/mock-bank

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/bin/mock-bank /usr/local/bin/mock-bank

RUN adduser -D -g "" -u 1000 oci
USER oci
ENTRYPOINT ["/usr/local/bin/mock-bank"]
