param(
    [Parameter(Mandatory = $true)]
    [string]$Version
)

$ErrorActionPreference = "Stop"

$binaryName = "myclaw"
$installerPath = Join-Path "dist" "$binaryName-windows-$Version.exe"

if (Test-Path $installerPath) {
    Remove-Item $installerPath -Force
}

# Build with Wails
Set-Location cmd/myclaw-desktop
wails build -platform windows/amd64 -o myclaw.exe -nsis -webview2 download -m -s
Set-Location ../..

# Move installer to dist
$builtInstaller = Get-ChildItem -Path "cmd/myclaw-desktop/build/bin" -Filter "*-installer.exe" | Select-Object -First 1
if ($null -eq $builtInstaller) {
    throw "Installer not found in cmd/myclaw-desktop/build/bin"
}

New-Item -ItemType Directory -Path "dist" -Force | Out-Null
Copy-Item $builtInstaller.FullName -Destination $installerPath

Write-Host "Created $installerPath"
