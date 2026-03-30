# UDP RMM Admin Shell v2.0 - Testing Guide

## Overview
This guide walks through testing the **UDP RMM Admin Shell**, a cloud-ready remote command execution platform using fragmented UDP packets.

---

## Prerequisites

### System Requirements
- **OS**: Linux or Windows with Go 1.21+
- **Go**: Install from https://golang.org/dl/
- **Network**: Direct UDP connectivity between admin and agent on port 9999

### Install Go Dependencies
```bash
cd Windows/rmm
go mod download
```

---

## Building

### Build Admin Shell (arpshell)
```bash
cd Windows/rmm/cmd/arpshell
go build -o arpshell main.go
```
**Output**: `arpshell` (Linux) or `arpshell.exe` (Windows)

### Build Agent (rmm-agent)
```bash
cd Windows/rmm/cmd/rmm-agent
go build -o rmm-agent main.go
```
**Output**: `rmm-agent` (Linux) or `rmm-agent.exe` (Windows)

---

## Test Environment Setup

### Option 1: Local Testing (Same Machine)

#### Terminal 1: Start the Agent
```bash
cd Windows/rmm/cmd/rmm-agent
./rmm-agent -debug
```
**Expected output:**
```
[*] Starting UDP RMM Agent v2.0...
[*] Listening on 127.0.0.1:9999
[DEBUG] Listening for commands...
```

#### Terminal 2: Run Admin Shell
```bash
cd Windows/rmm/cmd/arpshell
./arpshell -target 127.0.0.1:9999 -debug
```
**Expected output:**
```
=== UDP RMM Admin Shell v2.0 (Cloud-Ready) ===
[*] Target: 127.0.0.1:9999
[*] Type 'quit' to exit

UDP-Admin>
```

---

### Option 2: Network Testing (Different Machines)

#### On Target Machine (Windows Server)
```bash
.\rmm-agent.exe -debug
```

#### On Admin Machine (Linux/Windows)
```bash
./arpshell -target <TARGET_IP>:9999 -debug
```

Replace `<TARGET_IP>` with the actual IP address of the target machine.

---

## Test Cases

### Test 1: Simple Echo Command
```
UDP-Admin> echo hello world
[TX] Fragmenting command into 1 UDP packet(s)
[DEBUG] Command: "echo hello world" (16 bytes)
[DEBUG] Sent fragment 1/1 (16 bytes)
[*] Waiting for response (max 30 seconds)...
[DEBUG] Listening on 127.0.0.1:XXXXX for response

--- RESPONSE ---
hello world
----------------
```

### Test 2: Command with Output
```
UDP-Admin> whoami
[TX] Fragmenting command into 1 UDP packet(s)
[DEBUG] Command: "whoami" (6 bytes)
[DEBUG] Sent fragment 1/1 (6 bytes)
[*] Waiting for response (max 30 seconds)...

--- RESPONSE ---
DOMAIN\USERNAME
----------------
```

### Test 3: Fragmented Command
```
UDP-Admin> powershell.exe -Command "Get-Process | Where-Object {$_.CPU -gt 10} | Select-Object Name, CPU, Memory"
[TX] Fragmenting command into 3 UDP packet(s)
[DEBUG] Command: "powershell.exe -Command ..." (123 bytes)
[DEBUG] Sent fragment 1/3 (4 bytes)
[DEBUG] Sent fragment 2/3 (4 bytes)
[DEBUG] Sent fragment 3/3 (4 bytes)
[*] Waiting for response (max 30 seconds)...

--- RESPONSE ---
<PowerShell output>
----------------
```

### Test 4: Invalid Target (Timeout)
```
UDP-Admin> whoami
[TX] Fragmenting command into 1 UDP packet(s)
[DEBUG] Command: "whoami" (6 bytes)
[DEBUG] Sent fragment 1/1 (6 bytes)
[*] Waiting for response (max 30 seconds)...

[TIMEOUT] No response from agent within 30 seconds.
Verify:
  1. Agent is running on target
  2. Agent is listening on port 9999
  3. Network connectivity exists
  4. Firewall allows UDP:9999
  5. Target IP:port is correct

UDP-Admin>
```

---

## Debug Mode

Enable detailed packet-level logging with the `-debug` flag:

```bash
./arpshell -target 192.168.1.100:9999 -debug
```

### Debug Output Includes:
- Command bytes and fragmentation details
- Sent UDP fragment info (sequence, size)
- Received packet count and bytes
- Fragment reassembly progress
- Response completion status

---

## Troubleshooting

| Issue | Cause | Solution |
|-------|-------|----------|
| `[TIMEOUT] No response` | Agent not running | Start agent with `./rmm-agent -debug` |
| `[ERROR] Invalid target format` | Wrong target syntax | Use format `IP:PORT` (e.g., `192.168.1.100:9999`) |
| `[!] Command too long (max 512 bytes)` | Command exceeds limit | Split into smaller commands |
| Connection refused | Firewall blocking UDP 9999 | Check firewall rules: `sudo ufw allow 9999/udp` |
| Agent crashes | Unknown command | Check agent logs for command execution errors |

---

## Performance Testing

### Fragment Distribution Test
Send a 400-byte command and verify it fragments correctly:

```bash
# Create a test string (128 bytes of 'a' repeated to ~400 bytes)
UDP-Admin> powershell -Command "Write-Output $('a' * 400)"

[TX] Fragmenting command into approximately 100 UDP packet(s)
```

The admin shell will automatically fragment into 4-byte chunks (3 bytes data + 1 control).

### Timeout Stress Test
Send 10 rapid commands:
```bash
UDP-Admin> whoami
UDP-Admin> echo test
UDP-Admin> dir
UDP-Admin> ipconfig
...
```

Verify responses remain stable and no packets are dropped.

---

## Network Capture (Optional)

To inspect UDP packets on the wire:

### Linux:
```bash
sudo tcpdump -i any -n udp port 9999 -A
```

### Windows (with Npcap):
```bash
tshark -i adapter_name -f "udp.port == 9999"
```

Expected packet format:
```
[Control Byte] [Data...]
Control Byte: [More-Frags Flag (1 bit) | Sequence ID (7 bits)]
```

---

## Cleanup

### Stop Agent
Press `Ctrl+C` in agent terminal.

### Exit Admin Shell
```
UDP-Admin> quit
[*] Goodbye!
```

---

## Next Steps

- [ ] Test with real target agent
- [ ] Verify firewall rules allow UDP:9999
- [ ] Stress test with large commands (up to 512 bytes)
- [ ] Monitor packet loss in high-latency networks
- [ ] Implement response encryption (optional)
