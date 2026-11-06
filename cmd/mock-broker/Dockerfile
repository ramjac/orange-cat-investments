FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/mock-broker ./cmd/mock-broker

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/bin/mock-broker /usr/local/bin/mock-broker

RUN adduser -D -g "" -u 1000 oci
USER oci
ENTRYPOINT ["/usr/local/bin/mock-broker"]
