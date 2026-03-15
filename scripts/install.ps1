# install.ps1 - installs the latest ssx release on Windows.
#
# Usage (run in PowerShell as normal user):
#   irm https://raw.githubusercontent.com/highfredo/ssx/main/scripts/install.ps1 | iex
#
# Override the install directory:
#   $env:INSTALL_DIR = "C:\Tools"; irm ... | iex

[CmdletBinding()]
param(
  [string]$InstallDir = $(
    if ($env:INSTALL_DIR) { $env:INSTALL_DIR }
    else { "$env:USERPROFILE\.local\bin" }
  )
)

$ErrorActionPreference = "Stop"

$Repo   = "highfredo/ssx"
$Binary = "ssx.exe"

# -- helpers ------------------------------------------------------------------

function Info  { Write-Host ">>  $args" -ForegroundColor Cyan }
function Ok    { Write-Host "OK  $args" -ForegroundColor Green }
function Fail  { Write-Host "ERR $args" -ForegroundColor Red; exit 1 }

# -- detect arch --------------------------------------------------------------

# Windows arm64 is not published (ignored in goreleaser config), always amd64.
$Arch = "amd64"

# -- fetch latest version -----------------------------------------------------

Info "Fetching latest release..."
try {
  $Release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
} catch {
  Fail "Could not reach GitHub API: $_"
}

$Tag     = $Release.tag_name       # e.g. "v1.2.3"
$Version = $Tag -replace '^v', '' # e.g. "1.2.3"
Info "Latest version: $Tag"

# -- build URLs ---------------------------------------------------------------

$ArchiveName  = "ssx_${Version}_windows_${Arch}.zip"
$BaseUrl      = "https://github.com/$Repo/releases/download/$Tag"
$ArchiveUrl   = "$BaseUrl/$ArchiveName"
$ChecksumsUrl = "$BaseUrl/checksums.txt"

# -- download -----------------------------------------------------------------

$TmpDir = Join-Path $env:TEMP ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $TmpDir | Out-Null

try {
  $ArchivePath   = Join-Path $TmpDir $ArchiveName
  $ChecksumsPath = Join-Path $TmpDir "checksums.txt"

  Info "Downloading $ArchiveName..."
  Invoke-WebRequest $ArchiveUrl   -OutFile $ArchivePath   -UseBasicParsing
  Invoke-WebRequest $ChecksumsUrl -OutFile $ChecksumsPath -UseBasicParsing

  # -- verify checksum --------------------------------------------------------

  Info "Verifying checksum..."
  $ChecksumLine = Get-Content $ChecksumsPath | Where-Object { $_ -match " $([regex]::Escape($ArchiveName))$" }
  if (-not $ChecksumLine) { Fail "Checksum not found for $ArchiveName in checksums.txt" }

  $Expected = ($ChecksumLine -split '\s+')[0].ToLower()
  $Actual   = (Get-FileHash $ArchivePath -Algorithm SHA256).Hash.ToLower()

  if ($Actual -ne $Expected) {
    Fail "Checksum mismatch!`n  expected: $Expected`n  got:      $Actual"
  }
  Ok "Checksum OK"

  # -- extract & install ------------------------------------------------------

  Expand-Archive -Path $ArchivePath -DestinationPath $TmpDir -Force

  if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
  }

  Copy-Item (Join-Path $TmpDir $Binary) (Join-Path $InstallDir $Binary) -Force
  Ok "ssx $Tag installed -> $InstallDir\$Binary"

  # -- PATH hint --------------------------------------------------------------

  $UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
  if ($UserPath -notlike "*$InstallDir*") {
    Write-Host ""
    Write-Host "  Add $InstallDir to your PATH by running:" -ForegroundColor Yellow
    Write-Host "  [Environment]::SetEnvironmentVariable('PATH', `$env:PATH + ';$InstallDir', 'User')" -ForegroundColor Yellow
  }

} finally {
  Remove-Item $TmpDir -Recurse -Force -ErrorAction SilentlyContinue
}
