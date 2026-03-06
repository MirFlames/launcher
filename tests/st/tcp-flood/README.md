# tcp-flood — нагрузочное тестирование mc-proxy

Go-утилита для нагрузочного тестирования mc-proxy: открывает множество TCP-соединений, имитирует Minecraft-клиент (Handshake + Login Start) и измеряет поведение прокси.

## Сборка

```bash
go build -o tcp-flood .
```

## Использование

```bash
# Базовый тест: 100 подключений, 20 параллельно
tcp-flood -target localhost:25565 -connections 100 -concurrent 20

# Режим status (handshake next_state=1) — без авторизации
tcp-flood -target localhost:25565 -mode status -connections 500 -concurrent 50

# Режим login (handshake next_state=2) — с проверкой сессии
tcp-flood -target localhost:25565 -mode login -connections 200 -uuid <session_uuid>

# Агрессивный тест
tcp-flood -target localhost:25565 -connections 1000 -concurrent 100 -timeout 5s
```

## Параметры

| Параметр      | По умолчанию | Описание                          |
|---------------|--------------|-----------------------------------|
| `-target`     | localhost:25565 | Адрес mc-proxy                  |
| `-connections`| 100          | Общее число подключений           |
| `-concurrent` | 20           | Параллельных горутин              |
| `-mode`       | login        | `login` или `status`               |
| `-nickname`   | LoadTest     | Никнейм для login                 |
| `-uuid`       | 00000000-... | session_uuid (должен быть в sessions.db для успешной авторизации) |
| `-timeout`    | 10s          | Таймаут на подключение            |

## Режимы

- **status** — отправляет Handshake с next_state=1, затем Status Request. Прокси пробрасывает без авторизации.
- **login** — отправляет Handshake с next_state=2 и Login Start. Прокси проверяет сессию; при невалидном uuid — AUTH_FAIL.

## Результаты

Утилита выводит: число успешных подключений, ошибок, общее время и скорость (conn/s).
