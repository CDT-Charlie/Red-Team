# ARP-Based RMM

A Layer 2 Remote Monitoring and Management tool that operates exclusively at the Data Link Layer using the Address Resolution Protocol (ARP). Commands are encoded in ARP packet metadata fields, sent from a Linux admin box to a Windows Server 2022 endpoint, executed via PowerShell, and responses returned over the same channel.

## Architecture

```
Admin (Linux)                              Agent (Windows Server 2022)
┌─────────────┐    ARP Request             ┌──────────────────────┐
│  arpshell   │  ─── SPA = cmd frags ───>  │     agent.exe        │
│             │                            │  ┌────────────────┐  │
│  Fragment   │                            │  │ Reassemble cmd │  │
│  Send       │    ARP Reply               │  │ Route + Exec   │  │
│  Listen     │  <── SPA = out frags ───   │  │ Fragment output│  │
│  Display    │                            │  └────────────────┘  │
└─────────────┘                            └──────────────────────┘
```

Data is smuggled in the 4-byte **Sender Protocol Address (SPA)** field of ARP packets:
- 1 byte control header (bit 7 = more-fragments, bits 0-6 = sequence ID)
- 3 bytes payload data per fragment

## Technology Stack

| Component       | Technology                                      |
|-----------------|------------------------------------------------|
| Control Node    | Linux (Ubuntu/Debian)                          |
| Endpoints       | Windows Server 2022                            |
| Language        | Go with `github.com/google/gopacket`           |
| Packet Driver   | Npcap (WinPcap API-compatible) on Windows      |
| Deployment      | Ansible via WinRM                              |
| Persistence     | NSSM (Non-Sucking Service Manager)             |

## Project Layout

```
cmd/agent/           Windows agent binary (compiles to agent.exe)
cmd/arpshell/        Linux admin shell binary (compiles to arpshell)
internal/fragment/   Fragmentation and reassembly logic
internal/craft/      ARP packet crafting and transmission
internal/mac/        Deterministic MAC hopping (PSK + SHA256)
internal/execution/  PowerShell execution and command routing
roles/arp_agent/     Ansible role for deployment
scripts/             Service registration helpers
```

## Prerequisites

**Linux Admin Box:**
```bash
sudo apt-get install libpcap-dev
go get github.com/google/gopacket
```

**Windows Targets:**
- Administrator access (required for service installation and Npcap setup)
- No pre-installation needed — Npcap and NSSM are deployed automatically

## Build

Cross-compile the Windows agent from the Linux admin box:

```bash
make agent    # produces dist/agent.exe
```

Build the Linux admin shell:

```bash
make admin    # produces dist/arpshell
```

Build both:

```bash
make all
```

## Deployment (Ansible)

1. Edit `inventory.ini` with your Windows Server IPs and credentials.

2. Verify the deployment files are present:
   - `dist/agent.exe` - Compiled agent binary
   - `nssm.exe` - Service manager
   - `npcap-1.87.exe` - Packet capture driver installer (**included in repo**)

3. **CRITICAL: Pre-install Npcap on the target manually:**
   ```powershell
   # On Windows target as Administrator:
   C:\ProgramData\WinNetExt\npcap-1.87.exe /S /winpcap_mode=yes /loopback_support=yes
   # Wait 2-3 minutes for silent installation to complete
   ```

4. Run the playbook:

```bash
ansible-playbook -i inventory.ini site.yml
```

This will:
- Verify Npcap is already installed (now required to be pre-installed)
- Create `C:\ProgramData\WinNetExt` on each target
- Deploy `agent.exe` and `nssm.exe`
- Register the **Windows Network Extension Service** via NSSM
- Start the service with auto-start on reboot
- **Run comprehensive diagnostics** to verify agent is running and Npcap is functional

**Note:** If Npcap is not pre-installed, the playbook will fail with clear instructions to install it first, then re-run.

For manual deployment, copy the files and run `scripts/service-mode.ps1` on the target.

## Usage

### Admin Shell (Linux)

```bash
sudo ./dist/arpshell -iface eth1 -psk "S3cur3_Adm1n_K3y" -mvp -target-mac "fa:16:3e:e2:4a:df"
```

Flags:
- `-iface` - Network interface (default: `eth0`)
- `-psk` - Pre-shared key for MAC hopping
- `-mvp` - MVP mode (static MACs, no hopping)
- `-target-mac` - Agent MAC filter for MVP mode

### Agent (Windows)

```cmd
agent.exe -iface "\Device\NPF_{GUID}" -psk "S3cur3_Adm1n_K3y" -mvp
```

Flags:
- `-iface` - Npcap interface name (auto-detected if omitted)
- `-psk` - Pre-shared key (must match admin)
- `-mvp` - MVP mode (accept all ARP, no MAC validation)
- `-admin-mac` - Filter by admin MAC in MVP mode

### Supported Commands

| Command        | PowerShell Equivalent                           |
|----------------|------------------------------------------------|
| `HELO`         | Handshake (returns `READY`)                    |
| `who`          | `whoami /all`                                  |
| `hostname`     | `hostname`                                     |
| `net`          | `Get-NetIPAddress` (interface + IP listing)    |
| `RESTART_IIS`  | `Restart-Service W3SVC -Force`                 |
| `ST_SRV_MSSQL` | `Start-Service 'MSSQLSERVER'`                  |
| `GET_SERVICES` | `Get-Service` (running, first 10)              |
| `DISK_USAGE`   | `Get-PSDrive C` (used/free)                   |
| `uptime`       | Last boot time via CIM                         |
| `arp`          | `Get-NetNeighbor` (ARP table)                  |

