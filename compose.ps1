# Обёртка над docker compose: при build сначала собирает backend на хосте
# Использование: .\compose.ps1 build | up | down | ...

param([Parameter(ValueFromRemainingArguments = $true)] $Args)

$isBuild = $Args -contains "build"
# Собирать backend: "build" без сервиса (все) или "build backend"
$buildBackend = $isBuild -and (
    ($Args.Count -le 1) -or
    ($Args[1] -eq "backend") -or
    ($Args -contains "backend")
)
# Собирать mc-proxy при любом build (используется при -f docker-compose.proxy.yml)
$buildMcProxy = $isBuild

if ($buildBackend) {
    Push-Location $PSScriptRoot\backend
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    go build -o launcher-backend .
    $goExit = $LASTEXITCODE
    Pop-Location
    if ($goExit -ne 0) {
        Write-Error "Ошибка сборки backend"
        exit 1
    }
}

if ($buildMcProxy) {
    Push-Location $PSScriptRoot\mc-proxy
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    go build -o mc-proxy .
    $goExit = $LASTEXITCODE
    Pop-Location
    if ($goExit -ne 0) {
        Write-Error "Ошибка сборки mc-proxy"
        exit 1
    }
}

docker compose @Args
