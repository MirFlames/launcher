# Сборка

С self-signed сертификатом
powershell -NoProfile -ExecutionPolicy Bypass -File build.ps1 sign

# Launcher (Wails)

Example десктопное приложение на **Wails** (Go + React + TypeScript).

## Система сборки

Для Go/Wails **Gradle не используется**. Вместо него:

- **Go** — встроенная сборка (`go build`, `go mod`)
- **Wails CLI** — управляет сборкой приложения (`wails build`, `wails dev`)
- **npm** — сборка фронтенда (Vite)

Gradle подходит для Java/Kotlin. Для Go-проектов стандартны: `go build`, Makefile или Taskfile.

## Требования

- [Go](https://go.dev/dl/) 1.21+
- [Node.js](https://nodejs.org/) 18+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation): `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **Windows:** WebView2 (обычно уже установлен с Windows 10/11)

## Быстрый старт

```bash
cd client/launcher

# Установка зависимостей
cd frontend && npm install && cd ..

# Режим разработки (hot reload)
wails dev

# Сборка production
wails build
```

## Команды сборки

| Действие | Команда | Альтернатива (Windows) |
|----------|---------|-------------------------|
| Разработка | `wails dev` | `.\build.ps1 dev` |
| Сборка | `wails build` | `.\build.ps1 build` |
| Очистка | `make clean` | `.\build.ps1 clean` |

С Makefile (Git Bash, WSL, или `choco install make`):
```bash
make dev      # разработка
make build    # сборка
make clean    # очистка
```

## Структура проекта

```
client/launcher/
├── main.go          # Точка входа, конфигурация окна
├── app.go           # Бизнес-логика (биндинги Go -> JS)
├── wails.json       # Конфиг Wails
├── frontend/        # React + Vite + TypeScript
│   ├── src/
│   │   ├── App.tsx
│   │   └── main.tsx
│   └── wailsjs/     # Автогенерируемые биндинги
├── build/           # Иконки, манифесты для сборки
├── Makefile         # Удобные команды (dev, build, clean)
└── build.ps1        # То же для PowerShell
```

## Результат сборки

После `wails build`:
- **Windows:** `build/bin/launcher.exe`
- **macOS:** `build/bin/launcher.app`
- **Linux:** `build/bin/launcher`
