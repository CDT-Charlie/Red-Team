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

## Deployment Workflow (Fixed & Automated)

### Prerequisites on Windows Target

Ansible playbook now handles everything automatically:
- ✅ Npcap installation (with WinPcap API-compatible mode)
- ✅ Agent binary deployment
- ✅ Service registration
- ✅ Auto-start configuration

**All you need:**
- Administrator access on Windows targets
- Network connectivity to Windows targets from Linux admin box
- WinRM enabled on Windows (Ansible requirement)

### Deployment from Linux

```bash
cd ~/Red-Team/Windows/rmm

# 1. Build agent
make agent

# 2. Update inventory.ini with Windows target IPs
nano inventory.ini

# 3. Run playbook - Npcap is installed automatically
ansible-playbook -i inventory.ini site.yml
```

**Expected Output:**
```
TASK [arp_agent : Copy Npcap installer]
changed: [192.168.0.24]

TASK [arp_agent : Check if Npcap is already installed (Registry check)]
ok: [192.168.0.24]

TASK [arp_agent : Install Npcap (WinPcap API-compatible mode)]
changed: [192.168.0.24]
[*] Starting Npcap installation...
[*] Executing: C:\ProgramData\WinNetExt\npcap-1.87.exe /S /winpcap_mode=yes /loopback_support=yes
[*] Npcap installer exited with code: 0
[+] Npcap installation completed

TASK [arp_agent : Wait for file system to stabilize after Npcap installation]
ok: [192.168.0.24]

TASK [arp_agent : Verify Npcap was installed]
ok: [192.168.0.24]
✓ Npcap installation verified (Registry)

TASK [arp_agent : Install Agent as a Windows Service via NSSM]
changed: [192.168.0.24]
Service installed successfully: WinNetExtension

TASK [arp_agent : Start the Agent service]
changed: [192.168.0.24]

TASK [arp_agent : Verify service is running]
ok: [192.168.0.24]

TASK [arp_agent : Display service status]
ok: [192.168.0.24] =>
    Status    DisplayName                    StartType
    ------    -----------                    ---------
    Running   Windows Network Extension...   Auto
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

### Issue 1: Deployment times out during Npcap installation

**Cause:** Npcap installer takes longer than expected on slow systems.

**Solution:**
1. Increase timeout in playbook (if needed):
   - Change `Start-Sleep -Seconds 15` to higher value in install task
   - Change 60-second wait loop to 120 seconds
2. Check Windows Event Viewer for installer errors
3. Manually verify installation on target:
   ```powershell
   Test-Path "HKLM:\Software\Npcap"
   Test-Path "C:\Program Files\Npcap"
   ```

**If It Still Stalls:**

1. Manually verify installation on target:
   ```powershell
   Test-Path "HKLM:\Software\Npcap"
   Test-Path "C:\Program Files\Npcap"
   ```

2. Check Windows Event Viewer for installer errors:
   ```powershell
   eventvwr  # -> Windows Logs -> System
   ```

3. Verify file permissions in deployment directory:
   ```powershell
   ls -la C:\ProgramData\WinNetExt\
   ```

---

### Issue 3: "No network interfaces found" after deployment

**Cause:** Npcap installed but network adapters not detected.

**Solution:**
1. Verify Npcap is properly set to WinPcap-compatible mode:
   ```powershell
   Get-ItemProperty "HKLM:\Software\Npcap" | fl *
   ```
2. Restart the WinNetExtension service:
   ```powershell
   Restart-Service -Name "WinNetExtension" -Force
   ```
3. As last resort, manually reinstall Npcap:
   ```powershell
   cd C:\ProgramData\WinNetExt
   .\npcap-1.87.exe /S /winpcap_mode=yes /loopback_support=yes
   # Wait 15 seconds
   Restart-Service -Name "WinNetExtension" -Force
   ```

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

### Issue 5 (OLD/FIXED): "Cannot find any service with service name 'npcap'"

**Status:** ✅ FIXED in v1.1

**What was wrong:** Playbook tried to check for a service named "npcap" which doesn't exist.

**How it's fixed:** Now using registry check instead (`HKLM:\Software\Npcap`).

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

### v1.1.3 (March 28, 2026 - Installation & Service Fixes)

**Problems Addressed:**
1. ❌ Npcap installer exits code 0 but files/registry never actually appear
2. ❌ Service marked for deletion error (0x430) blocks new service creation
3. ❌ Insufficient verification logging to diagnose installer failures

**Root Causes:**
1. **Npcap**: `-Verb RunAs` in WinRM context doesn't work reliably. Added diagnostics to see user context and admin status during execution.
2. **Service**: Previous failed deployments leave service markers in deletion state. NSSM remove doesn't force cleanup. Must use `sc.exe delete` with force-stop.
3. **Timing**: Service needs registration window before next task attempts access.

**Solutions:**

*Npcap Installation (v1.1.3)*:
- Removed `-Verb RunAs` (doesn't work in WinRM)
- Added user/admin context diagnostics
- Added 5-second post-install wait before verification
- Clearer exit code logging

```powershell
# Diagnostics added:
Write-Output "[*] Current user: $([System.Security.Principal.WindowsIdentity]::GetCurrent().Name)"
Write-Output "[*] Is admin: $([System.Security.Principal.WindowsPrincipal]::new(...).IsInRole('Administrator'))"

