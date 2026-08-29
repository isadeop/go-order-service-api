# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS builder

ARG SERVICE=order-service

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/service ./cmd/${SERVICE}

FROM alpine:3.20

RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 app

WORKDIR /app
COPY --from=builder --chown=app:app /out/service ./service
USER app
EXPOSE 8080
ENTRYPOINT ["./service"]
