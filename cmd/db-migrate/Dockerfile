ARG GO_VERSION=1.26
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/db-migrate ./cmd/db-migrate

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /app/bin/db-migrate /usr/local/bin/db-migrate
COPY --from=builder /app/scripts/migrations /app/scripts/migrations
USER nonroot:nonroot
ENV MIGRATIONS_DIR=/app/scripts/migrations
ENTRYPOINT ["/usr/local/bin/db-migrate"]