# Execution (simpler, no -Verb RunAs):
$process = Start-Process -FilePath $npcapInstaller `
  -ArgumentList "/S /winpcap_mode=yes /loopback_support=yes" `
  -PassThru -Wait -NoNewWindow -ErrorAction Stop
```

*Service Cleanup (NEW - v1.1.3)*:
- Added "Clean up any existing agent service" task before installation
- Forces service stop with `-Force`
- Uses `sc.exe delete` with immediate cleanup
- Waits 5 seconds for Windows to fully process deletion
- Verifies service no longer exists before attempting new install

```powershell
# Force cleanup any marker:
Stop-Service -Name $service -Force -ErrorAction SilentlyContinue
& cmd.exe /c "sc.exe delete `"$service`""
Start-Sleep -Seconds 5
```

*Service Registration Window (NEW - v1.1.3)*:
- Added "Wait for service registration" task after NSSM install
- Waits 3 seconds for Windows registry to update
- Verifies service is visible before next tasks
- Prevents "service not found" errors

**Files Modified:**
- `roles/arp_agent/tasks/main.yml`:
  - Lines 40-76: Simplified Npcap install (removed -Verb RunAs, added diagnostics)
  - Lines 168-190: NEW task to clean up existing service before install
  - Lines 229-240: NEW task to wait for service registration

**Expected Output (Success):**
```
TASK [arp_agent : Clean up any existing agent service] *** ok: [192.168.0.24]
[*] Checking for existing service: Windows Network Extension Service (WinNetExtension)
[+] No existing service found - proceeding with clean install

TASK [arp_agent : Install Agent as a Windows Service via NSSM] *** ok: [192.168.0.24]
[*] Installing service: Windows Network Extension Service (WinNetExtension)
[*] Running NSSM install...
[*] Configuring service properties...
[+] Service installed successfully: Windows Network Extension Service (WinNetExtension)

TASK [arp_agent : Wait for service registration] *** ok: [192.168.0.24]
[*] Waiting for service registration...
[+] Service registered and visible: Windows Network Extension Service (WinNetExtension)
    Status: Stopped
    StartType: Automatic

TASK [arp_agent : Start the Agent service] *** ok: [192.168.0.24]
... service started
```

**If Npcap Still Fails to Install:**
- Check installed Npcap manually: `Test-Path 'C:\Program Files\Npcap'`
- Check registry: `Test-Path 'HKLM:\Software\Npcap'`
- Try manual install: `C:\ProgramData\WinNetExt\npcap-1.87.exe /S /winpcap_mode=yes /loopback_support=yes`
- May require system reboot if previous Npcap version present
- Verify WinRM session has LocalSystem admin context

**If Service Still Shows "Marked for Deletion":**
- Manual cleanup: `sc.exe delete "Windows Network Extension Service (WinNetExtension)"`
- Wait 10 seconds: `Start-Sleep -Seconds 10`
- Reboot system if deletion persists

### v1.1.2 (March 28, 2026 - Installation Reliability)

**Problem Addressed:**
- ❌ Npcap installer exited code 0 but didn't actually install
- ❌ Registry check failed immediately after installation
- ❌ Agent service failed to start (no Npcap driver available)
- ❌ Insufficient wait time between installer exit and verification

