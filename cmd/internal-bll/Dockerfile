ARG GO_VERSION=1.26
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/internal-bll ./cmd/internal-bll

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /app/bin/internal-bll /usr/local/bin/internal-bll
USER nonroot:nonroot
EXPOSE 8086
ENTRYPOINT ["/usr/local/bin/internal-bll"]
