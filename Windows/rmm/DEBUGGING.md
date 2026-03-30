# ARP-RMM Debugging Guide

## Quick Start: Diagnose Timeouts

When `arpshell` times out waiting for responses, follow these steps in order:

### Step 1: Run Ansible Diagnostics (Recommended First Step)
The updated playbook now runs comprehensive diagnostics automatically. After running:
```bash
ansible-playbook -i inventory.ini site.yml
```

Look for output from these diagnostic tasks:
- **Check if Agent process is running** - Shows if agent.exe is active
- **Verify Npcap is loaded** - Confirms driver installation  
- **Verify agent binary is executable** - Tests basic functionality
- **Check Windows Firewall rules** - Identifies Layer 3/4 blocks
- **Capture Event logs** - Shows any errors from system/application

**If any [!] errors appear**, the diagnostics will point you to the problem.

---

### Step 2: Manual Verification on Windows Target

```powershell
# 1. Check if agent process is running
Get-Process -Name "agent" -ErrorAction SilentlyContinue
# Expected: Agent process should appear with PID

# 2. Check service status
Get-Service "Windows Network Extension Service" | Select-Object Status, StartType
# Expected: Status=Running, StartType=Automatic

# 3. Run agent manually to see errors
cd C:\ProgramData\WinNetExt
.\agent.exe -psk "S3cur3_Adm1n_K3y" -mvp
# Keep running. If it crashes, you'll see the error

# 4. Check agent log file (created during execution)
Get-Content "C:\ProgramData\WinNetExt\agent.log"
# Will show detailed startup and runtime logs
```

---

### Step 3: Test Network Connectivity (Linux Admin)

```bash
# In terminal 1: Capture all ARP traffic
sudo tcpdump -i eth1 arp -vv

# In terminal 2: Run arpshell with debug enabled
sudo ./dist/arpshell -iface eth1 -psk "S3cur3_Adm1n_K3y" -mvp \
  -target-mac "fa:16:3e:e2:4a:df" -debug

# Expected in tcpdump:
# 1. ARP Request from your admin MAC to agent MAC (ff:ff:ff:ff:ff:ff for broadcast)
# 2. ARP Reply from agent MAC back to your admin MAC
```

---

## Timeout Scenarios & Solutions

### Scenario 1: Timeout with 0 ARP packets received
**Symptom:** arpshell sends fragments but gets no response

**Possibilities:**
1. Agent process not running
2. Npcap not installed or loaded
3. Wrong network interface on admin side
4. Firewall blocking Layer 2

**Diagnosis:**
```powershell
# On Windows target:
Get-Process agent  # Should exist
Get-PnpDevice -Class "Net" | findstr "Npcap"  # Should exist
Get-EventLog -LogName Application -Newest 5
```

```bash
# On Linux admin:
sudo tcpdump -i 1 arp  # Replace 1 with your interface number
# Should show nothing if packets aren't reaching Windows
# Check: Is Windows on same network segment?
```

### Scenario 2: Timeout with some ARP packets but incomplete response
**Symptom:** Some fragments received but response never completes

**Possibilities:**
1. MAC mismatch between admin and agent
2. Sequence number misalignment
3. PSK (pre-shared key) mismatch
4. Windows adapter has correct interface selected

**Solutions:**
```bash
# Use debug flag to see MAC/seq details
sudo ./dist/arpshell -iface eth1 -psk "S3cur3_Adm1n_K3y" \
  -mvp -target-mac "fa:16:3e:e2:4a:df" -debug
```

Look for `[DEBUG] MAC mismatch:` messages. If yes:
- Verify `-target-mac` matches actual Windows adapter MAC: `Get-NetAdapter` on Windows

### Scenario 3: Agent crashes immediately
**Symptom:** Agent process not found in `Get-Process`

**Check agent log:**
```powershell
Get-Content C:\ProgramData\WinNetExt\agent.log -Tail 50
```

