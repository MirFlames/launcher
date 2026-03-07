# mc-proxy

Легковесный прокси для Minecraft (TCP) и Simple Voice Chat (UDP) на Go.

## Назначение

- **Minecraft (25565/TCP)** — пересылка с проверкой авторизации (репликация сессий по HTTP с backend)
- **Simple Voice Chat (24454/UDP)** — пересылка UDP-пакетов с поддержкой нескольких клиентов

Авторизация обязательна: mc-proxy не запускается без сессий. При подключении парсится Login Start (nickname, session_uuid) через [go-mc](https://github.com/Tnze/go-mc), проверка по реплике с backend. mc-proxy может быть на отдельном VPS — сессии загружаются по HTTP (GET /api/sessions/export) каждые 30 сек. Мод **launcher_auth** на mc-сервере при использовании прокси можно удалить.

Полезно, когда:
- mc-сервер в Docker-сети, наружу нужен один хост
- Нужно скрыть реальный IP сервера
- Авторизация ближе к клиенту (без HTTP-запросов с mc-сервера)

## Переменные окружения

| Переменная       | По умолчанию | Описание                         |
|------------------|--------------|----------------------------------|
| `MC_BACKEND`     | `mc:25565`   | Адрес Minecraft-сервера          |
| `VOICE_BACKEND`  | `mc:24454`   | Адрес Simple Voice Chat          |
| `MC_LISTEN`      | `:25565`     | Адрес прослушивания TCP          |
| `VOICE_LISTEN`   | `:24454`     | Адрес прослушивания UDP          |
| `SESSIONS_API_URL` | — | URL backend для репликации, например `https://launcher.example.com/api/sessions/export` |
| `SESSIONS_API_TOKEN` | — | Токен (тот же что SESSIONS_EXPORT_TOKEN в backend). Заголовок Authorization: Bearer |
| `BAN_IP`           | —             | Начальный бан: IP через запятую |
| `BAN_FILE`         | `bans.txt`    | Файл для хранения банов         |
| `MC_MAX_CONNECTIONS` | `200`       | Глобальный лимит одновременных соединений |
| `MC_MAX_BACKEND_DIALS` | `50`     | Лимит одновременных dial к mc-серверу (семафор) |
| `MC_MAX_CONNECTIONS_PER_IP` | `3` | Лимит соединений с одного IP (2–3 для игрока) |

## Запуск

### Локально

```bash
cd mc-proxy
go run .
```

### Docker

Бинарник собирается на хосте (как backend). Из корня проекта:

```powershell
# Сборка (compose.ps1 собирает mc-proxy на хосте перед docker build)
.\compose.ps1 -f docker-compose.yml -f docker-compose.proxy.yml build

# Или отдельно
.\build-mc-proxy.ps1

# Запуск
docker compose -f docker-compose.yml -f docker-compose.proxy.yml up -d
```

### Без прокси (как раньше)

```bash
docker compose up -d
```

## mc-proxy на отдельном VPS

Backend и mc-proxy могут быть на разных хостах. На VPS с mc-proxy задайте:

```
SESSIONS_API_URL=https://your-backend.example.com/api/sessions/export
SESSIONS_API_TOKEN=<секрет из SESSIONS_EXPORT_TOKEN backend>
MC_BACKEND=<IP:port Minecraft-сервера>
VOICE_BACKEND=<IP:port Simple Voice Chat>
```

Backend должен быть доступен по HTTPS (или HTTP в доверенной сети). Токен — один и тот же в backend и mc-proxy.

## Конфигурация лаунчера

При использовании прокси клиент подключается к хосту, где запущен mc-proxy. Порт остаётся 25565 (и 24454 для голоса). Менять `server_host` и `server_port` в API/лаунчере не нужно, если прокси на том же хосте.

## Удаление launcher_auth при использовании прокси

При работе через mc-proxy мод launcher_auth на mc-сервере не нужен — проверка сессии выполняется на прокси. Удалите `launcher_auth-*.jar` из `mc-server/data/mods/`.

## Бан и логирование

- **Подозрительные пакеты** (ошибка handshake, login start, неожиданный packet id, ошибка парсинга) → IP добавляется в бан.
- **Формат логов**: `[MC] CONNECT ip=... addr=... reason=... ts=... details`
- **Причины** (reason): `CONNECT`, `BANNED`, `RATE_LIMIT`, `GLOBAL_LIMIT`, `PER_IP_LIMIT`, `SUSPICIOUS`, `AUTH_FAIL`, `AUTH_OK`, `SUCCESS`, `DISCONNECT`, `STATUS_PING` (ping для списка серверов, без авторизации), `BACKEND_DIAL_ERROR`, `BACKEND_WRITE_ERROR`
- **Начальный бан**: `BAN_IP=1.2.3.4,5.6.7.8`
- **Файл банов**: `BAN_FILE` (по умолчанию `bans.txt`), один IP на строку, `#` — комментарий
- **Rate limit**: 20 подключений с одного IP за 10 секунд, при превышении — RATE_LIMIT
