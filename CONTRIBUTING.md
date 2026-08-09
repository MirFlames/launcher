# Разработка

Документ для разработчиков и ИИ-агентов, работающих с этим репозиторием.
Пользовательская документация — в [README.md](README.md).

---

## Содержание

- [Стек и структура](#стек-и-структура)
- [Требования](#требования)
- [Быстрый старт](#быстрый-старт)
- [Переменные окружения](#переменные-окружения)
- [Сборка](#сборка)
- [Тесты](#тесты)
- [Архитектура лаунчера](#архитектура-лаунчера)
- [Автообновление](#автообновление)
- [Установка JDK](#установка-jdk)
- [Пути на диске](#пути-на-диске)
- [CI/CD](#cicd)
- [Выпуск релиза](#выпуск-релиза)
- [Грабли](#грабли)
- [TODO](#todo)

---

## Стек и структура

| Слой | Технология |
|------|------------|
| Десктоп-оболочка | [Wails v2](https://wails.io) 2.11 |
| Бэкенд лаунчера | Go 1.23 |
| Фронтенд | React 18 + TypeScript + Vite |
| 3D | Three.js |
| Установщик Windows | Inno Setup 6 |
| CI/CD | GitHub Actions |

**Gradle для лаунчера не используется** — у Go своя система сборки (`go build`, `go mod`),
а сборкой приложения управляет Wails CLI. Gradle в репозитории отвечает только за Java-моды.

```
launcher/
├── .github/workflows/
│   └── launcher-release.yml     # сборка под 3 ОС + публикация релиза
├── backend/                     # серверный API (отдельный компонент)
├── mods/                        # Java-моды (Gradle)
└── client/launcher/             # сам лаунчер
    ├── main.go                  # точка входа, конфигурация окна Wails
    ├── app.go                   # биндинги Go → JS
    ├── auth.go                  # авторизация + Yggdrasil-сессия
    ├── config.go                # загрузка/сохранение конфига
    ├── version.go               # LauncherVersion
    ├── updater.go               # проверка и загрузка обновлений
    ├── updater_windows.go       # применение обновления (bat-скрипт)
    ├── updater_other.go         # применение обновления (sh-скрипт)
    ├── jdk.go                   # скачивание и распаковка OpenJDK
    ├── jdk_test.go              # тесты распаковки на синтетических архивах
    ├── launch.go                # оркестрация запуска Minecraft
    ├── launch_windows.go        # SysProcAttr для Windows
    ├── launch_other.go          # SysProcAttr для Unix
    ├── downloader.go            # загрузка файлов с прогрессом и проверкой хешей
    ├── modpack.go               # модпак + getLauncherDir()
    ├── frontend/                # React + Vite
    ├── installer/launcher.iss   # Inno Setup
    ├── build.ps1                # сборка под Windows
    └── Makefile                 # сборка под Unix
```

---

## Требования

- [Go](https://go.dev/dl/) 1.23+
- [Node.js](https://nodejs.org/) 20+
- [Wails CLI](https://wails.io/docs/gettingstarted/installation):
  `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

Дополнительно по платформам:

| ОС | Что нужно |
|----|-----------|
| Windows | WebView2 (обычно уже стоит в Windows 10/11) |
| macOS | Xcode Command Line Tools |
| Linux | `libgtk-3-dev` и `libwebkit2gtk-4.1-dev` (на Ubuntu ≤22.04 — `libwebkit2gtk-4.0-dev`) |

---

## Быстрый старт

```bash
cd client/launcher
cd frontend && npm install && cd ..
wails dev          # режим разработки с hot reload
```

После изменения сигнатур методов в `app.go` пересоберите биндинги для фронтенда:

```bash
wails generate module
```

---

## Переменные окружения

Секреты лежат в `.env` в корне репозитория. Скопируйте `.env.example` в `.env` и заполните.

Значения из `.env` **встраиваются в бинарник** через `-ldflags -X` на этапе сборки —
они нужны клиенту, у которого ещё нет своего `launcher-config.json`:

| Переменная | Назначение |
|------------|-----------|
| `API_BASE_URL` | адрес backend API |
| `SERVER_HOST`, `SERVER_PORT` | адрес игрового сервера |
| `SOCKS_PROXY_HOST`, `SOCKS_PROXY_PORT` | прокси (опционально) |
| `UPDATE_MANIFEST_URL` | URL манифеста обновления |
| `UPDATE_SIGNATURE_URL` | URL подписи манифеста |
| `UPDATE_PUBLIC_KEY_HEX` | публичный ключ Ed25519 для проверки подписи |
| `CODESIGN_PFX`, `CODESIGN_PASSWORD` | подпись exe под Windows |

Соответствующие Go-переменные объявлены в `build_defaults.go` (`buildDefaultApiBaseUrl`,
`buildUpdateManifestURL` и т.д.). Если переменная не задана при сборке, соответствующая
функциональность просто отключается — например, пустой `UPDATE_MANIFEST_URL` выключает
автообновление, а не роняет лаунчер.

---

## Сборка

### Windows

```powershell
cd client/launcher
.\build.ps1 build      # сборка + подпись, если настроен сертификат
.\build.ps1 dev        # то же, что wails dev
.\build.ps1 sign       # подписать уже собранный exe
.\build.ps1 clean
```

`build.ps1` сам читает `.env` из корня (или `client/launcher/.sign.env`), собирает `ldflags`
и обновляет иконку из `frontend/src/assets/images/appicon.png`.

### macOS / Linux

```bash
cd client/launcher
make build-macos     # wails build -platform darwin/amd64
make build-linux     # wails build -platform linux/amd64
make clean
```

В CI macOS собирается как `darwin/universal` — см. раздел [Грабли](#грабли).

### Результат

| ОС | Путь |
|----|------|
| Windows | `build/bin/launcher.exe` |
| macOS | `build/bin/launcher.app` |
| Linux | `build/bin/launcher` |

### Java-моды (Gradle)

```powershell
.\gradlew :mods:launcher_auth:build            # требует AUTH_API_URL из .env
.\gradlew :mods:launcher_auto_connect:build    # секретов не требует
```

### Backend

```powershell
cd backend && go run .
```

Ищет `.env` в `backend/` или в корне. Секреты нужны только при запуске, не при сборке.

### Docker

Используйте `.\compose.ps1` вместо `docker compose` — при `build` он сначала собирает
backend на хосте, это заметно быстрее.

```powershell
.\compose.ps1 build
.\compose.ps1 up -d
```

---

## Тесты

```bash
cd client/launcher
go test ./...
```

`jdk_test.go` проверяет распаковку JDK на **синтетических архивах**, которые собираются
в памяти прямо в тесте — сеть и реальные JDK не нужны. Покрыты: раскладка Windows/Linux,
бандл macOS, симлинки, определение формата по сигнатуре, zip-slip, диагностируемость ошибки.

Проверки exec-бита и симлинков осмысленны только на Unix — на Windows они вырождаются
в no-op, поэтому `go test` дополнительно гоняется в CI на macOS и Linux.

Кросс-компиляция как быстрая проверка, что ничего не разъехалось по платформам:

```bash
GOOS=windows go build ./... && GOOS=darwin go build ./... && GOOS=linux go build ./...
```

---

## Архитектура лаунчера

### Поток запуска игры

`app.go: PlayMinecraft()` → `launch.go: LaunchMinecraft()`:

1. `getLauncherDir()` — определяет папку с игровыми данными (зависит от ОС)
2. `ensurePrerequisites()` → `EnsureJDK()` — Java, конфиг, модпак, версия сервера
3. `ensureGameFiles()` — моды, библиотеки, `client.jar`, natives, ассеты
4. `buildClasspath()` — собирает classpath через `os.PathListSeparator`
5. `resolveJvmArguments()` — подставляет `${natives_directory}`, `${classpath}`
6. `spawnMinecraftProcess()` — запускает java с `sysProcAttrForLaunch`

Прогресс уходит на фронтенд событием `launch-progress`, завершение игры — `launch-ended`.

### Биндинги Go → JS (`app.go`)

`GetConfig`, `SaveConfig`, `AuthStartLogin`, `AuthIsAuthenticated`, `AuthGetSession`,
`AuthRefreshSession`, `AuthLogout`, `PlayMinecraft`, `CheckLauncherUpdate`,
`ApplyLauncherUpdate`, `GetNewsFeed`, `GetLauncherVersion`.

### Авторизация (`auth.go`)

`POST /api/auth/init` → код → `browser.OpenURL()` для подтверждения →
поллинг `GET /api/auth/check?code=` до подтверждения → `GET /api/auth/verify` →
Yggdrasil-сессия (`AccessToken`, `ProfileID`, `ProfileName`) для запуска игры.

---

## Автообновление

Механизм подписан Ed25519 — клиент не доверяет ничему, что не прошло проверку.

**Проверка** (`updater.go: CheckLauncherUpdate`):

1. Скачивается манифест `launcher-update-<platform>.json` и подпись `.sig` (hex)
2. `ed25519.Verify` по вшитому `buildUpdatePublicKeyHex` — при несовпадении всё отбрасывается
3. Версии сравниваются `compareVersions()`
4. Если `min_mandatory_version` выше текущей версии клиента — обновление становится
   **обязательным**, даже если сам релиз помечен опциональным (значит, пропущен критический апдейт)

**Применение** — разное по ОС, поэтому вынесено в отдельные файлы:

| Файл | Как работает |
|------|--------------|
| `updater_windows.go` | скачивает рядом `launcher.new.exe`, запускает скрытый bat-скрипт: ждёт освобождения файла → подменяет → перезапускает |
| `updater_other.go` | то же через sh-скрипт с `Setpgid`, чтобы процесс пережил завершение родителя |

Обновление всегда «на месте», чтобы ярлыки продолжали указывать на рабочий бинарник.

Формат манифеста — структура `LauncherUpdateManifest` в `updater.go`.

---

## Установка JDK

`jdk.go`. Java не берётся ни из `JAVA_HOME`, ни из системного `PATH` — только своя копия
в папке данных, чтобы версия была предсказуемой.

1. `fetchJDKInfo()` — `GET /api/jdk/info` у backend
2. `normalizeJDKInfo()` — приводит ответ к текущей ОС (backend отдаёт всем один
   Windows-ориентированный JSON, см. [Грабли](#грабли))
3. `checkJDKExists()` — если java уже на месте, дальше ничего не делается
4. `downloadJDK()` — Adoptium API, `archive_type` подбирается по ОС
5. `extractJDKArchive()` — формат определяется **по сигнатуре первых байт**, не по расширению
6. Распаковка в staging-папку → перенос на место целиком (прерванная установка не оставляет
   полу-JDK, который потом сойдёт за готовый)

Нормализация дерева приводит раскладку к единому виду `<targetDir>/bin/java[.exe]` на всех ОС:
срезается общий корневой каталог, а у сборок macOS дополнительно `Contents/Home`
(служебные `Contents/Info.plist` и `Contents/MacOS` отбрасываются — нужен только JAVA_HOME).

Права доступа и симлинки переносятся из архива: **без exec-бита java не запустится вовсе**.

---

## Пути на диске

**Игровые данные** (`getLauncherDir()` в `modpack.go`) — JDK, versions, mods, assets:

| ОС | Путь |
|----|------|
| Windows | рядом с exe |
| macOS | `~/Library/Application Support/minecraft-online` |
| Linux | `$XDG_DATA_HOME/minecraft-online` или `~/.local/share/minecraft-online` |

На Windows путь намеренно оставлен прежним: у существующих пользователей там лежат
гигабайты файлов, и менять его нельзя. На macOS путь рядом с бинарником вёл бы внутрь
`launcher.app/Contents/MacOS/`, а запись в бандл ломает подпись приложения и стирается
при каждом обновлении.

**Конфиг и логи** (`config.go`, `logger.go`) — через `os.UserConfigDir()`:

| ОС | Путь |
|----|------|
| Windows | `%APPDATA%\FamMCLauncher` |
| macOS | `~/Library/Application Support/FamMCLauncher` |
| Linux | `~/.config/FamMCLauncher` |

Логи — в подпапке `logs`, текущий запуск всегда `latest.log`, предыдущие ротируются
в `YYYY-MM-DD-N.log`.

---

## CI/CD

`.github/workflows/launcher-release.yml`, запускается по пушу тега `v*`.

```
meta ──┬── build-windows ──┐
       ├── build-macos ────┼── release
       └── build-linux ────┘
```

| Job | Что делает |
|-----|-----------|
| `meta` | разбирает тег: версия, флаг `mandatory`, поиск последней `-critical` версии через GitHub API |
| `build-windows` | `build.ps1 build` → Inno Setup → `launcher-windows.zip` + `minecraft-online-setup.exe` |
| `build-macos` | `wails build -platform darwin/universal` → `go test` → `launcher-macos.zip` + `.dmg` |
| `build-linux` | `wails build -platform linux/amd64` → `go test` → `launcher-linux.zip` + `.tar.gz` |
| `release` | считает SHA256, генерирует и подписывает манифесты, создаёт GitHub Release |

Каждая платформа собирается со **своим** `UPDATE_MANIFEST_URL`, указывающим на
`launcher-update-<platform>.json`.

### Секреты репозитория

| Секрет | Назначение |
|--------|-----------|
| `API_BASE_URL`, `SERVER_HOST`, `SERVER_PORT` | встраиваются в бинарник |
| `LAUNCHER_UPDATE_PRIVATE_KEY` | приватный ключ Ed25519 (PEM) для подписи манифестов |
| `LAUNCHER_UPDATE_PUBLIC_KEY_HEX` | публичный ключ, вшивается в клиент |
| `CODESIGN_PFX_BASE64`, `CODESIGN_PASSWORD` | подпись под Windows (опционально) |
| `APPLE_CERT_BASE64`, `APPLE_CERT_PASSWORD` | подпись под macOS (опционально) |

Подписи опциональны: без секретов сборка проходит, просто без подписи.

---

## Выпуск релиза

1. **Поднять версию в двух местах** — их легко рассинхронизировать:
   - `client/launcher/version.go` → `const LauncherVersion = "1.0.44"`
   - `client/launcher/wails.json` → `"productVersion": "1.0.44"`

2. Закоммитить и запушить `main`.

3. Создать и запушить тег:
   ```bash
   git tag v1.0.44              # обычный релиз
   git tag v1.0.44-critical     # обязательный к установке
   git push origin v1.0.44-critical
   ```

4. Дождаться зелёного workflow в **Actions**.

Суффикс `-critical` ставит в манифесте `mandatory: true` — клиент покажет только кнопку
«Обновить», без возможности отложить. У обычных релизов заполняется
`min_mandatory_version` (последняя `-critical` версия): клиент ниже неё считает
обновление обязательным, даже если сам релиз опциональный.

**Перевыпуск того же тега** (если CI упал на середине):

```bash
git tag -d v1.0.44-critical
git push origin :refs/tags/v1.0.44-critical
git tag v1.0.44-critical
git push origin v1.0.44-critical
```

### Артефакты релиза

`launcher-windows.zip`, `minecraft-online-setup.exe`, `launcher-macos.zip`,
`launcher-macos.dmg`, `launcher-linux.zip`, `launcher-linux.tar.gz`,
по паре `launcher-update-<platform>.json` + `.sig` на каждую платформу,
плюс `launcher-update.json` + `.sig` — алиас копии Windows-манифеста.

**Алиас удалять нельзя.** В клиентах ≤1.0.40 URL манифеста вшит в бинарник и указывает
именно на `launcher-update.json`; без этого файла они навсегда потеряют возможность обновиться.

---

## Грабли

Собранные на практике неочевидности. Прочитайте перед тем, как трогать соответствующий код.

<details>
<summary><b>Adoptium не отдаёт zip под macOS и Linux</b></summary>

<br>

Под macOS доступны только `.pkg` и `.tar.gz`, под Linux — только `.tar.gz`. Запрос
`archive_type=zip` возвращает **HTTP 200 с tar.gz внутри**, а не ошибку, — поэтому раньше
падало на `zip.OpenReader` с невнятным `zip: not a valid zip file`.

Отсюда правило: `extractJDKArchive` определяет формат по сигнатуре первых байт, а не по
расширению и не по тому, что мы запрашивали.

</details>

<details>
<summary><b>У macOS-сборок JDK лишний уровень вложенности</b></summary>

<br>

Корень архива — `jdk-21.x/Contents/Home/bin/java`, на сегмент глубже Windows и Linux.
Наивный «срезать первый сегмент» ломается. См. `jdkStripPrefix()`.

</details>

<details>
<summary><b>Запись корневого каталога ломает вычисление префикса</b></summary>

<br>

Реальные архивы Adoptium содержат запись самого корневого каталога (`jdk-21/` со слэшем
на конце). После `path.Clean` слэш пропадает, запись выглядит как файл в корне архива и
обнуляет вычисление общего префикса — файлы распаковываются на уровень глубже.

`jdkStripPrefix()` отличает такие записи по исходному суффиксу `/`. Случай закрыт тестом.

</details>

<details>
<summary><b>Backend отдаёт один статичный JSON всем ОС</b></summary>

<br>

`GET /api/jdk/info` не смотрит ни на query-параметры, ни на User-Agent и всегда возвращает
`{"version":"21","relative_path":"jre_default\\jdk","java_executable":"bin\\java.exe"}`.

Backend живёт в отдельном репозитории и зеркалится вручную, поэтому лаунчер приводит ответ
к своей ОС сам — `normalizeJDKInfo()`. Не полагайтесь на то, что backend когда-нибудь
станет OS-aware.

</details>

<details>
<summary><b>OpenSSL 3.x требует -rawin для Ed25519</b></summary>

<br>

Без флага `openssl pkeyutl -sign` падает с `operation not supported for this keytype`.
Ed25519 хеширует данные внутри себя, и OpenSSL 3.x требует явно сказать, что вход
передаётся сырым. Go-сторона (`ed25519.Verify`) как раз и ждёт подпись сырых данных.

</details>

<details>
<summary><b>Ubuntu 24.04 переименовала пакет WebKit</b></summary>

<br>

`libwebkit2gtk-4.0-dev` → `libwebkit2gtk-4.1-dev`. `ubuntu-latest` в GitHub Actions уже
24.04, поэтому в workflow стоит автоопределение через `apt-cache`, а Wails получает
тег сборки `-tags webkit2_41`.

</details>

<details>
<summary><b>Поля SysProcAttr не совпадают между платформами</b></summary>

<br>

`HideWindow` есть только в Windows-версии структуры, `Setpgid` — только в Unix-версии.
Это ошибка **компиляции**, а не рантайма, поэтому код с ними обязан лежать в файлах
с build-тегами: `updater_windows.go` / `updater_other.go`, `launch_windows.go` /
`launch_other.go`.

</details>

<details>
<summary><b>macOS нужно собирать как universal</b></summary>

<br>

Под `darwin/amd64` на Apple Silicon лаунчер идёт через Rosetta, `runtime.GOARCH` рапортует
`amd64`, и `downloadJDK` тянет x64-JDK — вся игра поедет через трансляцию.
С `darwin/universal` `GOARCH` даёт честный `arm64`.

</details>

<details>
<summary><b>go test не работает, пока не собран фронтенд</b></summary>

<br>

`main.go` содержит `//go:embed all:frontend/dist`, а `frontend/dist` лежит в `.gitignore`.
Значит **пакет вообще не компилируется**, пока фронтенд не собран, — и `go test` падает
не с ошибкой теста, а с ошибкой сборки.

Локально это незаметно: `dist` остаётся от прошлых запусков. В CI после чистого checkout
папки нет, поэтому шаг `Run Go tests` обязан идти **после** `wails build` (он и запускает
`npm run build`). Однажды это уже уронило релиз.

Если нужно прогнать тесты в чистом дереве:

```bash
cd client/launcher/frontend && npm ci && npm run build && cd ..
go test ./...
```

</details>

<details>
<summary><b>go build мусорит в каталоге пакета</b></summary>

<br>

`go build ./...` в `client/launcher` кладёт бинарник рядом с исходниками: `launcher.exe`
на Windows и `launcher` на Unix. Причём `client/launcher/launcher.exe` **отслеживается
git'ом**, так что сборка показывает его изменённым. Проверяйте `git status` перед коммитом
и не утаскивайте бинарник случайно.

</details>

<details>
<summary><b>Версия живёт в двух файлах</b></summary>

<br>

`version.go` и `wails.json`. CI берёт версию из тега и рассинхрон не заметит — разъедутся
молча.

</details>

---

## TODO

- [ ] Перейти с HTTP на HTTPS
- [ ] Ускорить загрузку файлов игры
- [ ] Нотаризация сборки под macOS (сейчас пользователи видят предупреждение Gatekeeper)
- [x] Кроссплатформенная сборка и установщики под Windows/macOS/Linux
- [x] Автоматические тесты в CI
- [x] Перевод пользователей со старых лаунчеров без автообновления
- [x] Админ-панель на отдельном закрытом порту
- [x] Единый конфигурационный файл для смены окружения
- [x] Полная синхронизация модов сервера и клиента
- [x] CI/CD через GitHub Actions
- [x] Логи в папке `logs`, один файл на запуск
