# Launcher

## Сборка и переменные окружения

Секреты и настройки хранятся в `.env` в корне проекта. Скопируйте `.env.example` в `.env` и заполните значения.

### Backend

Backend загружает `.env` при запуске (ищет `backend/.env` или `launcher/.env`).

```powershell
cd backend
go run .
```

Или из корня:

```powershell
go run ./backend
```

Секреты нужны только при запуске, не при `go build`.

### Launcher (Wails)

Скрипт `build.ps1` читает `.env` из корня или `client/launcher/.sign.env`:
- **Подпись:** `CODESIGN_PFX`, `CODESIGN_PASSWORD`
- **Дефолты при сборке** (встраиваются в exe): `API_BASE_URL`, `SERVER_HOST`, `SERVER_PORT` — используются когда у клиента нет launcher-config.json

```powershell
cd client/launcher
.\build.ps1 build
```

### Моды (Gradle)

**launcher_auth** — требует `AUTH_API_URL` из `.env` (плагин dotenv читает корневой `.env`).

```powershell
.\gradlew :mods:launcher_auth:build
```

**launcher_auto_connect** — секретов при сборке не требует.

```powershell
.\gradlew :mods:launcher_auto_connect:build
```

**Оба мода сразу:**

```powershell
.\gradlew :mods:launcher_auth:build :mods:launcher_auto_connect:build
```

### Сводка

| Компонент | Команда | Источник .env |
|-----------|---------|---------------|
| Backend | `go run .` (из `backend/`) | `backend/.env` или `launcher/.env` |
| Launcher | `.\build.ps1 build` (из `client/launcher/`) | `launcher/.env` или `client/launcher/.sign.env` |
| launcher_auth | `.\gradlew :mods:launcher_auth:build` (из корня) | `launcher/.env` |
| launcher_auto_connect | `.\gradlew :mods:launcher_auto_connect:build` | не требуется |

---

### TODO
- [done] Вынести хосты подключения в один конфигурационный файл для простоты смены окружения разработки и прод
- [done] Полная синхронизация модов сервера и клиента (удаление, добавление, обновление)
- Создание комфортного CI/CD для минимизации ручных действий при нововведениях (попробовать github actions)
- Внедрить сборку и автоматические тесты (используя github actions, например)
- Перейти с HTTP на HTTPS
- Доработать логирование: логи должны храниться в папке logs, 1 лог файл = 1 запуск лаунчера. Нейминг лог-файлов: YYYY-MM-DD-NUM.log, где NUM - порядковый номер лог-файла, если за день было несколько запусков лаунчера.
- Ускорить загрузку файлов игры


Подписывайтесь на тгк :) https://t.me/mc_fam