**Root Causes:**
1. Only 5 seconds wait before checking registry (installer still completing)
2. No explicit admin context on installer execution
3. No retry logic if first verification failed
4. Generic error messages without troubleshooting guidance

**The Fix (v1.1.2):**
```powershell
# 1. Added explicit admin context
$process = Start-Process -FilePath $npcapInstaller `
  -ArgumentList "/S /winpcap_mode=yes /loopback_support=yes" `
  -PassThru -Wait -Verb RunAs -ErrorAction Stop  # <-- -Verb RunAs added

# 2. Increased wait: 30 seconds initial + 30 seconds retry (60 total)
Start-Sleep -Seconds 30

# 3. Two-stage verification with retry
# Verify attempt 1 (after 30 sec)
# If failed, sleep 30 more seconds
# Verify attempt 2 (after 60 sec total)
```

**Files Modified:**
- `roles/arp_agent/tasks/main.yml` - Lines 40-140: Complete install/verify sequence rewrite
  - Added `-Verb RunAs` for admin context
  - Increased wait to 30 seconds
  - Added second verification task with retry
  - Added detailed logging and error messages

**Expected Output (Success):**
```
TASK [arp_agent : Install Npcap] *** ok: [192.168.0.24]
[*] Starting Npcap installation...
[*] Installer exists: True
[*] Npcap installer exited with code: 0
[+] Npcap installation completed (exit code: 0)

TASK [arp_agent : Verify Npcap was installed (Attempt 1)] *** ok: [192.168.0.24]
✓ Npcap verified via registry

TASK [arp_agent : Display Npcap verification result] *** ok: [192.168.0.24]
Npcap installation succeeded
```

**If Verification Still Fails:**
The playbook will output detailed troubleshooting:
```
✗ Npcap still not detected after 60 second wait
[!] Possible causes:
    1. Installer requires interactive mode
    2. System pending reboot
    3. UAC or permission issues

Manual verification commands:
  Test-Path 'HKLM:\Software\Npcap'
  Test-Path 'C:\Program Files\Npcap'
  Get-ChildItem 'C:\Program Files' | grep -i npcap
```

### v1.1.1 (March 28, 2026 - Patch)

**Critical Bug Fix:**
- ✅ **Fixed Npcap installation hang** — Playbook stalled indefinitely during installer execution
- ✅ Changed from PowerShell `&` operator to `Start-Process -Wait` for proper process completion
- ✅ Added proper error handling and exit code validation (code 3010 = reboot needed)
- ✅ Added 10-minute timeout to prevent infinite hangs
- ✅ Simplified post-installation verification

**What Was Wrong:**
The previous implementation used `& $installer /S /flags` which doesn't properly wait for the executable to exit. The task would continue immediately after spawning the process, before installation actually completed.

**The Fix:**
```powershell
$process = Start-Process -FilePath $installer `
  -ArgumentList "/S /winpcap_mode=yes /loopback_support=yes" `
  -PassThru `
  -Wait `  # <-- This properly waits for completion
  -ErrorAction Stop
```

**Files Modified:**
- `roles/arp_agent/tasks/main.yml` - Fixed Npcap installation logic with `Start-Process -Wait`
- `DEPLOYMENT_TROUBLESHOOTING.md` - Added troubleshooting steps for timeout issues

### v1.1 (March 28, 2026)

**Enhancements:**
- ✅ Fixed incorrect Npcap detection (was checking for non-existent service)
- ✅ **Automated Npcap installation** — playbook now installs Npcap with WinPcap API-compatible mode
- ✅ Npcap installer (npcap-1.87.exe) included in repository
- ✅ Added pre-flight Npcap verification in agent
- ✅ Enhanced logging with structured messages
- ✅ Added `-debug` flag for troubleshooting
- ✅ Improved error messages with actionable solutions
- ✅ Better playbook error handling and cleanup
- ✅ Network interface enumeration on startup
- ✅ Automatic detection: skips Npcap installation if already present

**Files Modified:**
- `roles/arp_agent/tasks/main.yml` - Automated Npcap installation, registry-based detection
- `cmd/agent/main.go` - Enhanced diagnostics & validation
- `cmd/arpshell/main.go` - Better error reporting & debugging
- `README.md` - Updated deployment instructions
- `DEPLOYMENT_TROUBLESHOOTING.md` - Comprehensive guide

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
