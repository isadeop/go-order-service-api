# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/order-service-api ./cmd

FROM alpine:3.20

RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 app

WORKDIR /app
COPY --from=builder --chown=app:app /out/order-service-api ./order-service-api
USER app
EXPOSE 8080
ENTRYPOINT ["./order-service-api"]
