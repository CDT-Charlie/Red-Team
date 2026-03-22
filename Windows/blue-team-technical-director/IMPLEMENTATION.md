# ARP-Based RMM - Implementation Guide

## Objective

Build a **Layer 2 Remote Monitoring and Management (RMM)** tool that operates exclusively at the Data Link Layer using the Address Resolution Protocol (ARP). This tool enables executing commands on a Windows Server 2022 endpoint and receiving output using only Ethernet frames, bypassing traditional IP-based filtering.

---

## Architecture Overview

The tool encodes command data inside ARP packet metadata fields:
- **Linux Admin Box (Sender):** Fragments commands and sends them in ARP Request packets
- **Windows Server Agent (Receiver):** Sniffs ARP frames, reassembles commands, and executes PowerShell
- **Response Flow:** Agent sends PowerShell output back to Admin via ARP Reply packets

### Communication Flow
```
Admin (Linux):  Fragments "GET_SERVICES" → Sends 4 ARP Requests
Agent (Windows): Sniffs ARP → Reassembles → Executes PowerShell
Agent (Windows): Captures Stdout → Fragments Output → Sends 50 ARP Requests
Admin (Linux):  Sniffs ARP → Reassembles → Prints to Terminal
```

---

## Prerequisites

### Required Tools & Libraries
```bash
sudo apt-get install libpcap-dev
go get github.com/google/gopacket
```

### System Requirements
- **Admin Node:** Ubuntu/Debian Linux  
- **Managed Endpoints:** Windows Server 2022 (Standard/Datacenter)  
- **Driver:** Npcap (WinPcap API-compatible mode)
- **Language:** Go (Golang) with static cross-compilation support

---

## Phase 1: MVP Implementation

### Step 1: Command Fragmentation [fragmentation.go](fragmentation.go)

**Objective:** Break commands into 3-byte data chunks with a 1-byte control header.

**Implementation Details:**
- Each fragment is 4 bytes total (1 byte control + 3 bytes data)
- Control Byte Structure:
  - Bit 7 (MSB): "More Fragments" flag (1 = more coming, 0 = last packet)
  - Bits 0-6: Sequence ID (0 to 127)
- Null-byte pad final packet if needed (receiver trims before exec)

**Example:** `RESTART_IIS` (11 chars) fragments into:
```
[0x81, 'R', 'E', 'S'] (0x81 = More bits + Seq 1)
[0x82, 'T', 'A', 'R'] (0x82 = More bits + Seq 2)
[0x83, 'T', '_', 'I'] (0x83 = More bits + Seq 3)
[0x04, 'I', 'S', 0x00] (0x04 = Final + Seq 4)
```

The sender then **loops through fragments and sends 4 separate ARP requests**.

---

### Step 2: Windows Agent Reassembly [windows-reassembly.go](windows-reassembly.go)

**Objective:** Sniff ARP packets, extract smuggled data, and reconstruct commands.

**Reassembly Logic:**
1. Agent sniffs each incoming ARP packet
2. Extracts and checks Control Byte (Index 0)
3. Stores data (Indices 1-3) in a map: `buffer[sequenceID] = data`
4. When "More Fragments" bit is 0, command is complete
5. Sort map by keys (Sequence IDs) to handle out-of-order arrival
6. Concatenate byte slices into final command string

**Key Pattern:**
- Use a stateful `CommandBuffer` struct with:
  - `Fragments map[int][]byte` (indexed by sequence ID)
  - `IsComplete bool` (tracks receipt of final fragment)
  - `ExpectedCount int` (for validation)

---

### Step 3: PowerShell Execution [ps-execution.go](ps-execution.go) & Command Routing [route-cmd.go](route-cmd.go)

**Objective:** Map abbreviated command strings to hardened PowerShell scripts and execute.

**Execution Flow:**
1. Once reassembly completes, pass string to a **command router**
2. Router maps shorthand commands to full PowerShell scripts:
   - `RESTART_IIS` → `Restart-Service W3SVC`
   - `ST_SRV_MSSQL` → `Start-Service 'MSSQLSERVER'`
   - `whoami` → `whoami /all`
3. Execute via `powershell.exe -NoProfile -Command <string>`
4. Capture stdout and prepare for reverse transmission

