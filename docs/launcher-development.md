# Руководство по разработке лаунчера

Документация для разработчиков Minecraft-лаунчера.

## Архитектура проекта

```
launcher/
├── launcher/          # Java-приложение лаунчера (Swing)
├── updater/           # Модуль обновления (отдельный JAR)
├── backend/           # Go API-сервер (версии, JDK, файлы)
├── docs/              # Документация
└── launcher/resources/configs/
    └── modpack.json              # Манифест версии (Mojang/Fabric)
```

---

## Поток данных при запуске

```
1. Пользователь нажимает "Играть"
2. JDKManager.ensureJDKInstalled() → проверка/установка Java
3. MinecraftLauncher.startMinecraft():
   - Требуется configs/modpack.json → launchFromModpack():
     * ModpackConfigLoader.load() → modpack.json
     * Сборка classpath: client.jar + libraries (с учётом rules по ОС)
     * Извлечение natives в папку natives/
     * Разрешение JVM и game аргументов с плейсхолдерами
     * Запуск ProcessBuilder(java, jvmArgs, mainClass, gameArgs)
   - При отсутствии modpack.json → ошибка
```

---

## Конфигурационные файлы

### modpack.json

**Используется:** приоритетно при наличии в `configs/modpack.json`. Если файл есть — лаунчер запускает игру по этой конфигурации (формат Mojang/Fabric).  
Подробнее: [modpack-specification.md](./modpack-specification.md)

**Требования для modpack:**
- Client JAR: `versions/{id}/{id}.jar` или `versions/{id}/client.jar` (при отсутствии скачивается из `downloads.client.url`)
- Библиотеки: `libraries/` (пути из `artifact.path`). Отсутствующие скачиваются из `artifact.url`
- Natives: извлекаются из `*-natives-*.jar` в папку `natives/`

---

## Backend API

| Endpoint | Описание |
|----------|----------|
| GET /api/version | Версия Minecraft, client_files, mods, mods_hash |
| GET /api/launcher/version | Обновление лаунчера |
| GET /api/jdk/info | Конфигурация JDK |
| GET /files/{path} | Скачивание файлов |

Конфиг backend: `config.json` (генерируется при первом запуске).

---

## Ключевые классы лаунчера

| Класс | Назначение |
|-------|------------|
| `ModpackConfigLoader` | Загрузка modpack.json, разрешение rules и аргументов |
| `MinecraftConfigLoader` | Определение папки Minecraft (getMinecraftFolder) |
| `MinecraftLauncher` | Сборка команды и запуск процесса по modpack.json |
| `NativesExtractor` | Извлечение natives из JAR в папку natives/ |
| `LibraryDownloader` | Скачивание отсутствующих библиотек и client.jar по URL из modpack |
| `JDKManager` | Проверка/установка JDK |
| `UpdateManager` | Проверка обновлений лаунчера |
| `LauncherFrame` | UI (Swing) |

---

## Плейсхолдеры modpack.json → подстановка

| Плейсхолдер | Источник |
|-------------|----------|
| `${auth_player_name}` | Имя игрока (оффлайн/онлайн) |
| `${version_name}` | ID версии |
| `${game_directory}` | Папка Minecraft |
| `${assets_root}` | Папка assets |
| `${assets_index_name}` | ID индекса ассетов |
| `${auth_uuid}` | UUID игрока |
| `${auth_access_token}` | Токен (онлайн) |
| `${auth_xuid}` | XUID (Xbox) |
| `${version_type}` | fabric / vanilla |
| `${natives_directory}` | Папка natives |
| `${classpath}` | Собранный classpath |
| `${launcher_name}`, `${launcher_version}` | Бренд лаунчера |

---

## Сборка и развёртывание

- **Лаунчер:** `./gradlew :launcher:jar` или native image
- **Updater:** `./gradlew :updater:jar`
- **Backend:** `go run ./backend` или `go build -o backend.exe ./backend`

Конфиги копируются в `build/libs/configs/` при сборке лаунчера.
