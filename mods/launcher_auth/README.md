# Launcher Auth

Серверный мод Fabric для ограничения подключения к Minecraft-серверу только аутентифицированным через лаунчер игрокам.

## Требования

- Minecraft 1.21.11
- Fabric Loader 0.18.4+
- Fabric API

## Установка

1. Соберите мод: `gradlew :mods:launcher_auth:build`
2. Скопируйте JAR из `mods/launcher_auth/build/libs/launcher_auth-1.0.0.jar` в `mods/launcher_auth/` на сервере (или в `mods/` сервера)
3. Настройте `launcher_auth.properties` в JAR или создайте конфиг (см. ниже)

## Конфигурация

URL backend API задаётся в `launcher_auth.properties` (внутри JAR, можно переопределить):

```properties
auth_api_url=http://localhost:80
```

Для продакшена укажите реальный URL backend.

## Как это работает

1. Игрок входит через лаунчер (Telegram-бот)
2. Лаунчер передаёт в игру `--username` (никнейм) и `--uuid` (session_uuid)
3. При подключении к серверу мод вызывает `GET /api/auth/verify?nickname=X&session_uuid=Y`
4. Если backend возвращает `{"valid": true}` — игрок допускается
5. Иначе — отключение с сообщением «Доступ запрещён. Войдите через лаунчер.»

## Размещение

Мод размещается в `mods/launcher_auth/` (JAR внутри этой папки или сама папка в `mods/`).
