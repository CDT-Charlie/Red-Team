# ARP RMM - Deployment & Troubleshooting Guide

**Version:** 1.1 (Enhanced Diagnostics)  
**Date:** March 28, 2026

---

## Summary of Fixes (v1.1)

### Root Cause Analysis

**Previous Error:** The Ansible playbook was attempting to verify Npcap by checking for a Windows service named `"npcap"`. However, **Npcap does NOT expose itself as a service** — this caused the check to fail, which silently skipped all subsequent service installation tasks, leaving the agent undeployed.

### Solutions Implemented

#### 1. **Fixed Ansible Npcap Detection** (Playbook)

**Before:**
```powershell
Get-Service -Name "npcap"  # ❌ Service named "npcap" doesn't exist
```

**After:**
```powershell
$npcapPath = "HKLM:\Software\Npcap"
$winpcapPath = "HKLM:\Software\WinPcap"

if ((Test-Path $npcapPath) -or (Test-Path $winpcapPath)) {
    # ✅ Check registry for actual Npcap installation
}
```

**Impact:** Playbook now correctly detects Npcap instead of silently failing.

---

#### 2. **Added Robust Npcap Validation** (Agent)

The Windows agent (`agent.exe`) now verifies Npcap before attempting to use it:

```go
// Checks if pcap.FindAllDevs() succeeds
// Enumerates all available network interfaces
// Logs detailed interface information for debugging
verifyNpcapAvailable()
```

**Features:**
- ✅ Detects missing or misconfigured Npcap immediately
- ✅ Lists all available network adapters
- ✅ Provides actionable troubleshooting steps
- ✅ Fails fast with clear error message instead of hanging

**Error Message Example:**
```
[FATAL] Npcap verification failed: No network interfaces detected by Npcap

TROUBLESHOOTING:
1. Ensure Npcap is installed from https://npcap.com/
2. During installation, enable 'WinPcap API-compatible mode'
3. Run this agent as Administrator
...
```

---

#### 3. **Enhanced Logging & Debugging**

Both agent and admin shell now provide structured, timestamped logs:

**Agent Logging:**
- `[RX]` - Fragment received (with sequence & control bytes)
- `[TX]` - Transmitting response fragments
- `[OK]` - Command successfully reassembled/executed
- `[DEBUG]` - Detailed packet flow (use `-debug` flag)
- `[ERR]` - Error conditions with context
- `[FATAL]` - Unrecoverable errors with troubleshooting

**Example:**
```
=== ARP RMM Agent v1.1 (Enhanced Diagnostics) ===
[*] Verifying Npcap installation...
[DEBUG] Npcap enumeration successful. Available devices:
[DEBUG]   [0] \Device\NPF_{12345678-1234-5678-ABCD-EFGH12345678}
[DEBUG]       Description: Ethernet Adapter
[DEBUG]       Address[0]: 192.168.1.100
[+] Npcap verification complete
[*] Using specified interface: \Device\NPF_{12345678-1234-5678-ABCD-EFGH12345678}
=== ARP RMM Agent Ready for Incoming Commands ===
[*] Listening for ARP commands...
```

---

#### 4. **Improved Playbook Reliability**

**Before:**
```yaml
- name: Install Agent as Windows Service via NSSM
  when: npcap_check.rc == 0  # ❌ Skipped if Npcap check failed
```

**After:**
```yaml
- name: Fail if Npcap is missing
  fail:
    msg: |
      ERROR: Npcap is not installed...
      SOLUTION: Download from https://npcap.com/
```

**Plus:**
- Service cleanup: Removes old service before reinstalling
- Better error messaging with actionable steps
- Auto-start configuration
- Post-deployment verification

---

## Deployment Workflow (Fixed)

### Prerequisites on Windows Target

1. **Install Npcap** (CRITICAL - This was the missing piece!)
   ```powershell
   # Download: https://npcap.com/
   # Run installer as Administrator
   # ✅ CHECK: "Install Npcap in WinPcap API-compatible Mode"
   # Restart Windows
   ```

2. **Verify Installation**
   ```powershell
   # Check registry
   Get-ItemProperty "HKLM:\Software\Npcap"
   
   # List network adapters Npcap can access
   # (Use PowerShell on target after Npcap install)
   ```

### Deployment from Linux

```bash
cd ~/Red-Team/Windows/rmm

# 1. Build agent
make agent

# 2. Update inventory.ini with Windows target IPs
nano inventory.ini

# 3. Run playbook
ansible-playbook -i inventory.ini site.yml
```

**Expected Output:**
```
TASK [arp_agent : Check if Npcap is installed (Registry check)]
ok: [192.168.0.24]

TASK [arp_agent : Install Agent as a Windows Service via NSSM]
changed: [192.168.0.24]

TASK [arp_agent : Start the Agent service]
changed: [192.168.0.24]

TASK [arp_agent : Verify service is running]
ok: [192.168.0.24]
```

---

## Debugging

