# Документация проекта Launcher

## Разделы

| Документ | Описание |
|----------|----------|
| [modpack-specification.md](./modpack-specification.md) | Спецификация `modpack.json` — полный манифест версии Minecraft |
| [launcher-development.md](./launcher-development.md) | Руководство по разработке лаунчера, архитектура, конфиги |
| [README.md](./README.md) | API Backend и Swagger UI (ниже) |

---

# API документация Backend

OpenAPI 3.0 спецификация для тестирования хендлеров Backend в браузере.

## Структура

```
docs/
├── openapi.yaml           # Главный файл спецификации
├── modpack-specification.md  # Спецификация modpack.json
├── launcher-development.md   # Руководство по разработке
├── index.html             # Swagger UI для интерактивного тестирования
├── paths/                 # Описание эндпоинтов
│   ├── version.yaml
│   ├── launcher.yaml
│   ├── jdk.yaml
│   └── files.yaml
└── schemas/               # Схемы данных
    ├── common.yaml
    ├── version.yaml
    ├── launcher.yaml
    └── jdk.yaml
```

## Запуск

### 1. Запустить Backend

```bash
cd backend
go run main.go
```

По умолчанию сервер слушает порт 8080 (или из `config.json`).

### 2. Открыть Swagger UI

Из папки `docs/`:

```bash
cd docs
go run .
```

Или из корня проекта:

```bash
go run ./docs
```

По умолчанию сервер слушает порт 3000. Переменная `PORT` меняет порт.

Откройте в браузере: **http://localhost:3000**

### 3. Настроить server URL

В Swagger UI выберите сервер (по умолчанию `http://localhost:8080`) или задайте свой (например `http://localhost:80`).

## Эндпоинты

| Метод | Путь | Описание |
|-------|------|----------|
| GET | /api/version | Версия Minecraft, моды, client_files |
| GET | /api/launcher/version | Информация об обновлении лаунчера |
| GET | /api/jdk/info | Конфигурация JDK |
| GET | /files/{path} | Скачивание файла |