**Security Note:** Always use a switch statement to map commands. Never execute raw user input directly (prevents command injection).

---

### Step 4: Agent Sniffing Loop

**Objective:** Run the agent as a continuous background service.

**Implementation:**
- Use `pcap.Handle` in a for loop to continuously sniff
- Set Berkeley Packet Filter (BPF): `arp and dst host [Admin_MAC]`
- This ensures the agent only processes traffic intended for it
- Bind Npcap to the correct network adapter
- Run as a Windows Service (using NSSM for auto-start on reboot)

---

## Phase 1 Success Criteria

- [ ] **Handshake:** Linux box sends "HELO" ARP packet; Windows responds with "READY"
- [ ] **Unidirectional Command:** Trigger `Restart-Service W3SVC` via a single ARP frame
- [ ] **Bidirectional Data:** Execute `hostname` and display result on Linux terminal
- [ ] **Persistence:** Windows Agent runs as background service and survives reboot

---

## Phase 2: Advanced Features (v2.0)

### Step 5: Response Loop & Linux Reassembly [linux-reassembly.go](linux-reassembly.go)

**Challenge:** ARP is inherently unidirectional. How does the Linux Admin see command output?

**Solution: Reverse Fragmentation**
- Windows Agent becomes the "Sender," fragmenting PowerShell output into 3-byte chunks
- Broadcasts response back as ARP packets
- Linux Admin box sniffs these and reassembles them

**Linux-Side Implementation:**
- Use `Sync.Map` or `sync.Mutex`-protected `map[int][]byte` for incoming response fragments
- Implement **ConcurrentTimeout pattern:** Linux must listen until seeing "Final" bit or timeout
- Handle out-of-order packet arrival due to network jitter
- Use gopacket to sniff ARP packets from Windows Server's MAC address

---

### Step 6: MAC Address Hopping [mac-generator.go](mac-generator.go) & [predict-next-mac.go](predict-next-mac.go)

**Objective:** Make traffic appear as randomized network noise.

**Problem:** Seeing thousands of ARP requests from the same MAC address is a loud anomaly.

**Solution: Deterministic MAC Hopping**
1. Use a **Pre-Shared Key (PSK)** and **Sequence ID** as seed
2. Generate deterministic "random" MAC via: `sha256(PSK + seqID)`
3. Both Admin and Agent must stay in sync with the same PSK
4. Each fragment sent uses a new hopped MAC address

**Linux-Side Prediction:**
- Implements **Reverse-Hopping Sniffer:** Pre-calculates expected MAC for current sequence
- Only accepts packets matching mathematical prediction
- Creates a "private channel" on public wire
- Dramatically reduces CPU by ignoring non-matching packets

---

### Step 7: The ARP Shell [run-arp-shell.go](run-arp-shell.go)

**Objective:** State machine managing terminal, sequence IDs, and bidirectional fragmentation.

**State Transitions:**
- **IDLE:** Wait for user input on Linux terminal
- **SEND:** Fragment command, calculate hopped MACs, broadcast ARP packets
- **LISTEN:** Switch to Promiscuous Mode, predict next MACs, reassemble response
- **DISPLAY:** Print PowerShell output, return to IDLE

**Admin Loop (Half-Duplex Operation):**
1. Prompt user for input
2. Fragment the command
3. Send fragments with MAC Hopping (5ms delay between packets to avoid collisions)
4. Shift into "Sniffer Mode" to wait for response
5. Reassemble and display results

---

### Advanced Build Sequence

**Step 1: Handshake**
- Admin sends broadcast ARP with specific "Start" payload
- Windows responds with "ACK"

**Step 2: State Sync**
- Both devices initialize `SequenceID = 0`
- Both load the PSK from configuration

**Step 3: Command Phase**
- Linux sends fragmented command
- Windows reassembles

**Step 4: Response Phase**
- Windows fragments PowerShell output (e.g., 5,000 chars = 1,666 packets at 3 bytes/packet)
- Linux reassembles via Reverse-Hopping with kernel-level BPF filter

**Step 5: Cleanup**
- After final "End of Output" fragment
- Both sides increment SequenceID by large random offset
- Next session starts on completely different set of MACs