### On Windows (Agent)

**Run Agent with debug output:**
```powershell
cd C:\ProgramData\WinNetExt
.\agent.exe -iface "\Device\NPF_{GUID}" -psk "S3cur3_Adm1n_K3y" -mvp -debug
```

**Watch logs:**
```powershell
# Real-time service logs
Get-EventLog -LogName System -Source NSSM -Newest 20 | fl

# Or check agent logs (if redirected)
# tail -f C:\ProgramData\WinNetExt\agent.log
```

**Fix "No network interfaces" error:**
```powershell
# 1. Verify Npcap is installed
Get-ItemProperty "HKLM:\Software\Npcap"

# 2. List network adapters
Get-NetAdapter | fl Name, InterfaceDescription

# 3. Use the adapter name in agent command
# Find an adapter name like: "\Device\NPF_{GUID}"
```

---

### On Linux (Admin Shell)

**Run shell with debug output:**
```bash
sudo ./dist/arpshell -iface eth0 -psk "S3cur3_Adm1n_K3y" -mvp -target-mac "00:15:5d:01:02:03" -debug
```

**Verify libpcap:**
```bash
# Check if libpcap is installed
dpkg -l | grep libpcap

# Install if missing
sudo apt-get install libpcap-dev
```

**Capture ARP traffic with Wireshark/tcpdump:**
```bash
# Terminal 1: Capture all ARP
sudo tcpdump -i eth0 'arp' -X

# Terminal 2: Run admin shell
sudo ./dist/arpshell -iface eth0 -psk "Test123" -mvp -target-mac "00:11:22:33:44:55"
```

---

## Common Issues & Solutions

### Issue 1: "No network interfaces found"

**Cause:** Npcap is not installed or not in WinPcap-compatible mode.

**Solution:**
1. Download Npcap: https://npcap.com/
2. Run installer with Administrator privileges
3. **CHECK** the option: "Install Npcap in WinPcap API-compatible Mode"
4. Complete installation
5. **Restart Windows**
6. Re-run playbook

---

### Issue 2: "Cannot find any service with service name 'npcap'"

**Old Behavior:** Ansible playbook would silently fail and skip service installation.

**New Behavior:** Playbook now properly checks Windows registry and provides actionable error.

**Status:** ✅ FIXED in this version

---

### Issue 3: Agent starts but doesn't receive commands

**Check:**
1. Verify admin shell and agent are using same PSK:
   ```bash
   # On Linux
   ./dist/arpshell -psk "S3cur3_Adm1n_K3y" -mvp ...
   
   # On Windows (in NSSM service)
   # agent.exe -psk "S3cur3_Adm1n_K3y" -mvp
   ```

2. Verify MAC addresses match (MVP mode):
   ```bash
   # Get Windows agent MAC
   ipconfig /all
   
   # Use in admin shell
   ./dist/arpshell -target-mac "00:15:5D:01:02:03"
   ```

3. Verify network connectivity:
   ```bash
   ping <windows-ip>  # From Linux admin box
   arp -a              # Check ARP table
   ```

---

### Issue 4: Service fails to start

**Check service logs:**
```powershell
# View NSSM logs
Get-Content "C:\ProgramData\WinNetExt\WinNetExtension_error.log"

# Or check event viewer for NSSM errors
eventvwr  # -> Windows Logs -> System -> NSSM
```

**Restart service:**
```powershell
Restart-Service -Name "WinNetExtension" -Force
```

---

## Testing Checklist

- [ ] Npcap installed with WinPcap API-compatible mode on Windows
- [ ] Windows target restarted after Npcap installation
- [ ] Playbook passes "Check if Npcap is installed (Registry check)" task
- [ ] Service starts successfully
- [ ] Admin shell connects and sends `HELO` command
- [ ] Agent responds with `READY`
- [ ] Execute `hostname` command successfully
- [ ] Response displays on admin shell
- [ ] Service persists after Windows reboot

---

## Version History

### v1.1 (Current - March 28, 2026)

**Enhancements:**
- ✅ Fixed incorrect Npcap detection (was checking for non-existent service)
- ✅ Added pre-flight Npcap verification in agent
- ✅ Enhanced logging with structured messages
- ✅ Added `-debug` flag for troubleshooting
- ✅ Improved error messages with actionable solutions
- ✅ Better playbook error handling and cleanup
- ✅ Network interface enumeration on startup

**Files Modified:**
- `roles/arp_agent/tasks/main.yml` - Registry-based Npcap check
- `cmd/agent/main.go` - Enhanced diagnostics & validation
- `cmd/arpshell/main.go` - Better error reporting & debugging

### v1.0 (Previous)

- Initial MVP implementation
- Basic Npcap check (had bug with service detection)

---

## Support

For issues not covered here:

1. Check agent log output (especially with `-debug` flag)
2. Review Wireshark captures of ARP traffic
3. Verify PSK and MAC addresses match on both sides
4. Ensure admin box and Windows target are on same network segment
