# shot.ps1 <out.png> — screenshot of the "t05 wallpaper" window area (the app sits on top of it).
# shot.ps1 -Place — move the "Focus Widget" window onto the wallpaper first.
param([string]$Out, [switch]$Place)
Add-Type -AssemblyName System.Drawing
Add-Type -Namespace Ui -Name Win -MemberDefinition @'
[DllImport("user32.dll")] public static extern bool SetProcessDPIAware();
[DllImport("user32.dll", CharSet = CharSet.Unicode)] public static extern IntPtr FindWindow(string cls, string title);
[DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
[DllImport("user32.dll")] public static extern bool SetWindowPos(IntPtr h, IntPtr after, int x, int y, int cx, int cy, uint flags);
public struct RECT { public int L, T, R, B; }
'@
[void][Ui.Win]::SetProcessDPIAware()
$wall = [Ui.Win]::FindWindow([NullString]::Value, 't05 wallpaper')
$r = New-Object Ui.Win+RECT
[void][Ui.Win]::GetWindowRect($wall, [ref]$r)
if ($Place) {
    $app = [Ui.Win]::FindWindow([NullString]::Value, 'Focus Widget')
    $topmost = [IntPtr]::new(-1)
    # Lift the wallpaper above the dev console, then the app above the wallpaper.
    # SWP_NOMOVE | SWP_NOSIZE | SWP_NOACTIVATE
    [void][Ui.Win]::SetWindowPos($wall, $topmost, 0, 0, 0, 0, 0x0002 -bor 0x0001 -bor 0x0010)
    # SWP_NOSIZE | SWP_NOACTIVATE
    [void][Ui.Win]::SetWindowPos($app, $topmost, $r.L + 240, $r.T + 140, 0, 0, 0x0001 -bor 0x0010)
    return
}
$w = $r.R - $r.L; $h = $r.B - $r.T
$bmp = New-Object System.Drawing.Bitmap $w, $h
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.CopyFromScreen($r.L, $r.T, 0, 0, $bmp.Size)
$bmp.Save($Out, [System.Drawing.Imaging.ImageFormat]::Png)
$g.Dispose(); $bmp.Dispose()
