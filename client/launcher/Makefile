# Wails Launcher - Makefile
# Для Go/Wails не используется Gradle — Go имеет встроенную систему сборки (go build, go mod).
# Wails CLI (wails build, wails dev) управляет сборкой приложения.
#
# Использование: make [target]
# На Windows: используйте Git Bash, WSL или "make" из Chocolatey (choco install make)

.PHONY: dev build build-prod build-macos build-linux clean install deps

# Запуск в режиме разработки (hot reload)
dev:
	wails dev

# Сборка production бинарника
build:
	wails build

# Сборка с минификацией и оптимизацией
build-prod:
	wails build -clean

# Сборка под macOS (запускать на macOS)
build-macos:
	wails build -platform darwin/amd64 -ldflags "-s -w" -trimpath

# Сборка под Linux (запускать на Linux или в Docker)
build-linux:
	wails build -platform linux/amd64 -ldflags "-s -w" -trimpath

# Очистка артефактов сборки
clean:
	rm -rf build/bin
	rm -rf frontend/dist
	go clean

# Установка зависимостей (Go + npm)
install: deps

deps:
	go mod download
	cd frontend && npm install

# Генерация биндингов Go -> TypeScript (после изменения app.go)
bindings:
	wails generate module
