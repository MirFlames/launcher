# Loki + Prometheus — логи и метрики

Стек: **Loki** (логи) + **Promtail** (сбор логов) + **Prometheus** (метрики) + **Grafana** (визуализация).

Дополнительно: **mc-monitor** (метрики Minecraft), **cAdvisor** (метрики контейнеров).

## Запуск

Из корня проекта:

```bash
docker compose up -d
```

## Доступ

- **Grafana**: http://localhost:3000 (логин: `admin`, пароль: `admin`)
- **Prometheus**: http://localhost:9090
- **Loki API**: http://localhost:3100
- **cAdvisor**: http://localhost:8180

## Источники данных в Grafana

Loki и Prometheus подключаются автоматически при первом запуске.

## Метрики

### Backend (Go)
- `http_requests_total` — запросы по endpoint, method, status
- `http_request_duration_seconds` — время ответа
- `auth_init_total`, `auth_complete_total`, `auth_verify_total` — аутентификация
- `auth_sessions_active` — активные сессии
- `news_requests_total` — запросы новостей
- `file_downloads_total`, `file_download_bytes_total` — скачивания

### Minecraft (mc-monitor)
- `minecraft_status_players_online_count` — игроков онлайн
- `minecraft_status_players_max_count` — максимум игроков
- `minecraft_status_healthy` — 1 = сервер отвечает, 0 = нет
- `minecraft_status_response_time_seconds` — время ответа сервера

### Контейнеры (cAdvisor)
- CPU, память, сеть по контейнерам

## Просмотр логов (Loki)

1. Explore → Loki
2. LogQL: `{container=~".*mc.*"}` или `{container=~".*backend.*"}`

**Примечание:** mc-proxy не имеет label `logging=promtail` — иначе при перезапуске Promtail выдаёт флуд "No such container". `refresh_interval` увеличен до 15s.

## Важно для Windows

На Windows с Docker Desktop проверьте, что Docker использует WSL2. Для cAdvisor путь `/dev/disk` может отсутствовать — при ошибках можно убрать этот volume из compose.
