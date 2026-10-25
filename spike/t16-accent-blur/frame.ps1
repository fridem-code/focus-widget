# frame.ps1 -L 0 -R 0 -T 0 -B 1 [-Corner 2] — sets DWM frame margins and corner preference on the widget.
param([int]$L = 0, [int]$R = 0, [int]$T = 0, [int]$B = 0, [int]$Corner = -1)
Add-Type -Namespace Fr -Name W -MemberDefinition @'
[DllImport("user32.dll", CharSet = CharSet.Unicode)] public static extern IntPtr FindWindow(string c, string t);
[DllImport("dwmapi.dll")] public static extern int DwmExtendFrameIntoClientArea(IntPtr h, ref MARGINS m);
[DllImport("dwmapi.dll")] public static extern int DwmSetWindowAttribute(IntPtr h, int a, ref int v, int s);
public struct MARGINS { public int L, R, T, B; }
'@
$h = [Fr.W]::FindWindow([NullString]::Value, 'Focus Widget')
$m = New-Object Fr.W+MARGINS; $m.L = $L; $m.R = $R; $m.T = $T; $m.B = $B
"margins ($L,$R,$T,$B) -> $([Fr.W]::DwmExtendFrameIntoClientArea($h, [ref]$m))"
if ($Corner -ge 0) { "corner $Corner -> $([Fr.W]::DwmSetWindowAttribute($h, 33, [ref]$Corner, 4))" }
