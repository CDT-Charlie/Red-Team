# RMM blue-team lab deployment helper
param(
    [switch]$Persist
)

if (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Warning "[-] Administrator privileges are required for this lab deployment."
    exit
}

$ScriptDir = $PSScriptRoot
if ([string]::IsNullOrEmpty($ScriptDir)) { $ScriptDir = "." }

$LocalAgent = Join-Path $ScriptDir "rmm-agent.exe"
$LabRoot = "C:\ProgramData\RMMBlueTeam"
$LogDir = Join-Path $LabRoot "logs"
$DestPath = Join-Path $LabRoot "rmm-agent.exe"
$StopFile = Join-Path $LabRoot "STOP"
$ServiceName = "RMMBlueTeamMonitor"

New-Item -Path $LabRoot -ItemType Directory -Force | Out-Null
New-Item -Path $LogDir -ItemType Directory -Force | Out-Null

$TranscriptPath = Join-Path $LogDir ("deploy-" + (Get-Date -Format "yyyyMMdd-HHmmss") + ".log")
Start-Transcript -Path $TranscriptPath -Append | Out-Null

Write-Host "[AUDIT] Lab root: $LabRoot"
Write-Host "[AUDIT] Deployment transcript: $TranscriptPath"
Write-Host "[AUDIT] Stop file: $StopFile"

if (-Not (Test-Path $LocalAgent)) {
    Write-Host "[-] rmm-agent.exe was not found next to this script."
    Stop-Transcript | Out-Null
    exit
}

Write-Host "[*] Copying the agent into the lab directory..."
Copy-Item -Path $LocalAgent -Destination $DestPath -Force
Write-Host "[+] Copied agent to $DestPath"

Write-Host "[*] Launching the agent in visible lab mode..."
$env:RMM_DEMO_MODE = "1"
$env:RMM_LAB_DIR = $LabRoot
$env:RMM_AUDIT_DIR = $LogDir
$env:RMM_STOP_FILE = $StopFile
Start-Process -FilePath $DestPath -WindowStyle Normal

if ($Persist) {
    Write-Host "[*] Creating explicit startup task '$ServiceName'..."
    if (Get-ScheduledTask -TaskName $ServiceName -ErrorAction SilentlyContinue) {
        Unregister-ScheduledTask -TaskName $ServiceName -Confirm:$false
    }

    $TaskCommand = "`$env:RMM_DEMO_MODE='1'; `$env:RMM_LAB_DIR='$LabRoot'; `$env:RMM_AUDIT_DIR='$LogDir'; `$env:RMM_STOP_FILE='$StopFile'; Start-Process -FilePath '$DestPath' -WindowStyle Normal"
    $Action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument "-NoProfile -ExecutionPolicy Bypass -Command $TaskCommand"
    $Trigger = New-ScheduledTaskTrigger -AtStartup
    $Principal = New-ScheduledTaskPrincipal -UserId "NT AUTHORITY\SYSTEM" -LogonType ServiceAccount -RunLevel Highest
    $Settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -DontStopOnIdleEnd -ExecutionTimeLimit (New-TimeSpan -Hours 1)

    Register-ScheduledTask -TaskName $ServiceName -Action $Action -Trigger $Trigger -Principal $Principal -Settings $Settings -Description "RMM blue-team lab monitor" | Out-Null
    Write-Host "[+] Scheduled task registered."
} else {
    Write-Host "[*] Persistence is disabled by default for this lab helper."
}

Write-Host "[*] Deployment complete."
Stop-Transcript | Out-Null