## Phase 1: MVP Checklist

- [ ] **Handshake:** Send `HELO`, receive `READY`
- [ ] **Unidirectional:** Trigger `Restart-Service W3SVC` via ARP
- [ ] **Bidirectional:** Execute `hostname` and see output on Linux
- [ ] **Persistence:** Agent survives Windows reboot

## Phase 2: Roadmap (v2.0)

- **MAC Hopping:** Randomize source MAC per fragment using PSK + SHA256
- **Fragmentation Engine:** Large data transfers across hundreds of ARP packets
- **Encryption:** XXTEA or AES-GCM on payloads
- **Self-Destruct:** Kill-packet sequence to wipe the agent from disk

## Debugging with Wireshark

| Filter                           | Purpose                                   |
|----------------------------------|-------------------------------------------|
| `arp`                            | Show all ARP traffic                      |
| `eth.src[0] == 0x02`            | Locally administered bit (hopped MACs)    |
| `arp.src.proto_ipv4 == 1.2.3.4` | Filter by specific fake IP payload        |

**Common issues:**
- Packets appear but agent ignores them: verify Npcap is bound to the correct adapter
- `HELO` appears as `OLEH`: use `binary.BigEndian` when packing IP fields
- Missing sequence IDs in Wireshark: switch dropped a packet, check for gaps

## Troubleshooting Timeouts

If `arpshell` times out waiting for responses:

### 1. **Verify Agent is Running (on Windows target)**

```powershell
# Check if agent.exe process exists
Get-Process -Name "agent" -ErrorAction SilentlyContinue

# If not running, check service status
Get-Service "Windows Network Extension Service" | Select-Object Status, StartType

# Check recent errors
Get-EventLog -LogName Application -Source "*agent*" -Newest 5 -ErrorAction SilentlyContinue
Get-EventLog -LogName System -Source "NSSM" -Newest 5 -ErrorAction SilentlyContinue
```

### 2. **Verify Npcap is Installed and Loaded (on Windows target)**

```powershell
# Check if Npcap directory exists
Test-Path "C:\Program Files\Npcap"

# Check Npcap registry (32-bit hive)
Get-Item "HKLM:\Software\Wow6432Node\Npcap" -ErrorAction SilentlyContinue

# Check if Npcap driver is loaded in Device Manager
Get-PnpDevice -Class "Net" | Where-Object { $_.Name -match "Npcap" }
```

### 3. **Test Agent Manually (on Windows target)**

```powershell
# Run agent directly (not via NSSM service) to see error output
cd C:\ProgramData\WinNetExt

# Try with auto-detect interface
.\agent.exe -psk "S3cur3_Adm1n_K3y" -mvp

# Or specify interface explicitly (find via: Get-NetAdapter)
.\agent.exe -iface "\Device\NPF_{GUID}" -psk "S3cur3_Adm1n_K3y" -mvp
```

### 4. **Verify Network Connectivity (on Linux admin box)**

```bash
# Capture ARP packets from the target
sudo tcpdump -i eth1 arp -v

# While running arpshell in another terminal
sudo ./dist/arpshell -iface eth1 -psk "S3cur3_Adm1n_K3y" -mvp -target-mac "TARGET_MAC_HERE"
```

**Expected behavior:** You should see ARP Request and ARP Reply packets in tcpdump. If only Request appears but no Reply:
- Agent isn't running
- Agent can't access Npcap interface
- Network firewall is blocking Layer 2 traffic (unlikely for ARP)

### 5. **Common Diagnostic Steps**

**Problem: Agent process crashes immediately**
- Check Event logs for stack traces
- Try manual execution: `C:\ProgramData\WinNetExt\agent.exe`
- Verify Npcap is correctly installed: `C:\Program Files\Npcap\` exists

**Problem: Agent runs but doesn't respond to ARP**
- Verify correct network interface: `Get-NetAdapter`
- Ensure arpshell target MAC matches Windows adapter MAC
- Try MVP mode to bypass MAC validation: agent.exe `-mvp`
- Check if Windows Firewall blocks Npcap (unlikely): `netsh advfirewall`

**Problem: Npcap installation failed**
- Run interactively on Windows: `C:\ProgramData\WinNetExt\npcap-1.87.exe`
- Choose "Install Npcap in WinPcap API-compatible mode"
- Reboot if prompted
- Verify: `Test-Path "C:\Program Files\Npcap"`

**Problem: NSSM Service reports errors**
- Check service installation: `nssm.exe query "Windows Network Extension Service"`
- View service log: `nssm.exe get "Windows Network Extension Service" AppStdout`
- Re-install service:
  ```powershell
  sc.exe delete "Windows Network Extension Service"
  Start-Sleep -Seconds 5
  nssm.exe install "Windows Network Extension Service" C:\ProgramData\WinNetExt\agent.exe
  nssm.exe set "Windows Network Extension Service" Start SERVICE_AUTO_START
  nssm.exe start "Windows Network Extension Service"
  ```

### 6. **Enable Verbose Logging (future enhancement)**

Currently, agent.exe has minimal output when run as a service. To add verbose logging:

1. Modify agent source to write to a log file:
   ```go
   // In cmd/agent/main.go
   logFile, _ := os.OpenFile("C:\\ProgramData\\WinNetExt\\agent.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
   log.SetOutput(logFile)
   ```

2. Rebuild: `make agent`

3. Re-deploy and check: `cat C:\ProgramData\WinNetExt\agent.log`
