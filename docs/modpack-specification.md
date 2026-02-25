# Спецификация modpack.json

Документация формата конфигурации модпака для разработки лаунчера.

## Обзор

`modpack.json` — полный манифест версии Minecraft в формате, совместимом с Mojang/Fabric. Файл описывает всё необходимое для запуска клиента: библиотеки, аргументы JVM и игры, загрузки, ассеты.

**Расположение:** `launcher/resources/configs/modpack.json`

---

## Структура верхнего уровня

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | string | Идентификатор версии (например, `"modpack"`) |
| `time` | string | ISO 8601 дата/время сборки |
| `releaseTime` | string | ISO 8601 дата релиза |
| `type` | string | Тип версии: `"modified"`, `"release"`, `"snapshot"` |
| `mainClass` | string | Главный класс для запуска (Fabric: `net.fabricmc.loader.impl.launch.knot.KnotClient`) |
| `minimumLauncherVersion` | number | Минимальная версия лаунчера (21) |
| `arguments` | object | JVM и игровые аргументы |
| `assets` | string | ID индекса ассетов (например, `"29"`) |
| `libraries` | array | Список библиотек |
| `assetIndex` | object | Метаданные индекса ассетов |
| `downloads` | object | URL клиента, сервера, маппингов |
| `complianceLevel` | number | Уровень совместимости (1.0) |
| `javaVersion` | object | Требования к Java |
| `logging` | object | Конфигурация логирования |

---

## arguments

### jvm (аргументы JVM)

Массив объектов с полями:
- `values` — массив строк (аргументы)
- `rules` (опционально) — условия применения

**Плейсхолдеры:**
- `${natives_directory}` — путь к нативным библиотекам
- `${classpath}` — classpath
- `${launcher_name}` — имя лаунчера
- `${launcher_version}` — версия лаунчера

**Примеры правил по ОС:**
```json
{
  "values": ["-XstartOnFirstThread"],
  "rules": [{"action": "allow", "os": {"name": "osx"}}]
}
```

### game (аргументы игры)

Формат `--key value`. Плейсхолдеры:
- `${auth_player_name}` — имя игрока
- `${version_name}` — версия
- `${game_directory}` — папка игры
- `${assets_root}` — корень ассетов
- `${assets_index_name}` — индекс ассетов
- `${auth_uuid}`, `${auth_access_token}`, `${clientid}`, `${auth_xuid}`
- `${version_type}` — тип версии (fabric, vanilla и т.д.)
- `${resolution_width}`, `${resolution_height}` — разрешение (если `has_custom_resolution`)
- `${quickPlayPath}`, `${quickPlaySingleplayer}`, `${quickPlayMultiplayer}`, `${quickPlayRealms}` — Quick Play

### default_user_jvm

Массив дополнительных JVM-аргументов по умолчанию (часто пустой).

---

## libraries

Каждая библиотека:

```json
{
  "name": "groupId:artifactId:version[:classifier]",
  "rules": [{"action": "allow", "os": {"name": "windows"}}],
  "artifact": {
    "sha1": "...",
    "size": 12345,
    "path": "path/to/file.jar",
    "url": "https://..."
  }
}
```

- `rules` — опционально; без правил библиотека загружается на всех ОС
- `artifact.path` — относительный путь в папке `libraries/`
- Источники: `libraries.minecraft.net`, `res.tlauncher.org`, и др.

**Типичные группы:**
- `net.fabricmc` — Fabric Loader, intermediary, sponge-mixin
- `org.lwjgl` — LWJGL (графика, звук)
- `com.mojang` — authlib, brigadier, datafixerupper
- `io.netty` — сеть
- `org.ow2.asm` — ASM

**Natives** — библиотеки с classifier (`natives-windows`, `natives-linux`, `natives-macos`, `natives-macos-arm64`, `natives-windows-arm64`, `natives-windows-x86`).

---

## assetIndex

```json
{
  "id": "29",
  "totalSize": 439074397,
  "size": 529372,
  "url": "https://piston-meta.mojang.com/v1/packages/.../29.json"
}
```

Индекс ассетов (текстуры, звуки и т.д.) для версии.

---

## downloads

```json
{
  "client": {
    "sha1": "...",
    "size": 31152600,
    "url": "https://piston-data.mojang.com/v1/objects/.../client.jar"
  },
  "client_mappings": {...},
  "server": {...},
  "server_mappings": {...}
}
```

- `client` — JAR клиента Minecraft
- `client_mappings` — маппинги для разработки
- `server`, `server_mappings` — для серверной части

---

## javaVersion

```json
{
  "component": "java-runtime-delta",
  "majorVersion": 21.0
}
```

Требуемая версия Java (21).

---

## logging

```json
{
  "client": {
    "argument": "-Dlog4j.configurationFile=${path}",
    "file": {
      "id": "client-1.21.2.xml",
      "sha1": "...",
      "size": 1073,
      "url": "https://..."
    },
    "type": "log4j2-xml"
  }
}
```

Конфигурация логирования (Log4j2).
