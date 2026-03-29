# MailBus Automatic Installation Script for Windows
# This script detects your platform and downloads the appropriate MailBus binary

param(
    [string]$Version = "latest",
    [string]$InstallDir = "$env:USERPROFILE\bin"
)

Write-Host "MailBus Installation Script" -ForegroundColor Cyan
Write-Host "==========================" -ForegroundColor Cyan
Write-Host "Version: $Version"
Write-Host "Install Directory: $InstallDir"
Write-Host ""

# Detect architecture
$Arch = $env:PROCESSOR_ARCHITECTURE
switch ($Arch) {
    "AMD64" { $Arch = "amd64" }
    "x86" { $Arch = "386" }
    "ARM64" { $Arch = "arm64" }
    default {
        Write-Host "Error: Unsupported architecture: $Arch" -ForegroundColor Red
        exit 1
    }
}
Write-Host "Detected Architecture: $Arch"

# Binary name
$BinaryName = "mailbus-windows-$Arch.exe"
Write-Host "Binary: $BinaryName"

# Download URL
$DownloadUrl = "https://github.com/mailbus/mailbus/releases/$Version/download/$BinaryName"
Write-Host "Download URL: $DownloadUrl"
Write-Host ""

# Create temp directory
$TempDir = Join-Path $env:TEMP "mailbus-install"
if (Test-Path $TempDir) {
    Remove-Item -Recurse -Force $TempDir
}
New-Item -ItemType Directory -Path $TempDir | Out-Null

# Download
Write-Host "Downloading MailBus..." -ForegroundColor Yellow
try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile "$TempDir\$BinaryName" -UseBasicParsing
} catch {
    Write-Host "Error: Download failed" -ForegroundColor Red
    Write-Host $_.Exception.Message
    exit 1
}

# Download checksum if available
$ChecksumUrl = "https://github.com/mailbus/mailbus/releases/$Version/download/$BinaryName.sha256"
try {
    Write-Host "Downloading checksum..." -ForegroundColor Yellow
    Invoke-WebRequest -Uri $ChecksumUrl -OutFile "$TempDir\$BinaryName.sha256" -UseBasicParsing

    # Verify checksum
    Write-Host "Verifying checksum..." -ForegroundColor Yellow
    $ExpectedHash = Get-Content "$TempDir\$BinaryName.sha256"
    $ActualHash = (Get-FileHash "$TempDir\$BinaryName" -Algorithm SHA256).Hash.ToLower()

    if ($ExpectedHash -eq $ActualHash) {
        Write-Host "Checksum verified successfully" -ForegroundColor Green
    } else {
        Write-Host "Warning: Checksum verification failed" -ForegroundColor Yellow
        Write-Host "Expected: $ExpectedHash"
        Write-Host "Actual: $ActualHash"
        $Response = Read-Host "Continue anyway? (y/N)"
        if ($Response -ne 'y') {
            exit 1
        }
    }
} catch {
    Write-Host "Checksum not available or verification failed" -ForegroundColor Yellow
}

# Create install directory
if (!(Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir | Out-Null
}

# Install
Write-Host ""
Write-Host "Installing MailBus to $InstallDir..." -ForegroundColor Yellow
Copy-Item "$TempDir\$BinaryName" -Destination "$InstallDir\mailbus.exe"

# Add to PATH if not already there
$PathEnv = [Environment]::GetEnvironmentVariable("Path", "User")
if ($PathEnv -notlike "*$InstallDir*") {
    Write-Host ""
    Write-Host "Adding $InstallDir to user PATH..." -ForegroundColor Yellow
    [Environment]::SetEnvironmentVariable("Path", "$PathEnv;$InstallDir", "User")
    Write-Host "PATH updated. You may need to restart your terminal." -ForegroundColor Green
}

# Cleanup
Remove-Item -Recurse -Force $TempDir

Write-Host ""
Write-Host "MailBus installed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "To verify installation:"
Write-Host "  mailbus.exe version"
Write-Host ""
Write-Host "To get started:"
Write-Host "  mailbus.exe config init"
Write-Host ""
Write-Host "For more information:"
Write-Host "  https://github.com/mailbus/mailbus"
Write-Host "  https://github.com/mailbus/mailbus/blob/main/AGENT_INSTALLATION.md"
