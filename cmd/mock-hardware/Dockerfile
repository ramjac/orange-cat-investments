FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/mock-hardware ./cmd/mock-hardware

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/bin/mock-hardware /usr/local/bin/mock-hardware

USER 1000:1000
ENTRYPOINT ["/usr/local/bin/mock-hardware"]
