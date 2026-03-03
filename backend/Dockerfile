# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o launcher-backend .

# Runtime stage
FROM alpine:3.21
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /build/launcher-backend /app/launcher-backend

# Data directory for config.json, sessions.db
RUN mkdir -p /data /app/files

WORKDIR /data
ENV FILES_PATH=/app/files

EXPOSE 8080
ENTRYPOINT ["/app/launcher-backend"]
