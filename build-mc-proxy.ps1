# Сборка mc-proxy: компиляция на хосте + docker build
# Запуск из корня проекта: .\build-mc-proxy.ps1

Set-Location $PSScriptRoot\mc-proxy

$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o mc-proxy .

if ($LASTEXITCODE -ne 0) {
    Write-Error "Ошибка сборки Go"
    exit 1
}

Set-Location $PSScriptRoot
docker compose -f docker-compose.yml -f docker-compose.proxy.yml build mc-proxy
