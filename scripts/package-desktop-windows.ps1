param(
    [Parameter(Mandatory = $true)]
    [string]$Version
)

$ErrorActionPreference = "Stop"

$binaryName = "myclaw"
$versionedBaseName = "$binaryName-$Version"
$versionedExeName = "$versionedBaseName.exe"
$versionedInstallerName = "$versionedBaseName-amd64-installer.exe"
$installerPath = Join-Path "dist" $versionedInstallerName
$ldflags = "-X main.appVersion=$Version"

if (Test-Path $installerPath) {
    Remove-Item $installerPath -Force
}

# Build with Wails
Set-Location cmd/myclaw-desktop
wails build -platform windows/amd64 -o $versionedExeName -nsis -webview2 download -ldflags $ldflags -m -s
Set-Location ../..

# Normalize installer filename so build/bin and dist both carry the version.
$buildBinDir = Join-Path "cmd/myclaw-desktop/build" "bin"
$versionedBuiltInstallerPath = Join-Path $buildBinDir $versionedInstallerName
$builtInstaller = Get-ChildItem -Path $buildBinDir -Filter "$versionedBaseName-*-installer.exe" | Select-Object -First 1
if ($null -eq $builtInstaller) {
    $builtInstaller = Get-ChildItem -Path $buildBinDir -Filter "*-installer.exe" | Sort-Object LastWriteTimeUtc -Descending | Select-Object -First 1
}
if ($null -eq $builtInstaller) {
    throw "Installer not found in cmd/myclaw-desktop/build/bin"
}

if ($builtInstaller.Name -ne $versionedInstallerName) {
    if (Test-Path $versionedBuiltInstallerPath) {
        Remove-Item $versionedBuiltInstallerPath -Force
    }
    Move-Item $builtInstaller.FullName $versionedBuiltInstallerPath
}

New-Item -ItemType Directory -Path "dist" -Force | Out-Null
Copy-Item $versionedBuiltInstallerPath -Destination $installerPath

Write-Host "Created $installerPath"
