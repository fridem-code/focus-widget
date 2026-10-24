# Measures the SPEC §4 numbers on a release build: exe size, cold start to a
# visible window, idle CPU and memory of the app with its WebView2 processes.
#
# Prereqs: wails3 build (or pass -Exe for an installed copy). Quit a running
# widget first. The app starts with the real settings; nothing is changed.
param(
    [string]$Exe = "$PSScriptRoot\..\..\bin\focuswidget.exe",
    [int]$Runs = 3,
    [int]$IdleSeconds = 30
)
$ErrorActionPreference = 'Stop'
$Exe = (Resolve-Path $Exe).Path
$name = [IO.Path]::GetFileNameWithoutExtension($Exe)
if (Get-Process $name -ErrorAction SilentlyContinue) { throw "$name is running; quit it first" }

Add-Type -Namespace Nfr -Name Win -MemberDefinition @'
[DllImport("user32.dll", CharSet = CharSet.Unicode)] public static extern IntPtr FindWindow(string cls, string title);
[DllImport("user32.dll")] public static extern bool IsWindowVisible(IntPtr h);
'@

# The app plus every process below it (the WebView2 browser, renderer, GPU... processes).
function Get-Tree([int]$RootId) {
    $all = Get-CimInstance Win32_Process | Select-Object ProcessId, ParentProcessId
    $ids = @($RootId)
    do {
        $more = $all | Where-Object { $ids -contains $_.ParentProcessId -and $ids -notcontains $_.ProcessId } |
            ForEach-Object { $_.ProcessId }
        $ids += $more
    } while ($more)
    Get-Process -Id $ids -ErrorAction SilentlyContinue
}

function Stop-App([int]$Id) {
    Stop-Process -Id $Id -Force -ErrorAction SilentlyContinue
    for ($i = 0; $i -lt 50 -and (Get-Process $name -ErrorAction SilentlyContinue); $i++) { Start-Sleep -Milliseconds 100 }
}

"exe size:        {0:N1} MB (limit 20)" -f ((Get-Item $Exe).Length / 1MB)

$times = @()
for ($r = 1; $r -le $Runs; $r++) {
    $sw = [Diagnostics.Stopwatch]::StartNew()
    $p = Start-Process $Exe -PassThru
    $shown = $false
    while ($sw.ElapsedMilliseconds -lt 10000) {
        $h = [Nfr.Win]::FindWindow([NullString]::Value, 'Focus Widget')
        if ($h -ne [IntPtr]::Zero -and [Nfr.Win]::IsWindowVisible($h)) { $shown = $true; break }
        Start-Sleep -Milliseconds 10
    }
    $sw.Stop()
    if (-not $shown) { Stop-App $p.Id; throw 'the window did not show within 10 s' }
    $times += $sw.ElapsedMilliseconds
    if ($r -lt $Runs) { Stop-App $p.Id; Start-Sleep -Seconds 1 }
}
"start to window: {0} ms (median of {1}: {2}; limit 2000)" -f (($times | Sort-Object)[[int][math]::Floor($Runs / 2)]), $Runs, ($times -join ', ')

# Idle: the last run keeps going; let it settle, then sample CPU time.
Start-Sleep -Seconds 5
$before = Get-Tree $p.Id
$cpu0 = ($before | ForEach-Object { $_.TotalProcessorTime.TotalMilliseconds } | Measure-Object -Sum).Sum
Start-Sleep -Seconds $IdleSeconds
$after = Get-Tree $p.Id
$cpu1 = ($after | ForEach-Object { $_.TotalProcessorTime.TotalMilliseconds } | Measure-Object -Sum).Sum
$cores = [Environment]::ProcessorCount
"idle CPU:        {0:N2} % of the machine over {1} s ({2} processes, {3} cores; limit 1)" -f `
    (100 * ($cpu1 - $cpu0) / ($IdleSeconds * 1000 * $cores)), $IdleSeconds, $after.Count, $cores
"idle memory:     {0:N0} MB private, {1:N0} MB working set" -f `
    (($after | Measure-Object -Property PrivateMemorySize64 -Sum).Sum / 1MB),
    (($after | Measure-Object -Property WorkingSet64 -Sum).Sum / 1MB)

Stop-App $p.Id
