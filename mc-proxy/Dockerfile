# Бинарник собирается на хосте (compose.ps1 / build-mc-proxy.ps1)
FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY mc-proxy /mc-proxy
EXPOSE 25565 24454/udp
ENTRYPOINT ["/mc-proxy"]
