# syntax=docker/dockerfile:1

# Stage 1: Build binary
FROM golang:alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compile pure static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/aurahub-backend main.go

# Stage 2: Minimal runtime image
FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates curl

COPY --from=builder /app/aurahub-backend /app/aurahub-backend
COPY --from=builder /app/db /app/db

EXPOSE 8080

HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/api/health || exit 1

ENTRYPOINT ["/app/aurahub-backend"]
