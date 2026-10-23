# Runs the spike in one variant, reads the widget page state over CDP and saves
# screenshots of the widget area while it is focused and unfocused.
#   powershell -File capture.ps1 -Variant acrylic [-Compact] -OutDir <dir>
param(
    [string]$Variant = 'acrylic',
    [switch]$Compact,
    [switch]$Resize,   # start expanded, shrink to compact at 8 s
    [string]$OutDir = "$PSScriptRoot\shots"
)
$ErrorActionPreference = 'Stop'
New-Item -ItemType Directory -Force $OutDir | Out-Null

Add-Type -AssemblyName System.Drawing
Add-Type -Namespace Spike -Name Win -MemberDefinition @'
[DllImport("user32.dll")] public static extern bool SetProcessDPIAware();
[DllImport("user32.dll", CharSet = CharSet.Unicode)] public static extern IntPtr FindWindow(string cls, string title);
[DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
[DllImport("user32.dll")] public static extern IntPtr GetForegroundWindow();
public struct RECT { public int L, T, R, B; }
'@
[void][Spike.Win]::SetProcessDPIAware()

function Save-Shot([string]$name) {
    $wall = [Spike.Win]::FindWindow([NullString]::Value, 't05 wallpaper')
    $r = New-Object Spike.Win+RECT
    [void][Spike.Win]::GetWindowRect($wall, [ref]$r)
    $w = $r.R - $r.L; $h = $r.B - $r.T
    $bmp = New-Object System.Drawing.Bitmap $w, $h
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.CopyFromScreen($r.L, $r.T, 0, 0, $bmp.Size)
    $path = Join-Path $OutDir $name
    $bmp.Save($path, [System.Drawing.Imaging.ImageFormat]::Png)
    $g.Dispose(); $bmp.Dispose()
    $widget = [Spike.Win]::FindWindow([NullString]::Value, 't05 widget')
    $wr = New-Object Spike.Win+RECT
    [void][Spike.Win]::GetWindowRect($widget, [ref]$wr)
    $fg = if ([Spike.Win]::GetForegroundWindow() -eq $widget) { 'widget' } else { 'other' }
    "shot $name  widget=$($wr.R-$wr.L)x$($wr.B-$wr.T) foreground=$fg"
}

$exe = "$PSScriptRoot\bin\t05acrylic.exe"
$args = @('-variant', $Variant, '-blurafter', '16')
if ($Compact) { $args += '-compact' }
if ($Resize) { $args += @('-resizeafter', '8') }
$p = Start-Process $exe -ArgumentList $args -PassThru
try {
    Start-Sleep -Seconds 14
    $suffix = if ($Compact) { "$Variant-compact" } elseif ($Resize) { "$Variant-resized" } else { $Variant }
    Save-Shot "$suffix-focused.png"
    Start-Sleep -Seconds 3.5   # the app focuses the wallpaper at 16 s
    Save-Shot "$suffix-unfocused.png"
    if (-not $Compact) {
        node "$PSScriptRoot\state.mjs" 9334
    }
} finally {
    Stop-Process -Id $p.Id -Confirm:$false -ErrorAction SilentlyContinue
}
