ARG GO_VERSION=1.26
FROM golang:${GO_VERSION}-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download || true
COPY cmd/portal-server cmd/portal-server
RUN CGO_ENABLED=0 GOOS=linux go build -o /portal-server ./cmd/portal-server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=builder /portal-server /portal-server
COPY web/marketing/public /dist

ENV PORT=8080
ENV STATIC_DIR=/dist
USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/portal-server"]
