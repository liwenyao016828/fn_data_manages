<#
.SYNOPSIS
Build Feiniu FPK native app package

.DESCRIPTION
One-click packaging tool for Feiniu fnOS

.EXAMPLE
.\build-fpk.ps1
#>

$ErrorActionPreference = "Stop"

# Path config
$ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$FrontendDir = Join-Path $ProjectRoot "frontend"
$ServerDir = Join-Path $ProjectRoot "server"
$FpkDir = Join-Path $ProjectRoot "fpk"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Feiniu FPK Packager" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 1. Check required files
Write-Host "[1/6] Checking project structure..." -ForegroundColor Yellow

$RequiredFiles = @(
    (Join-Path $FrontendDir "package.json"),
    (Join-Path $ServerDir "main.go"),
    (Join-Path $FpkDir "manifest"),
    (Join-Path $FpkDir "fnpack.exe")
)

foreach ($file in $RequiredFiles) {
    if (-not (Test-Path $file)) {
        Write-Host "ERROR: Required file not found $file" -ForegroundColor Red
        exit 1
    }
}
Write-Host "  OK" -ForegroundColor Green

# 2. Build frontend
Write-Host ""
Write-Host "[2/6] Building frontend..." -ForegroundColor Yellow

Push-Location $FrontendDir
try {
    Write-Host "  Running npm run build..."
    npm run build
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: Frontend build failed" -ForegroundColor Red
        exit 1
    }
} finally {
    Pop-Location
}
Write-Host "  OK" -ForegroundColor Green

# 3. Copy frontend to FPK dir
Write-Host ""
Write-Host "[3/6] Copying frontend files..." -ForegroundColor Yellow

$FpkUiDist = Join-Path $FpkDir "app\ui\dist"
if (Test-Path $FpkUiDist) {
    Remove-Item $FpkUiDist -Recurse -Force
}
New-Item -ItemType Directory -Path $FpkUiDist -Force | Out-Null

$FrontendDist = Join-Path $FrontendDir "dist"
Copy-Item (Join-Path $FrontendDist "*") $FpkUiDist -Recurse -Force
Write-Host "  OK" -ForegroundColor Green

# 3.5 Clean up stray Windows binary (left from local dev, would bloat FPK)
$ServerExe = Join-Path $FpkDir "app\server.exe"
if (Test-Path $ServerExe) {
    Remove-Item $ServerExe -Force
    Write-Host "  Removed server.exe (Windows dev binary)" -ForegroundColor DarkGray
}

# 4. Build backend (Linux amd64)
Write-Host ""
Write-Host "[4/6] Building backend (Linux amd64)..." -ForegroundColor Yellow

Push-Location $ServerDir
try {
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    $OutputPath = Join-Path $FpkDir "app\server"
    
    Write-Host "  Running go build..."
    go build -o $OutputPath .
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: Backend build failed" -ForegroundColor Red
        exit 1
    }
    
    Write-Host "  OK" -ForegroundColor Green
} finally {
    Pop-Location
}

# 5. Copy icons
Write-Host ""
Write-Host "[5/6] Copying icons..." -ForegroundColor Yellow

$ImagesDir = Join-Path $FpkDir "app\ui\images"
if (-not (Test-Path $ImagesDir)) {
    New-Item -ItemType Directory -Path $ImagesDir -Force | Out-Null
}

Copy-Item (Join-Path $FpkDir "ICON.PNG") (Join-Path $ImagesDir "icon_64.png") -Force
Copy-Item (Join-Path $FpkDir "ICON_256.PNG") (Join-Path $ImagesDir "icon_256.png") -Force
Write-Host "  OK" -ForegroundColor Green

# 5.5 Normalize cmd/* to LF (CRITICAL: bash on Linux can't run CRLF scripts)
Write-Host ""
Write-Host "[5.5/6] Converting cmd/* to LF..." -ForegroundColor Yellow
Get-ChildItem (Join-Path $FpkDir "cmd") -File | ForEach-Object {
    $raw = [System.IO.File]::ReadAllText($_.FullName)
    if ($raw -match "`r`n") {
        $lf = $raw -replace "`r`n", "`n"
        [System.IO.File]::WriteAllText($_.FullName, $lf)
    }
}
Write-Host "  OK" -ForegroundColor Green

# 6. Build FPK
Write-Host ""
Write-Host "[6/6] Building FPK package..." -ForegroundColor Yellow

Push-Location $FpkDir
try {
    $FpkOutput = Join-Path $FpkDir "niudb.fpk"
    if (Test-Path $FpkOutput) {
        Remove-Item $FpkOutput -Force
    }
    
    Write-Host "  Running fnpack build..."
    .\fnpack.exe build
    
    if (-not (Test-Path $FpkOutput)) {
        Write-Host "ERROR: FPK build failed" -ForegroundColor Red
        exit 1
    }
    
    $Size = (Get-Item $FpkOutput).Length / 1MB
    Write-Host "  OK ($([math]::Round($Size, 2)) MB)" -ForegroundColor Green
} finally {
    Pop-Location
}

# Done
Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "  FPK Build Successful!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Output: $FpkOutput" -ForegroundColor Cyan
Write-Host ""
