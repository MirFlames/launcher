# Генерирует BMP для мастера Inno Setup (стилизация «в духе Minecraft»: тёмный фон, зелёная «трава», коричневый «грунт»).
$ErrorActionPreference = "Stop"
$assets = Join-Path $PSScriptRoot "assets"
New-Item -ItemType Directory -Force -Path $assets | Out-Null
Add-Type -AssemblyName System.Drawing

function Write-WizardBitmap {
    param(
        [int]$Width,
        [int]$Height,
        [string]$Path
    )
    $bmp = New-Object System.Drawing.Bitmap $Width, $Height
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::None
    $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::NearestNeighbor

    $sky = [System.Drawing.Color]::FromArgb(45, 85, 120)
    $grassTop = [System.Drawing.Color]::FromArgb(92, 181, 76)
    $grassSide = [System.Drawing.Color]::FromArgb(72, 142, 58)
    $dirt = [System.Drawing.Color]::FromArgb(110, 74, 47)

    $g.Clear($sky)
    $grassH = [Math]::Max(1, [int]($Height * 0.28))
    $dirtH = [Math]::Max(1, [int]($Height * 0.22))
    $yGrass = $Height - $grassH - $dirtH

    $g.FillRectangle((New-Object System.Drawing.SolidBrush $grassTop), 0, $yGrass, $Width, $grassH)
    $g.FillRectangle((New-Object System.Drawing.SolidBrush $dirt), 0, $yGrass + $grassH, $Width, $dirtH)
    $block = [Math]::Min($Width, $Height) / 3
    $bx = $Width - $block - 8
    $by = $Height - $block - 8
    if ($bx -gt 0 -and $by -gt 0) {
        $g.FillRectangle((New-Object System.Drawing.SolidBrush $grassSide), $bx, $by, $block, $block)
        $g.FillRectangle((New-Object System.Drawing.SolidBrush $grassTop), $bx, $by, $block, [int]($block * 0.25))
    }

    $g.Dispose()
    $bmp.Save($Path, [System.Drawing.Imaging.ImageFormat]::Bmp)
    $bmp.Dispose()
}

Write-WizardBitmap -Width 164 -Height 314 -Path (Join-Path $assets "wizard-large.bmp")
Write-WizardBitmap -Width 55 -Height 55 -Path (Join-Path $assets "wizard-small.bmp")
Write-Host "Wizard BMPs written to $assets"
