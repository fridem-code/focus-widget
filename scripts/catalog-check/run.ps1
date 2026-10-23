# Checks that every catalog source plays in the app's own YouTube player
# (origin http://wails.localhost, exactly as users get it).
#   powershell -File scripts/catalog-check/run.ps1                     # the app's catalog
#   powershell -File scripts/catalog-check/run.ps1 -Candidates c.json  # sources not in the catalog yet
# Needs a dev build (wails3 build DEV=true). Sources play muted.
param([string]$Candidates, [int]$Port = 9333)
$ErrorActionPreference = 'Stop'
$root = Resolve-Path "$PSScriptRoot\..\.."
$nodeArgs = @("$PSScriptRoot\check.mjs", $Port)
if ($Candidates) { $nodeArgs += (Resolve-Path $Candidates).Path }

# The first launch of a freshly built exe sometimes drops the DevTools
# connection early (cause not pinned down); a second launch is stable.
foreach ($attempt in 1..2) {
    $env:FOCUSWIDGET_DEVTOOLS_PORT = "$Port"
    $app = Start-Process "$root\bin\focuswidget.exe" -PassThru
    Remove-Item Env:\FOCUSWIDGET_DEVTOOLS_PORT
    try {
        node @nodeArgs
        $ok = $LASTEXITCODE -eq 0
    } finally {
        Stop-Process -Id $app.Id -Confirm:$false -ErrorAction SilentlyContinue
    }
    if ($ok) { break }
    Write-Host "attempt $attempt failed, retrying"
    Start-Sleep -Seconds 2
}
