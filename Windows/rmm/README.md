# RMM Blue Team Monitor

`RMM` is a lab-only Windows monitoring agent for blue-team exercises. It collects host identity, process inventory, service inventory, network/interface state, and ARP observations, then writes JSONL snapshots and explicit audit logs to disk.

## What It Does

- Collects host, OS, domain, and last-boot details
- Inventories running processes and Windows services
- Captures interface, IPv4, gateway, DNS, and ARP neighbor state
- Flags simple ARP anomalies such as invalid entries, duplicate IP-to-MAC mappings, and suspicious broadcast values
- Writes file-backed audit logs and snapshot records under a visible lab directory

## Layout

- `cmd/rmm-agent` contains the Windows monitoring agent entrypoint
- `agent/` contains the collection loop and shutdown handling
- `telemetry/` contains snapshot models and Windows data collection helpers
- `shared/` contains lab guardrails and logging helpers
- `scripts/` contains explicit deployment helpers for lab use
- `deploy_rmm.yml` is an Ansible playbook for staging the agent

## Build

```bash
make agent
```

The default build output is `dist/rmm-agent.exe`.

For a local development build on the current platform:

```bash
make dev
```

## Deployment

The deployment helpers are intentionally visible and non-stealthy. They use the lab paths below by default:

- `C:\ProgramData\RMMBlueTeam`
- `C:\ProgramData\RMMBlueTeam\logs`
- `C:\ProgramData\RMMBlueTeam\STOP`

Set `RMM_DEMO_MODE=1` only inside the training sandbox. The agent refuses to run unless lab mode is explicitly enabled.

## Output

- Audit log: `C:\ProgramData\RMMBlueTeam\logs\rmm-agent.log`
- Deployment transcript: `C:\ProgramData\RMMBlueTeam\logs\deploy-*.log`
- Snapshot stream: `C:\ProgramData\RMMBlueTeam\snapshots.jsonl`
- Kill switch: create `C:\ProgramData\RMMBlueTeam\STOP` to stop future startup or collection

## Safety Notes

- The old command execution and ARP transport behavior has been removed from this module.
- Persistence is optional and explicit in the PowerShell deployment helper.
- The agent is designed for contained Windows lab systems, not covert operation.
