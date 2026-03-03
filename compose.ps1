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

if ($buildBackend) {
    # Сборка backend на хосте перед docker build
    Push-Location $PSScriptRoot\backend
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    go build -o launcher-backend .
    $goExit = $LASTEXITCODE
    Pop-Location
    if ($goExit -ne 0) {
        Write-Error "Ошибка сборки Go"
        exit 1
    }
}

docker compose @Args
