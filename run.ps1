$ErrorActionPreference = "Stop"

if (-not [Environment]::Is64BitOperatingSystem) {
  Write-Error "unsupported architecture"; exit 1
}

$TMP = [System.IO.Path]::GetTempFileName() + ".exe"
try {
  Invoke-WebRequest "https://github.com/elcuervo/nestor/releases/latest/download/nestor-windows-amd64.exe" -OutFile $TMP
  & $TMP @args
} finally {
  Remove-Item -Force -ErrorAction SilentlyContinue $TMP
}
