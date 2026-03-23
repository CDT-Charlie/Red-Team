$currentSessionId = (Get-Process -Id $PID).SessionId
$sessions = quser 2>$null

if ($sessions) {
    $sessions[1..($sessions.Count - 1)] | ForEach-Object {
        $line = $_.Trim()
        if ($line -match '(?<UserName>\S+)\s+(?<SessionName>\S+)?\s+(?<Id>\d+)\s+') {
            $sessionId = $Matches['Id']
            
            if ($sessionId -ne $currentSessionId) {
                Write-Host "Terminating Session ID: $sessionId" -ForegroundColor Cyan
                logoff $sessionId
            } else {
                Write-Host "Skipping current session ($sessionId)" -ForegroundColor Yellow
            }
        }
    }
} else {
    Write-Host "No active sessions found." -ForegroundColor Green
}
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