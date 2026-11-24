ARG GO_VERSION=1.26
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/facilities-api ./cmd/facilities-api

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /app/bin/facilities-api /usr/local/bin/facilities-api
USER nonroot:nonroot
EXPOSE 50052
ENTRYPOINT ["/usr/local/bin/facilities-api"]
