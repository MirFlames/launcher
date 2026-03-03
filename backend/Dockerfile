# Бинарник собирается на хосте (build.ps1), в образ копируется готовый файл
FROM alpine:3.21
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY launcher-backend /app/launcher-backend

# Data directory for config.json, sessions.db
RUN mkdir -p /data /app/files

WORKDIR /data
ENV FILES_PATH=/app/files

EXPOSE 8080
ENTRYPOINT ["/app/launcher-backend"]