**Common errors:**
- Npcap not installed: Install manually first
- No network interfaces: Run as Administrator, verify adapters exist
- Interface binding failed: Try: `.\agent.exe -iface "\Device\NPF_{GUID}"`

---

## Enable Verbose Logging

### On Windows Agent (Automatic)
Logs are automatically written to `C:\ProgramData\WinNetExt\agent.log` when running as service.

View live activity:
```powershell
Get-Content C:\ProgramData\WinNetExt\agent.log -Wait
```

### On Linux Admin
Use the `-debug` flag:
```bash
sudo ./dist/arpshell -iface eth1 -psk "S3cur3_Adm1n_K3y" -mvp \
  -target-mac "fa:16:3e:e2:4a:df" -debug
```

This will show:
- Each fragment sent (command bytes, sequence numbers, MAC)
- Each ARP packet received (MAC validation, sequence matching)
- Why packets are rejected (if any)

---

## Network Interface Discovery

### On Windows Target
```powershell
# List all physical network adapters
Get-NetAdapter -Physical | Select-Object Name, MacAddress, Status, LinkSpeed

# For Npcap interface names (used with -iface flag)
# Get all available Npcap devices
$devices = [Net.NetworkInformation.NetworkInterface]::GetAllNetworkInterfaces()
foreach ($d in $devices) { 
  Write-Host "$($d.Name): $($d.GetPhysicalAddress())"
}
```

### On Linux Admin
```bash
# List all interfaces with MAC addresses
ip link show

# Or use ifconfig
ifconfig

# Determine which interface can reach Windows target
ping -c 1 192.168.0.24
# Use the interface that successfully pings
```

---

## Pre-Flight Checklist Before Running arpshell

- [ ] Windows agent service is running: `Get-Service "Windows Network Extension Service"`
- [ ] Agent process exists: `Get-Process agent`
- [ ] Npcap driver is installed: `Get-PnpDevice -Class "Net" | findstr "Npcap"`
- [ ] Agent log shows no errors: `Get-Content C:\ProgramData\WinNetExt\agent.log`
- [ ] PSK is identical on both sides: `S3cur3_Adm1n_K3y` (or your custom key)
- [ ] Target MAC is correct: Match output of `Get-NetAdapter` on Windows
- [ ] Network interface is correct on Linux: Verify with `ip link show` and `ping`
- [ ] No firewall blocks between admin and target (optional for Layer 2)

---

## Common Error Messages

| Error | Cause | Solution |
|-------|-------|----------|
| `[FATAL] Npcap verification failed` | Npcap not installed | Manual install: `C:\ProgramData\WinNetExt\npcap-1.87.exe /S /winpcap_mode=yes /loopback_support=yes` |
| `[TIMEOUT] No response from agent` | Agent not running or unreachable | Check agent process, Npcap installation, network connectivity |
| `[DEBUG] MAC mismatch: got X, expected Y` | Admin/agent MAC differ | Verify `-target-mac` matches observed Windows MAC |
| `Failed to enumerate network devices` | Npcap library issue | Ensure Npcap installed in WinPcap-compatible mode |
| `No network interfaces detected by Npcap` | Npcap not in WinPcap mode | Reinstall: choose "WinPcap API-compatible mode" |

---

## Performance Tuning

If you see lots of `[DEBUG] Rejected...` messages:

1. Verify target MAC is exactly right (case-sensitive)
2. Use MVP mode to bypass MAC hopping (`-mvp` flag both sides)
3. Check if PSK character encoding matches (no spaces/special chars if unsure)

---

## Rebuilding with Debug Symbols

If you want to compile with more verbosity:

```bash
cd ~/Red-Team/Windows/rmm

# Add performance tracing
GOFLAG="-v" make agent
GOFLAG="-v" make admin

# Then rebuild and redeploy via Ansible
```

The re-built binaries will automatically write to the agent log with full details.
