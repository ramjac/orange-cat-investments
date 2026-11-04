FROM golang:1.25-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/internal-bll ./cmd/internal-bll

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && addgroup -S oci && adduser -S oci -G oci
WORKDIR /app
COPY --from=builder /app/bin/internal-bll /usr/local/bin/internal-bll
USER oci
EXPOSE 8086
ENTRYPOINT ["/usr/local/bin/internal-bll"]
