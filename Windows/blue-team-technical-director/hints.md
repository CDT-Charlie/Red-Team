# ARP-Based RMM - Implementation Hints & Technical Guide

## Prerequisites

```bash
sudo apt-get install libpcap-dev
go get github.com/google/gopacket
```

**Goal:** Build a Layer 2 Remote Monitoring and Management tool using ARP. Commands are encoded in ARP packet metadata, sent from Linux to Windows, executed, and responses sent back via reverse fragmentation.

---

## Phase 1: MVP - Core Implementation

### Command Fragmentation [fragmentation.go](fragmentation.go)

**Purpose:** Break command strings into 3-byte chunks with 1-byte control header, ready for ARP transmission.

**Key Concepts:**
- **Fragment Size:** 4 bytes total (1 control + 3 data)
- **Control Byte Bit 7:** More Fragments flag (1 = more packets coming, 0 = final)
- **Control Byte Bits 0-6:** Sequence ID (0-127)
- **Data Padding:** Null-byte pad if needed; receiver trims before execution

**Example Breakdown of "RESTART_IIS":**
```
Fragment 1: [0x81, 'R', 'E', 'S'] → Seq 1, More=1
Fragment 2: [0x82, 'T', 'A', 'R'] → Seq 2, More=1
Fragment 3: [0x83, 'T', '_', 'I'] → Seq 3, More=1
Fragment 4: [0x04, 'I', 'S', 0x00] → Seq 4, More=0 (final)
```

**Implementation Notes:**
- Loop through command bytes in 3-byte increments
- Set "More Fragments" bit (0x80) for all but final packet
- Increment sequence ID for each fragment
- Return array of 4-byte payloads ready for ARP field insertion

---

### Windows Agent Reassembly [windows-reassembly.go](windows-reassembly.go)

**Purpose:** Windows endpoint sniffs ARP packets, extracts smuggled data, reconstructs commands.

**Stateful Reassembly Buffer:**
```go
type CommandBuffer struct {
  Fragments    map[int][]byte  // indexed by Sequence ID
  IsComplete   bool            // Final fragment received?
  ExpectedCount int            // For validation
}
```

**Reception Loop:**
1. Sniff incoming ARP packets on network interface
2. Extract Control Byte from ARP Sender Protocol Address (SPA) field (Index 0)
3. Check "More Fragments" bit (Bit 7 of Control Byte)
4. Store data bytes (Indices 1-3) in map: `buffer[seqID] = data[1:4]`
5. When "More Fragments" = 0, command reassembly is complete

**Reassembly Logic:**
1. Sort map keys (Sequence IDs) to handle out-of-order packets
2. Concatenate all data bytes maintaining sort order
3. Trim null-padding from final fragment
4. Return reconstructed command string to execution layer

**Key Pattern:**
- Use `sort.Ints()` on fragment keys before concatenation
- Trim `0x00` bytes from the end using `strings.TrimRight()`
- Validate sequence continuity (no missing IDs)

---

### PowerShell Execution [ps-execution.go](ps-execution.go)

**Purpose:** Safely execute reassembled commands on Windows Server 2022.

**Execution Method:**
```
powershell.exe -NoProfile -Command <reconstructed-string>
```

**Key Flags:**
- `-NoProfile` → Skip loading user/system profiles (faster execution)
- `-Command` → Execute as command string
- Capture stdout for return transmission

**Output Handling:**
- Capture all standard output from PowerShell
- Prepare output for fragmentation (for response loop)
- Handle errors gracefully to avoid agent crashes

---

### Command Router & Parser [route-cmd.go](route-cmd.go)

**Purpose:** Map abbreviated "smuggled" commands to hardened PowerShell scripts.

**Why Smuggling?** ARP payload is limited to 4 bytes per fragment × fragment count. Abbreviating commands minimizes packet overhead.

**Command Mapping Examples:**
- `RESTART_IIS` → `Restart-Service W3SVC`
- `ST_SRV_MSSQL` → `Start-Service 'MSSQLSERVER'`

**Switch Implementation:**
```go
func RouteCommand(input string) string {
  switch input {
  case "who":
    return RunPS("whoami /all")
  default:
    return "Unknown Command"
  }
}
```

**Security:** Never allow direct PowerShell pass-through. Whitelist known commands only.

---

### Agent Sniffing Loop

**Purpose:** Continuously monitor network for inbound ARP commands.

**Sniffing Configuration:**
- Open network interface with `pcap.OpenLive(device, ...)`
- Set BPF (Berkeley Packet Filter): `arp and dst host [Admin_MAC]`
- This ensures agent only processes packets destined for it
- Run in infinite loop; block on packet arrival

**Key Points:**
- Bind Npcap correctly to the network adapter
- Use BPF to filter out noise (other network traffic)
- Requires admin/root privileges (Npcap driver level)
- Loop forever waiting for next command

---

### Windows Service Persistence

**Objective:** Agent runs automatically at boot, survives reboot, and operates without user login.

**Implementation:**
1. Compile Go binary as statically-linked `agent.exe`
2. Deploy to `C:\ProgramData\WinNetExt\agent.exe`
3. Use **NSSM (Non-Sucking Service Manager)**:
   ```
   nssm install "Windows Network Extension" C:\ProgramData\WinNetExt\agent.exe
   nssm start "Windows Network Extension"
   ```
4. Service auto-starts on system reboot
5. Verify with `Get-Service` or `sc query`

**Privilege Elevation:**
- Go binary must manifest with `requireAdministrator` attribute
- Npcap driver requires admin-level packet access
- Service runs in System context (highest privileges)

---

