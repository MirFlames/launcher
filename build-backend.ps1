# Сборка backend: компиляция на хосте + docker build
# Запуск из корня проекта: .\build-backend.ps1

Set-Location $PSScriptRoot\backend

# Сборка для Linux (amd64)
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o launcher-backend .

if ($LASTEXITCODE -ne 0) {
    Write-Error "Ошибка сборки Go"
    exit 1
}

Set-Location $PSScriptRoot
docker compose build backend
