#!/bin/bash

# ============================================================================
# RMM Blue Team Monitor file transfer helper
# Explicit lab staging for Windows hosts
# ============================================================================

TARGET=""
USER="Administrator"
SSH_PORT="22"
DEST_DIR="C:\ProgramData\RMMBlueTeam"
KEY_FILE=""

usage() {
    echo "RMM Blue Team Monitor SCP helper"
    echo "Usage: $0 -t <target_ip> [-u <username>] [-p <ssh_port>] [-d <remote_dir>] [-i <ssh_key>]"
    echo ""
    echo "Options:"
    echo "  -t  Target Windows IP address"
    echo "  -u  SSH username (default: Administrator)"
    echo "  -p  SSH port (default: 22)"
    echo "  -d  Destination directory on the target (default: C:\\ProgramData\\RMMBlueTeam)"
    echo "  -i  SSH private key file"
    echo "  -h  Show this help menu"
    exit 1
}

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

if [ -z "$TARGET" ]; then
    echo "[-] Error: target IP (-t) is required."
    usage
fi

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
BASE_DIR="$(dirname "$SCRIPT_DIR")"

FILES=(
    "$BASE_DIR/dist/rmm-agent.exe"
    "$SCRIPT_DIR/deploy.ps1"
)

echo "[*] Checking local artifacts..."
for file in "${FILES[@]}"; do
    if [ ! -f "$file" ]; then
        echo "[-] Missing file: $file"
        exit 1
    fi
done

echo "[*] Ensuring destination exists on $TARGET..."
ssh $KEY_FILE -p $SSH_PORT "${USER}@${TARGET}" "powershell -Command \"if (-not (Test-Path '${DEST_DIR}')) { New-Item -ItemType Directory -Force -Path '${DEST_DIR}' | Out-Null }\""

echo "[*] Copying files to ${USER}@${TARGET}:${DEST_DIR}"
for file in "${FILES[@]}"; do
    filename=$(basename "$file")
    echo "    -> $filename"
    scp $KEY_FILE -P $SSH_PORT "$file" "${USER}@${TARGET}:${DEST_DIR}/$filename"
    if [ $? -ne 0 ]; then
        echo "[-] Transfer failed for $filename"
        exit 1
    fi
done

echo "[+] Lab staging complete."
echo "[*] On the target, run: powershell -ExecutionPolicy Bypass -File C:\\ProgramData\\RMMBlueTeam\\deploy.ps1"
