FROM golang:1.25-alpine AS build
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN CGO_ENABLED=0 go build -o mc-proxy .

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=build /app/mc-proxy /mc-proxy
EXPOSE 25565 24454/udp
ENTRYPOINT ["/mc-proxy"]
