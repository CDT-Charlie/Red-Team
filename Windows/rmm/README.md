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
- Npcap installed in WinPcap API-compatible mode
- NSSM binary available for service registration

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

2. Place `dist/agent.exe` and `nssm.exe` into `roles/arp_agent/files/`.

3. Run the playbook:

```bash
ansible-playbook -i inventory.ini site.yml
```

This will:
- Create `C:\ProgramData\WinNetExt` on each target
- Deploy `agent.exe` and `nssm.exe`
- Register the **Windows Network Extension Service** via NSSM
- Start the service with auto-start on reboot

For manual deployment, copy the files and run `scripts/service-mode.ps1` on the target.

## Usage

### Admin Shell (Linux)

```bash
sudo ./dist/arpshell -iface eth0 -psk "S3cur3_Adm1n_K3y" -mvp -target-mac "00:15:5d:01:02:03"
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
