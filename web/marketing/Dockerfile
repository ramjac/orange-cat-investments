FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download || true
COPY cmd/portal-server cmd/portal-server
RUN CGO_ENABLED=0 GOOS=linux go build -o /portal-server ./cmd/portal-server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY --from=builder /portal-server /portal-server
COPY web/marketing/public /dist

ENV PORT=80
ENV STATIC_DIR=/dist
EXPOSE 80

ENTRYPOINT ["/portal-server"]