## Phase 1 Success Criteria

- [ ] **Handshake:** Linux sends "HELO" ARP packet; Windows responds with "READY"
- [ ] **Unidirectional Command:** Single ARP frame triggers `Restart-Service W3SVC`
- [ ] **Bidirectional Data:** Execute `hostname` and see result on Linux terminal
- [ ] **Persistence:** Agent survives Windows reboot

---

## Phase 2: Advanced Features (v2.0+)

### Response Loop & Linux Reassembly [linux-reassembly.go](linux-reassembly.go)

**Problem:** How does the Linux Admin see command output if ARP is nominally unidirectional?

**Solution: Reverse Fragmentation**
- Windows Agent becomes a "Sender," fragmenting PowerShell output into 3-byte chunks
- Sends fragments back as ARP Reply packets
- Linux Admin sniffs and reassembles responses

**Linux Response Architecture:**
- Use `Sync.Map` or mutex-protected `map[int][]byte` for incoming fragments
- Implement **ConcurrentTimeout pattern:**
  - Listen until receiving packet with "Final" bit (More Fragments = 0)
  - OR timeout after N seconds if Windows is slow
- Handle out-of-order arrival (network can reorder packets)
- Use gopacket to sniff ARP packets from Windows Server's MAC

---

### MAC Address Hopping [mac-generator.go](mac-generator.go)

**Problem:** Sending thousands of ARP requests from the same MAC is a loud anomaly that security tools detect.

**Solution: Deterministic MAC Hopping**
- Generate new pseudo-random MAC for every packet
- Use **Pre-Shared Key (PSK) + Sequence ID** as seed
- Both Admin and Agent stay in sync via same PSK and incrementing SeqID

**Generation Logic:**
```
Deterministic Hash = SHA256(PSK + sequence_id)
Hopped MAC = [0x02, hash[0], hash[1], hash[2], hash[3], hash[4]]
             ↑ Locally Administered bit (0x02) marks as virtual
```

---

### Reverse-Hopping Sniffer [predict-next-mac.go](predict-next-mac.go)

**Purpose:** Linux Admin predicts which MAC address the Windows Agent will use next, creating a "private channel" on public wire.

**Prediction Logic:**
1. Pre-calculate expected MAC: `SHA256(PSK + current_seqID)`
2. Sniff network interface
3. Ignore all packets NOT from the predicted MAC
4. When packet arrives from predicted MAC, process it
5. Increment Sequence ID for next prediction

**Performance Benefit:**
- Kernel-level BPF filter: `arp and ether src [predicted_mac]`
- Drastically reduces CPU usage vs software-only filtering

---

### The ARP Shell - State Machine [run-arp-shell.go](run-arp-shell.go)

**Purpose:** Interactive shell managing bidirectional half-duplex fragmentation and MAC hopping.

**State Machine States:**
- **IDLE:** Prompt user for input, wait for command
- **SEND:** Fragment command, calculate hopped MACs, transmit ARP packets
- **LISTEN:** Shift to Promiscuous Mode, predict response MACs, receive fragments
- **DISPLAY:** Reassemble and print command output, return to IDLE

**Half-Duplex Operation:**
- Only one direction transmits at a time (Admin → Agent → Admin)
- Synchronization via Sequence ID counter (both sides increment together)

---

## Performance & Evasion

### Handling Large Outputs
- Windows sends fragments with 10ms delay: `time.Sleep(10 * time.Millisecond)`
- Prevents overwhelming network switch buffer
- Linux reassembles all fragments using reverse-hopping

### Kernel-Level BPF Filtering
- Prevents CPU waste on non-matching packets
- Linux kernel filters at driver level

### Reliable Delivery (Pro Feature)
- Add **1-byte CRC8 checksum** to each fragment
- Windows sends ACK for each received fragment
- Admin tracks and resends missing fragments

---

## Security Features (v2.0+)

**XOR Encoding:** Obfuscate payload with pre-shared key
**Timestamp Verification:** Prevent replay attacks
**Session IDs:** Support multi-admin scenarios

**Result:**
- ✓ No IP address → Immune to IP-based firewalling
- ✓ No persistent MAC → Immune to MAC filtering  
- ✓ No TCP/UDP ports → Invisible to `netstat` and port scanners
- ✓ Metadata-encoded → Invisible to Deep Packet Inspection

---

## Debugging with Wireshark

**Key Display Filters:**
```
arp                              # Show all ARP traffic
eth.src[0] == 0x02              # Locally administered bit (hopped MACs)
arp.src.proto_ipv4 == 1.2.3.4   # Specific fake IP filter
```

**Troubleshooting:**
| Problem | Solution |
|---------|----------|
| Packet drops | Check Wireshark for missing Seq IDs |
| Endianness errors | Use `binary.BigEndian` when packing fields |
| Npcap not binding | Verify correct adapter in system settings |

---

## Completion Checklist

**Phase 1 (MVP):**
- [ ] Dependencies installed
- [ ] Fragmentation logic correct
- [ ] Windows reassembly accurate
- [ ] PowerShell execution safe
- [ ] Agent sniffs continuously
- [ ] Agent persists as Windows Service
- [ ] Handshake succeeds
- [ ] Single command executes
- [ ] Bidirectional response works
- [ ] Agent survives reboot

**Phase 2 (v2.0+):**
- [ ] Linux sniffs responses correctly
- [ ] MAC hopping generates deterministic chain
- [ ] Prediction logic stays in sync
- [ ] ARP Shell state machine works
- [ ] Large outputs split/reassemble
- [ ] Out-of-order packets handled
- [ ] Wireshark shows hopped MACs changing