# Launcher Auto Connect

Клиентский Fabric-мод для автоматического подключения к серверу при запуске Minecraft через лаунчер.

## Описание

При появлении главного меню (Title Screen) мод автоматически подключается к серверу, указанному в `configs/launcher-config.json`. Это позволяет игрокам сразу попадать на сервер без необходимости вручную выбирать его в списке.

## Конфигурация

Мод читает настройки из `configs/launcher-config.json` (тот же файл, что использует лаунчер):

```json
{
  "api_base_url": "http://localhost:8080",
  "server_host": "localhost",
  "server_port": 25565
}
```

- **server_host** — хост или IP-адрес сервера (обязательно для работы автоподключения)
- **server_port** — порт сервера (по умолчанию 25565)

Если `server_host` не указан или пустой, автоподключение отключается.

## Сборка

```bash
./gradlew :mods:launcher_auto_connect:build
```

JAR-файл будет в `mods/launcher_auto_connect/build/libs/launcher_auto_connect-1.0.0.jar`.

## Совместимость

- Minecraft 1.21.11
- Fabric Loader 0.18.4+
- Fabric API
