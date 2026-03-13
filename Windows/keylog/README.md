# Keylogger and Malware
by bryant :D
<br>
- Keylogger writes to log.txt file
- Exfiltrates data by attempting to connect to an IP socket and empty the contents of txt.
- If someone makes a C2, please tell me so I can change ^

### Sea green Wheel
- Spin the wheel!
- effects persists until the next wheel (5 minutes)
- Has 6 effects:
  - Does nothing
  - Swaps L/R mouse buttons
  - Continuously minimizes the foremost window every 30 seconds
  - Logs out the user
  - Randomly swaps 2 keys
  - reaction game

### Compile
Change IP address in winconsts.h
<br>
Look in MakeFile for commands
<br>

### Persistence
- Hybrid setup: main binary stored in Registry, but requires one VBS file to boot silently<br>
- malware only gets restarted on user login

### Red Team
To bypass the wheel spin, do LeftControl+"whatisthat"<br>