ARG GO_VERSION=1.26
FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/mock-broker ./cmd/mock-broker

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /app/bin/mock-broker /usr/local/bin/mock-broker

USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/mock-broker"]
