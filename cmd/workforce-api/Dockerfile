FROM golang:1.25-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/workforce-api ./cmd/workforce-api

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && addgroup -S oci && adduser -S oci -G oci
WORKDIR /app
COPY --from=builder /app/bin/workforce-api /usr/local/bin/workforce-api
USER oci
EXPOSE 50051
ENTRYPOINT ["/usr/local/bin/workforce-api"]
