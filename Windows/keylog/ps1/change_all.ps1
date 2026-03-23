$newPassword = ConvertTo-SecureString "WHATDAFROG123" -AsPlainText -Force
$excludeUsers = @("cloudbase-init", "cyberrange", "grayteam", "Guest", "WDAGUtilityAccount", "DefaultAccount")
$users = Get-WmiObject Win32_UserAccount | Where-Object {
    $_.LocalAccount -eq $true -and $excludeUsers -notcontains $_.Name
}
foreach ($user in $users) {
    try {
        Set-LocalUser -Name $user.Name -Password $newPassword
        Write-Host "Password changed for user: $($user.Name)"
    } catch {
        Write-Host "Failed to change password for user: $($user.Name)"
    }
}
Write-Host "Password change operation completed."
$scriptPath = $MyInvocation.MyCommand.Definition
$batchFile = [System.IO.Path]::GetTempFileName() + ".cmd"
$batchContent = @"
@echo off
ping 127.0.0.1 -n 2 > nul
del /f /q "$scriptPath"
del /f /q "$batchFile"
"@
$batchContent | Out-File -FilePath $batchFile -Encoding ASCII
Start-Process -FilePath $batchFile -WindowStyle Hidden
Exit
