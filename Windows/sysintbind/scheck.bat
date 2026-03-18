@echo off
REM scheck.bat - scheduled check installer
REM Creates scheduled tasks to run ncheck.bat on startup and every 10 minutes.
REM Run this as Administrator on the target Windows host.

setlocal ENABLEDELAYEDEXPANSION

REM Resolve full path to ncheck.bat in this directory
set "NCHECK=%~dp0ncheck.bat"

if not exist "%NCHECK%" (
    echo [!] ncheck.bat not found next to this script: "%NCHECK%"
    echo     Make sure you copied both files to the same folder.
    goto :eof
)

echo [*] Using ncheck.bat at: "%NCHECK%"

set "TASK_CMD=\"%NCHECK%\""

echo [*] Creating startup task (runs at boot under SYSTEM)...
schtasks /create ^
    /tn "scheck-startup" ^
    /tr %TASK_CMD% ^
    /sc ONSTART ^
    /ru SYSTEM ^
    /RL HIGHEST ^
    /F

echo [*] Creating repeating task (runs every 10 minutes under SYSTEM)...
schtasks /create ^
    /tn "scheck-10min" ^
    /tr %TASK_CMD% ^
    /sc MINUTE ^
    /mo 10 ^
    /ru SYSTEM ^
    /RL HIGHEST ^
    /F

echo [*] Done.
echo     - Task 'scheck-startup' will run on boot.
echo     - Task 'scheck-10min' will run every 10 minutes.

endlocal
exit /b 0

