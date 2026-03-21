import subprocess
import os
from pathlib import Path

def run_cmd(cmd):
    try:
        result = subprocess.run(
            cmd,
            shell=True,
            capture_output=True,
            text=True,
            timeout=15
        )
        return result.stdout.strip()
    except Exception:
        return ""

def print_section(title):
    print(f"\n==== {title} ====")

def current_user():
    print_section("Current User")
    print(run_cmd("whoami"))
    print(run_cmd("whoami /priv"))

def always_install_elevated():
    print_section("AlwaysInstallElevated")
    hkcu = run_cmd(r'reg query HKCU\Software\Policies\Microsoft\Windows\Installer /v AlwaysInstallElevated')
    hklm = run_cmd(r'reg query HKLM\Software\Policies\Microsoft\Windows\Installer /v AlwaysInstallElevated')

    if "0x1" in hkcu and "0x1" in hklm:
        print("[!] AlwaysInstallElevated appears to be enabled in HKCU and HKLM")
    else:
        print("[-] Not enabled")

def unquoted_service_paths():
    print_section("Unquoted Service Paths")
    output = run_cmd('wmic service get name,pathname,startmode')
    findings = []

    for line in output.splitlines():
        line = line.strip()
        if not line or line.lower().startswith("name"):
            continue
        if "Auto" not in line:
            continue
        if '"' in line:
            continue
        if "C:\\" in line and " " in line:
            findings.append(line)

    if findings:
        for item in findings:
            print(f"[!] {item}")
    else:
        print("[-] No obvious unquoted auto-start service paths found")

def scheduled_tasks():
    print_section("Scheduled Tasks")
    output = run_cmd(r'schtasks /query /fo LIST /v')
    lines_to_keep = []

    interesting = ("TaskName:", "Author:", "Run As User:", "Task To Run:")
    for line in output.splitlines():
        if line.startswith(interesting):
            lines_to_keep.append(line)

    if lines_to_keep:
        for line in lines_to_keep[:200]:
            print(line)
    else:
        print("[-] No scheduled task output returned")

def path_review():
    print_section("PATH Review")
    path_value = os.environ.get("PATH", "")
    for entry in path_value.split(";"):
        entry = entry.strip()
        if not entry:
            continue
        p = Path(entry)
        if p.exists():
            print(f"[+] {entry}")
        else:
            print(f"[-] Missing path entry: {entry}")

def service_overview():
    print_section("Running Services Overview")
    output = run_cmd("sc query state= all")
    lines = []
    keep_prefixes = ("SERVICE_NAME:", "DISPLAY_NAME:", "STATE")
    for line in output.splitlines():
        stripped = line.strip()
        if stripped.startswith(keep_prefixes):
            lines.append(stripped)

    if lines:
        for line in lines[:200]:
            print(line)
    else:
        print("[-] No service output returned")

def main():
    print("WinPrivScan - local read-only enumeration")
    current_user()
    always_install_elevated()
    unquoted_service_paths()
    scheduled_tasks()
    path_review()
    service_overview()

if __name__ == "__main__":
    main()
