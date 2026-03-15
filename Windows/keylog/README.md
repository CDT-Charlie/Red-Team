# Keylogger and Malware
by bryant :D
<br>
- Keylogger writes to log.txt file
- log.txt will be in Users\Public

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

### Reverse Shell
- Spawns as SYSTEM
- has a local "cd" command so it keeps track of where you are
- Takes in 3 commands:
    - 0exec [] => basically WinExec
    - 0log [path/to/file] => reads a file
    - 0spawn [IP] [PORT] => terminates current session and creates a new shell that connects to given IP and PORT

### Compile
Change IP address in winconsts.h
Ask Bryant

### Persistence
- Hybrid setup: main binary stored in Registry, but requires one VBS file to boot silently<br>
- malware only gets restarted on user login
- Reverse shell runs as a service
So basically, each user gets their own keylogger and wheel! Each Windows VM gets their own reverse shell.

### Red Team
To bypass the wheel spin, do LeftControl+"whatisthat"<br>