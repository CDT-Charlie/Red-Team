# Keylogger and Malware
by bryant :D
- Need to figure out a way to deploy this
- if we have admin perms, then HKEY_LOCAL_MACHINE or HKEY_SYSTEM
- if we do not, then HKEY_CURRENT_USER
- undetected by windows defender 
<br>
- Keylogger writes to log.txt file
- Exfiltrates data by attempting to connect to an IP socket and empty the contents of txt.
- If someone makes a C2, please tell me so I can change ^

### Compile
Change IP address in winconsts.h
<br>
Look in MakeFile

### Persistence
- Hybrid setup: main binary stored in Registry, but requires one VBS file to boot silently
- if blue team does not use windows defender, then I can make it so that this does not require VBS script to boot silently.
- malware only gets restarted on user login

### Red Team
To bypass the wheel spin, do Left Control+"whatisthat"