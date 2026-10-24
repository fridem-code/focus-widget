# App icon

`appicon.svg` is the source of `build/appicon.png` (1024×1024, transparent
corners), from which the build generates `windows/icon.ico` — the exe, window,
tray and toast icon. To change it, edit the SVG, render it and regenerate:

```powershell
$edge = 'C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe'
& $edge --headless=new --disable-gpu --hide-scrollbars --force-device-scale-factor=1 `
  --window-size=1024,1024 --default-background-color=00000000 `
  --user-data-dir="$env:TEMP\fw-icon-edge" --screenshot="$PWD\build\appicon.png" `
  "file:///$($PWD -replace '\','/')/build/icon-src/appicon.svg"
wails3 task common:generate:icons
```
