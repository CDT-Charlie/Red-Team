# Realm Ansible Auto-Deployer

Deploys Realm (Imix) agents across all 9 competition VMs in parallel.

## Target VMs
| Group   | Count | OS                    | Persistence      |
|---------|----|--------------------------|------------------|
| debian  | 3  | Debian 13 Trixie         | systemd service  |
| ubuntu  | 3  | Ubuntu 24.04 Desktop     | systemd service  |
| windows | 3  | Windows Server 2022      | Scheduled Task   |

## Project Structure
```
realm-ansible/
├── ansible.cfg                  # Ansible settings (forks=9 for parallel)
├── site.yml                     # Master deploy playbook
├── verify.yml                   # Confirms all 9 beacons in Tavern
├── cleanup.yml                  # Removes agents (between rounds)
├── run.sh                       # One-shot: build → stage → deploy → verify
├── inventory/
│   └── hosts.ini                # VM IPs grouped by OS
├── group_vars/
│   ├── all.yml                  # Shared vars (Tavern IP, ports)
│   ├── linux/vars.yml           # SSH creds + Linux agent config
│   └── windows/vars.yml         # WinRM creds + Windows agent config
└── roles/
    ├── realm-linux/tasks/       # Download + systemd persistence
    └── realm-windows/tasks/     # Download + scheduled task persistence
```

## Quick Start

### 1. Install dependencies
```bash
pip install ansible pywinrm --break-system-packages
ansible-galaxy collection install ansible.windows community.windows
```

### 2. Configure your environment
Edit these files with real IPs and credentials:
- `group_vars/all.yml`       → Set `tavern_ip`
- `group_vars/linux/vars.yml`  → Set SSH user/password
- `group_vars/windows/vars.yml` → Set Administrator password
- `inventory/hosts.ini`      → Set VM IPs

### 3. Enable WinRM on Windows VMs (run once per VM)
```powershell
# Run on each Windows Server 2022 VM
winrm quickconfig -q
winrm set winrm/config/service/auth '@{Basic="true"}'
winrm set winrm/config/service '@{AllowUnencrypted="true"}'
Set-Item WSMan:\localhost\Service\Auth\NTLM -Value $true
New-NetFirewallRule -Name "WinRM-HTTP" -DisplayName "WinRM HTTP" `
    -Protocol TCP -LocalPort 5985 -Action Allow
```

### 4. Run everything
```bash
chmod +x run.sh
./run.sh
```

### Or run steps individually
```bash
# Deploy only
ansible-playbook site.yml

# Deploy to specific group only
ansible-playbook site.yml --limit debian
ansible-playbook site.yml --limit windows

# Verify beacons
ansible-playbook verify.yml

# Clean up all agents
ansible-playbook cleanup.yml

# Dry run (no changes)
ansible-playbook site.yml --check
```

## Troubleshooting

| Problem | Fix |
|---|---|
| WinRM connection refused | Run WinRM quickconfig on Windows VM |
| SSH auth failed | Check `group_vars/linux/vars.yml` creds |
| Agent not beaconing | Verify Tavern IP and port 80 is open |
| Binary download fails | Confirm HTTP staging server is running |
| Only N/9 beacons show | Re-run `ansible-playbook site.yml --limit <failed_host>` |
