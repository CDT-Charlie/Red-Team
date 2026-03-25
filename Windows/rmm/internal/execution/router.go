package execution

import "fmt"

// RouteCommand maps abbreviated smuggled commands to hardened PowerShell
// scripts. Direct pass-through of arbitrary input is never permitted.
func RouteCommand(input string) string {
	switch input {
	case "HELO":
		return "READY"

	case "who":
		return runPS("whoami /all")

	case "hostname":
		return runPS("hostname")

	case "net":
		return runPS("Get-NetIPAddress | Select-Object InterfaceAlias, IPAddress")

	case "RESTART_IIS":
		return runPS("Restart-Service W3SVC -Force")

	case "ST_SRV_MSSQL":
		return runPS("Start-Service 'MSSQLSERVER'")

	case "GET_SERVICES":
		return runPS("Get-Service | Where-Object {$_.Status -eq 'Running'} | Select-Object -First 10")

	case "DISK_USAGE":
		return runPS("Get-PSDrive C | Select-Object Used, Free")

	case "uptime":
		return runPS("(Get-CimInstance Win32_OperatingSystem).LastBootUpTime")

	case "arp":
		return runPS("Get-NetNeighbor | Select-Object IPAddress, LinkLayerAddress, State")

	default:
		return fmt.Sprintf("ERR: unknown command '%s'", input)
	}
}

func runPS(command string) string {
	output, err := ExecutePowerShell(command)
	if err != nil {
		return fmt.Sprintf("ERR: %v", err)
	}
	return output
}
