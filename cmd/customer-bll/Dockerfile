ARG GO_VERSION=1.26
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/customer-bll ./cmd/customer-bll

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=builder /app/bin/customer-bll /usr/local/bin/customer-bll
USER nonroot:nonroot
EXPOSE 8085
ENTRYPOINT ["/usr/local/bin/customer-bll"]
