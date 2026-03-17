#!/bin/bash

# ============================================================================
# SewerRat SCP File Transfer Script
# Alternative to HTTP (Python server) for transferring artifacts
# ============================================================================

# Default variables
TARGET=""
USER="Administrator"
SSH_PORT="22"
DEST_DIR="C:\Windows\Temp"
KEY_FILE=""

# Display usage instructions
usage() {
    echo "SewerRat SCP Deployment Script"
    echo "Usage: $0 -t <target_ip> [-u <username>] [-p <ssh_port>] [-d <remote_dir>] [-i <ssh_key>]"
    echo ""
    echo "Options:"
    echo "  -t  Target Windows IP address (Required)"
    echo "  -u  SSH Username (Default: Administrator)"
    echo "  -p  SSH Port (Default: 22)"
    echo "  -d  Destination directory on target (Default: C:\Windows\Temp)"
    echo "  -i  Identity file (SSH Private Key) (Optional)"
    echo "  -h  Show this help menu"
    echo ""
    echo "Example: $0 -t 10.1.1.2 -u Administrator -d 'C:\Users\Administrator\Desktop'"
    exit 1
}

# Parse command line arguments
while getopts "t:u:p:d:i:h" opt; do
    case "$opt" in
        t) TARGET=$OPTARG ;;
        u) USER=$OPTARG ;;
        p) SSH_PORT=$OPTARG ;;
        d) DEST_DIR=$OPTARG ;;
        i) KEY_FILE="-i $OPTARG" ;;
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

# Validate local files exist before starting transfer
echo "[*] Checking local artifacts..."
for file in "${FILES[@]}"; do
    if [ ! -f "$file" ]; then
        echo "[-] Error: Required file not found -> $file"
        echo "    Tip: Make sure you ran 'make all' to build SewerRat.exe in /dist"
        exit 1
    fi
done

# Create target directory using SSH just in case it doesn't exist
echo "[*] Connecting to $TARGET to ensure '$DEST_DIR' exists..."
ssh $KEY_FILE -p $SSH_PORT "${USER}@${TARGET}" "powershell -Command \"if (-not (Test-Path '${DEST_DIR}')) { New-Item -ItemType Directory -Force -Path '${DEST_DIR}' | Out-Null }\""

# Optional: User may be prompted for password if not utilizing SSH keys
echo "[*] Moving files via SCP to ${USER}@${TARGET}:${DEST_DIR}"

for file in "${FILES[@]}"; do
    filename=$(basename "$file")
    echo "    -> Transferring $filename..."
    scp $KEY_FILE -P $SSH_PORT "$file" "${USER}@${TARGET}:${DEST_DIR}/$filename"
    
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