---

## Performance & Reliability Optimizations

### Handling Large Outputs
- Windows agent must iterate through output string in a loop
- Create ARP frames and send with 10ms delay (`time.Sleep(10 * time.Millisecond)`)
- Prevents overwhelming network switch buffer

### Reliable Delivery (Pro Feature)
- Add **1-byte CRC8 checksum** to each fragment
- Windows Server sends **ACK Reply** with Sequence ID in Target IP field
- Admin resends missing fragments based on ACKs

### Performance Tuning
- Use **kernel-level BPF filter** on Linux: `arp and ether src [winServerMAC]`
- Linux kernel filters packets before passing to Go, reducing CPU usage
- In Promiscuous Mode, agent can ignore thousands of irrelevant ARP packets

---

## Security Features

### Encryption (v2.0+)
- **XOR Encoding:** Obfuscate payload bytes with pre-shared key before transmission
- **Timestamp Verification:** Include coarse timestamp in control byte to prevent replay attacks
- **Session IDs:** Add 4-bit Session ID to control byte for multi-admin scenarios

### Detection Evasion
By completion, the tool provides:
- **No IP address** → Immune to IP-based firewalling
- **No persistent MAC** → Immune to MAC filtering
- **No TCP/UDP ports** → Invisible to `netstat` and port scanners
- **Metadata-encoded data** → Invisible to Deep Packet Inspection

---

## Deployment

### Windows Setup
1. Install Npcap in "WinPcap API-compatible mode"
2. Ensure Npcap driver has "Bind to Adapter" enabled
3. Deploy `agent.exe` to `C:\ProgramData\WinNetExt`
4. Use **NSSM (Non-Sucking Service Manager)** to register as Windows Service
5. Service auto-starts on reboot

**Note:** Raw packet sniffing typically bypasses Windows Firewall stack, but verify UDP 137/138/139 rules if needed.

### Linux Admin Node
1. Install libpcap-dev
2. Get gopacket library
3. Compile shell binary with static linking
4. Run: `./arpshell --mode admin --psk <shared-key>`

---

## Debugging & Wireshark Analysis

**Key Display Filters:**
- `arp` → Show all ARP traffic
- `eth.src[0] == 0x02` → Locate MAC-hopped packets (locally administered bit)
- `arp.src.proto_ipv4 == 1.2.3.4` → Filter by specific "fake" IP payload

**Packet Inspection:**
- **Sender Protocol Address (SPA):** Contains 4-byte command payload
  - Apply as column in Wireshark to see command strings scrolling
- **Sender MAC Address (SHA):** In v2, changes every packet
  - Verify OUI (first 3 bytes) matches your generator logic
- **Target MAC Address (THA):** Identifies intended recipient

**Common Failure Points:**
- Missing Sequence IDs → Check for dropped packets (e.g., Seq 2 → Seq 4 = missing Seq 3)
- Endianness errors → Ensure `binary.BigEndian` when packing IP fields
- Npcap not bound → Verify correct adapter in system settings

---

## Completion Checklist

- [ ] Prerequisites installed (libpcap-dev, Go, gopacket)
- [ ] [fragmentation.go](fragmentation.go) implements 3-byte chunking with control byte
- [ ] [windows-reassembly.go](windows-reassembly.go) handles ARP sniffing and reassembly
- [ ] [ps-execution.go](ps-execution.go) safely executes PowerShell commands
- [ ] [route-cmd.go](route-cmd.go) maps commands to hardened PowerShell scripts
- [ ] Agent compiles statically and runs as Windows Service
- [ ] [linux-reassembly.go](linux-reassembly.go) receives and reassembles responses
- [ ] [mac-generator.go](mac-generator.go) creates deterministic hopped MACs
- [ ] [predict-next-mac.go](predict-next-mac.go) predicts and filters packets
- [ ] [run-arp-shell.go](run-arp-shell.go) implements state machine shell
- [ ] Phase 1 handshake test passes
- [ ] Phase 1 unidirectional command execution works
- [ ] Phase 1 bidirectional data transfer verified
- [ ] Phase 1 persistence (service auto-start) confirmed
- [ ] Wireshark traces capture all fragments and responses
