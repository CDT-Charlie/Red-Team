# firewall-unblock

Linux red team tool that checks common host firewalls for **blocking** (DROP/REJECT) rules and removes them. Useful when defenders or automation add rules to block C2 or operator IPs.

## What it does

- **UFW** – Finds DENY/REJECT/DROP rules and deletes them (by rule number, high to low).
- **iptables** – Finds DROP/REJECT rules in INPUT, OUTPUT, and FORWARD and deletes them (by line number, reverse order).
- **nftables** – Finds rules with `drop` or `reject` in the ruleset and deletes them by handle.
- **firewalld** – Removes rich rules and direct rules that contain drop/reject, then reloads.

Only **blocking** rules are removed; ACCEPT and other rules are left as-is.

## Requirements

- Linux
- **root** (or sudo) to modify firewall state
- Bash

## Usage

**Run once (single pass):**

```bash
sudo ./firewall-unblock.sh once
```

**Run every 60 seconds (default):**

```bash
sudo ./firewall-unblock.sh
```

**Custom interval (e.g. 60 seconds):**

```bash
sudo INTERVAL=60 ./firewall-unblock.sh
```

**Log to a file:**

```bash
sudo LOG=/var/log/firewall-unblock.log ./firewall-unblock.sh
```

**Background (every minute):**

```bash
sudo nohup ./firewall-unblock.sh </dev/null >>/var/log/firewall-unblock.log 2>&1 &
```

## Run every minute via cron

```bash
sudo crontab -e
# Add:
* * * * * /path/to/firewall-unblock/firewall-unblock.sh once
```

## Run as a systemd service (every minute via timer)

See `firewall-unblock.service` and `firewall-unblock.timer` in this directory:

```bash
sudo cp firewall-unblock.service firewall-unblock.timer /etc/systemd/system/
sudo sed -i 's|/path/to/firewall-unblock|/opt/firewall-unblock|g' /etc/systemd/system/firewall-unblock.service
sudo systemctl daemon-reload
sudo systemctl enable --now firewall-unblock.timer
```

## Files

| File | Purpose |
|------|--------|
| `firewall-unblock.sh` | Main script; run `once` or in a loop (default 60s). |
| `README.md` | This file. |
| `firewall-unblock.service` | systemd unit (runs the script once). |
| `firewall-unblock.timer` | systemd timer (runs the service every minute). |

## OpSec notes

- Requires root; consider how the script is deployed and where logs go.
- Running in a loop or via cron/timer keeps re-removing blocking rules; defenders may notice repeated rule changes or a persistent process.
- Use only in authorized red team engagements.
