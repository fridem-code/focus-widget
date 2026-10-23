# Scripted UI check of a dev build: puts the app over the T0.5 "wallpaper"
# window (coloured blobs + white text, so the glass is visible), clicks through
# the timer states over the WebView2 DevTools protocol, prints the DOM state
# of each step and saves a screenshot per step to scripts/ui-check/out.
#
# Prereqs: wails3 build DEV=true, and the spike built once:
#   cd spike/t05-acrylic; go build -ldflags "-H windowsgui" -o bin/t05acrylic.exe .
# Note: it drives the real app, so the settings file is the real one too; the
# check only uses timer commands and ends with Reset.
param([int]$Port = 9333)
$ErrorActionPreference = 'Stop'
$root = Resolve-Path "$PSScriptRoot\..\.."
$out = New-Item -ItemType Directory -Force "$PSScriptRoot\out"

$wall = Start-Process "$root\spike\t05-acrylic\bin\t05acrylic.exe" -ArgumentList '-wallonly' -PassThru
Start-Sleep -Seconds 2
$env:FOCUSWIDGET_DEVTOOLS_PORT = "$Port"
$app = Start-Process "$root\bin\focuswidget.exe" -PassThru
Remove-Item Env:\FOCUSWIDGET_DEVTOOLS_PORT
try {
    Start-Sleep -Seconds 3
    & "$PSScriptRoot\shot.ps1" -Place
    Start-Sleep -Milliseconds 500
    node "$PSScriptRoot\ui-check.mjs" $Port $out.FullName
} finally {
    Stop-Process -Id $app.Id, $wall.Id -Confirm:$false -ErrorAction SilentlyContinue
}
