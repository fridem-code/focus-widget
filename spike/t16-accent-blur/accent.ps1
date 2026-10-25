# accent.ps1 -Mode system|acrylic|blur|none [-Color 0x00000000]
# Switches the "Focus Widget" window's backdrop: the DWM system backdrop (what the app ships),
# or the undocumented accent policy (SetWindowCompositionAttribute) with the system backdrop off.
# -Color is the accent gradient colour as 0xAABBGGRR (acrylic only).
param([string]$Mode = 'acrylic', [uint32]$Color = 0x01000000)
Add-Type -Namespace Ac -Name W -MemberDefinition @'
[DllImport("user32.dll", CharSet = CharSet.Unicode)] public static extern IntPtr FindWindow(string c, string t);
[DllImport("dwmapi.dll")] public static extern int DwmSetWindowAttribute(IntPtr h, int a, ref int v, int s);
[DllImport("user32.dll")] public static extern int SetWindowCompositionAttribute(IntPtr h, ref Data d);
[StructLayout(LayoutKind.Sequential)] public struct Accent { public int State; public int Flags; public uint Color; public int AnimationId; }
[StructLayout(LayoutKind.Sequential)] public struct Data { public int Attrib; public IntPtr Ptr; public int Size; }
public static int Apply(IntPtr h, int state, uint color) {
    var a = new Accent { State = state, Flags = 2, Color = color };
    int size = Marshal.SizeOf(a); IntPtr p = Marshal.AllocHGlobal(size);
    try { Marshal.StructureToPtr(a, p, false); var d = new Data { Attrib = 19, Ptr = p, Size = size }; return SetWindowCompositionAttribute(h, ref d); }
    finally { Marshal.FreeHGlobal(p); }
}
'@
$h = [Ac.W]::FindWindow([NullString]::Value, 'Focus Widget')
if ($h -eq [IntPtr]::Zero) { throw 'widget window not found' }
$backdrop = if ($Mode -eq 'system') { 3 } else { 1 } # DWMSBT_TRANSIENTWINDOW (Acrylic) / DWMSBT_NONE
[void][Ac.W]::DwmSetWindowAttribute($h, 38, [ref]$backdrop, 4)
$state = switch ($Mode) { 'acrylic' { 4 } 'blur' { 3 } default { 0 } } # ACCENT_ENABLE_ACRYLICBLURBEHIND / _BLURBEHIND / _DISABLED
"backdrop=$backdrop accent=$state color=0x{0:X8} -> {1}" -f $Color, ([Ac.W]::Apply($h, $state, $Color))
