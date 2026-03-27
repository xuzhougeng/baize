Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $PSCommandPath
$repoRoot = (Resolve-Path (Join-Path $scriptDir "..")).Path

Push-Location $repoRoot
try {
    git config core.hooksPath .githooks
    $hookPath = Join-Path $repoRoot ".githooks\commit-msg"
    if (Test-Path $hookPath) {
        Write-Host "Installed git hooks: core.hooksPath=$(git config --get core.hooksPath)"
    }
}
finally {
    Pop-Location
}
