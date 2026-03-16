# Agent Deploy — Realm · Sliver · Caldera

One Ansible command to:
1. Build Realm (Imix) Linux + Windows agent binaries from source
2. Clone + start Caldera C2 server
3. Download + start Sliver C2 server + generate implants
4. Start a staging HTTP server serving all binaries
5. Deploy all 3 agents to every victim machine (18 VMs across 2 Blue Teams)

| Agent   | Transport | Linux persistence | Windows persistence |
|---------|-----------|-------------------|---------------------|
| Realm   | HTTP/S    | systemd service   | Scheduled Task      |
| Sliver  | mTLS      | systemd service   | Scheduled Task      |
| Caldera | HTTP      | systemd service   | Scheduled Task      |

**Targets:** Windows Server 2022 · Debian 13 · Ubuntu Desktop 24.04 · Fedora

---

## Quick Start — One Command

```bash
cd ~/Red-Team/OST-Deployment/agent-deploy
ansible-playbook run-all.yml
```

This brings up all 3 C2 servers on your attacker box, then pushes agents to all
18 victim VMs. Every victim ends up with 3 active agents.

---

## Setup (edit 1 file before running)

### `inventory.yml` — the only file you need to touch

**1. Set your attacker box IP:**
```yaml
attacker_ip: "192.168.0.11"    # ← your C2/attacker machine IP
```
All C2 callback addresses and the staging server URL derive from this single value.

**2. Set shared credentials:**
```yaml
deploy_user:     "administrator"
deploy_password: "Password123!"
```

**3. Set victim IPs:**
```yaml
bt1_windows:
  hosts:
    bt1-armory:   { ansible_host: 10.1.1.1 }
    ...
```
Add/remove hosts per group as needed.

---

## Playbook Reference

| Playbook | What it does |
|----------|-------------|
| `run-all.yml` | **Full pipeline** — C2 setup + agent deployment (start here) |
| `setup-c2.yml` | C2 servers + staging only (no agent deployment) |
| `site.yml` | Agent deployment only (C2 must already be running) |
| `deploy-windows.yml` | All agents → Windows (BT1 + BT2) |
| `deploy-debian.yml` | All agents → Debian (BT1 + BT2) |
| `deploy-ubuntu.yml` | All agents → Ubuntu (BT1 + BT2) |
| `deploy-realm.yml` | Realm agent → all victims |
| `deploy-sliver.yml` | Sliver agent → all victims |
| `deploy-caldera.yml` | Caldera agent → all victims |
| `cleanup.yml` | Remove all agents from all victims |

**Targeting examples:**
```bash
# All agents to Blue Team 1 only
ansible-playbook site.yml --limit bt1

# Sliver only to Windows
ansible-playbook deploy-sliver.yml --limit windows

# All agents to a single host
ansible-playbook site.yml --limit bt2-armory
```

---

## Inventory Groups

| Group | Contains |
|-------|----------|
| `windows` | All Windows VMs (BT1 + BT2) |
| `debian` | All Debian VMs (BT1 + BT2) |
| `ubuntu` | All Ubuntu VMs (BT1 + BT2) |
| `fedora` | All Fedora VMs (BT1 + BT2) |
| `linux` | All Linux VMs (debian + ubuntu + fedora) |
| `bt1` | All BT1 VMs (all OS types) |
| `bt2` | All BT2 VMs (all OS types) |
| `victims` | All victim VMs |
| `c2` | Attacker box (runs C2 servers locally) |

---

## Directory Structure

```
agent-deploy/
├── inventory.yml           ← EDIT THIS: attacker IP + victim IPs + creds
├── group_vars/
│   ├── windows.yml         ← WinRM connection settings (no edits needed)
│   └── linux.yml           ← SSH + sudo settings (no edits needed)
├── ansible.cfg
│
├── run-all.yml             ← Full pipeline (C2 setup + agent deploy)
├── setup-c2.yml            ← C2 servers + staging only
├── site.yml                ← Agent deploy only (all victims)
├── deploy-windows.yml
├── deploy-debian.yml
├── deploy-ubuntu.yml
├── deploy-realm.yml
├── deploy-sliver.yml
├── deploy-caldera.yml
├── cleanup.yml
│
└── roles/
    ├── realm-server/       ← Builds Imix from source (Rust + cross-compile)
    ├── sliver-server/      ← Installs Sliver, starts service, generates implants
    ├── caldera-server/     ← Clones Caldera, starts service
    ├── staging-server/     ← Starts HTTP server serving agent binaries
    ├── realm/              ← Deploys Realm agent to victims
    ├── sliver/             ← Deploys Sliver implant to victims
    └── caldera/            ← Deploys Caldera Sandcat to victims
```

---

## Requirements

```bash
pip install ansible pywinrm
ansible-galaxy collection install ansible.windows community.windows
```

**On the attacker box (automatic via run-all.yml):**
- Rust + cargo (installed by realm-server role)
- gcc-mingw-w64 (Windows cross-compile)
- git, python3, python3-pip, python3-venv
- expect (for Sliver implant generation)

**Windows victims must have WinRM enabled on port 5985:**
```powershell
winrm quickconfig -q
winrm set winrm/config/service/auth '@{Basic="true"}'
Set-Item WSMan:\localhost\Service\AllowUnencrypted -Value $true
```
