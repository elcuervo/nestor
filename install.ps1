$ErrorActionPreference = "Stop"

if (-not [Environment]::Is64BitOperatingSystem) {
  Write-Error "unsupported architecture"; exit 1
}

Invoke-WebRequest "https://github.com/elcuervo/nestor/releases/latest/download/nestor-windows-amd64.exe" -OutFile "nestor.exe"
& ".\nestor.exe"
