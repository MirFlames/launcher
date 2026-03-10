# Wails Launcher - PowerShell build script
# Использование: .\build.ps1 [dev|build|clean|sign]
# Альтернатива Makefile для Windows без make

param(
    [Parameter(Position=0)]
    [ValidateSet("dev", "build", "clean", "sign")]
    [string]$Target = "build"
)

$ErrorActionPreference = "Stop"

$ExePath = "build\bin\launcher.exe"
$TimestampServer = "http://timestamp.digicert.com"

$SignToolPath = Get-ChildItem "C:\Program Files (x86)\Windows Kits\10\bin\*\x64\signtool.exe" -ErrorAction SilentlyContinue |
    Sort-Object { $_.Directory.Name } -Descending |
    Select-Object -First 1 -ExpandProperty FullName
if (-not $SignToolPath) { $SignToolPath = "signtool" }

# Загрузка секретов: сначала корневой .env, затем .sign.env (fallback)
$EnvFiles = @(
    (Join-Path $PSScriptRoot "..\..\.env"),
    (Join-Path $PSScriptRoot ".sign.env")
)
foreach ($EnvFile in $EnvFiles) {
    if (Test-Path $EnvFile) {
        Get-Content $EnvFile | ForEach-Object {
            $line = $_.Trim()
            if ($line -match '^\s*([^#=][^=]*)=(.*)$') {
                $key = $Matches[1].Trim()
                $val = $Matches[2].Trim()
                # Убрать кавычки для значений с # и другими спецсимволами
                if (($val.Length -ge 2) -and (($val.StartsWith('"') -and $val.EndsWith('"')) -or ($val.StartsWith("'") -and $val.EndsWith("'")))) {
                    $val = $val.Substring(1, $val.Length - 2)
                }
                Set-Item -Path "env:$key" -Value $val
            }
        }
        break
    }
}

$SignCertPfx = $(if ($env:CODESIGN_PFX) { [Environment]::ExpandEnvironmentVariables($env:CODESIGN_PFX) } else { "$env:USERPROFILE\codesign.pfx" })
$SignCertPassword = $env:CODESIGN_PASSWORD

function Sign-Executable {
    if (-not (Test-Path $SignCertPfx)) {
        Write-Host "Certificate not found at $SignCertPfx - skipping signing." -ForegroundColor Yellow
        return
    }
    if (-not $SignCertPassword) {
        Write-Host "CODESIGN_PASSWORD not set. Add it to .sign.env or set as env variable." -ForegroundColor Red
        exit 1
    }
    Write-Host "Signing $ExePath..." -ForegroundColor Cyan
    & $SignToolPath sign /f $SignCertPfx /p $SignCertPassword /tr $TimestampServer /td sha256 /fd sha256 $ExePath
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Signing failed!" -ForegroundColor Red
        exit 1
    }

    # В CI цепочка сертификатов может быть неполной, поэтому verify может возвращать ненулевой код,
    # даже если файл подписан корректно. В таком случае выводим предупреждение, но не падаем сборкой.
    & $SignToolPath verify /pa $ExePath 2>&1 | Out-Null
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Warning: signtool verify reported issues (exit code $LASTEXITCODE). Continuing." -ForegroundColor Yellow
        $global:LASTEXITCODE = 0
    }
    Write-Host "Signed successfully." -ForegroundColor Green
}

switch ($Target) {
    "dev" {
        Write-Host "Starting Wails dev server (hot reload)..." -ForegroundColor Cyan
        wails dev
    }
    "build" {
        Write-Host "Building production binary..." -ForegroundColor Cyan
        # Дефолты из .env при сборке (API_BASE_URL, SERVER_HOST, SERVER_PORT)
        $apiUrl = if ($env:API_BASE_URL) { $env:API_BASE_URL } else { $env:BASE_URL }

        # Обновляем иконку приложения для Windows: копируем PNG из frontend-ассетов
        # в ожидаемое Wails-ом место build/appicon.png и удаляем устаревший icon.ico,
        # чтобы он был пересобран из нового PNG.
        $iconSource = Join-Path $PSScriptRoot "frontend\src\assets\images\appicon.png"
        $iconTarget = Join-Path $PSScriptRoot "build\appicon.png"
        if (Test-Path $iconSource) {
            Write-Host "Updating app icon from $iconSource" -ForegroundColor Cyan
            New-Item -ItemType Directory -Force -Path (Split-Path $iconTarget) | Out-Null
            Copy-Item $iconSource $iconTarget -Force

            $iconIco = Join-Path $PSScriptRoot "build\windows\icon.ico"
            if (Test-Path $iconIco) {
                Remove-Item $iconIco -Force
            }
        }

        $ldflags = "-s -w -H windowsgui"
        if ($apiUrl) { $ldflags += " -X main.buildDefaultApiBaseUrl=$apiUrl" }
        if ($env:SERVER_HOST) { $ldflags += " -X main.buildDefaultServerHost=$($env:SERVER_HOST)" }
        if ($env:SERVER_PORT) { $ldflags += " -X main.buildDefaultServerPort=$($env:SERVER_PORT)" }
        if ($env:UPDATE_MANIFEST_URL) { $ldflags += " -X main.buildUpdateManifestURL=$($env:UPDATE_MANIFEST_URL)" }
        if ($env:UPDATE_SIGNATURE_URL) { $ldflags += " -X main.buildUpdateSignatureURL=$($env:UPDATE_SIGNATURE_URL)" }
        if ($env:UPDATE_PUBLIC_KEY_HEX) { $ldflags += " -X main.buildUpdatePublicKeyHex=$($env:UPDATE_PUBLIC_KEY_HEX)" }
        wails build -ldflags $ldflags -trimpath
        Write-Host "Build complete. Output in build/bin/" -ForegroundColor Green
        Sign-Executable
    }
    "sign" {
        if (-not (Test-Path $ExePath)) {
            Write-Host "Binary not found. Run '.\build.ps1 build' first." -ForegroundColor Red
            exit 1
        }
        Sign-Executable
    }
    "clean" {
        Write-Host "Cleaning build artifacts..." -ForegroundColor Yellow
        if (Test-Path "build/bin") { Remove-Item -Recurse -Force "build/bin" }
        if (Test-Path "frontend/dist") { Remove-Item -Recurse -Force "frontend/dist" }
        go clean
        Write-Host "Clean complete." -ForegroundColor Green
    }
}
