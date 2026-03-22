package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"sort"
	"strings"
	"time"

	"rmm/shared"
)

type Collector struct {
	SnapshotPath string
}

func NewCollector() *Collector {
	return &Collector{SnapshotPath: shared.SnapshotPath()}
}

func (c *Collector) Collect(ctx context.Context) (*HostSnapshot, error) {
	host, _ := os.Hostname()
	currentUser, _ := user.Current()

	snapshot := &HostSnapshot{
		Timestamp:    time.Now().UTC(),
		Hostname:     host,
		Username:     "",
		Platform:     runtime.GOOS,
		Architecture: runtime.GOARCH,
	}
	if currentUser != nil {
		snapshot.Username = currentUser.Username
	}

	hostIdentity, err := collectHostIdentity(ctx)
	if err != nil {
		snapshot.Notes = append(snapshot.Notes, fmt.Sprintf("host identity collection error: %v", err))
	} else {
		snapshot.Domain = hostIdentity.Domain
		snapshot.OSName = hostIdentity.OSName
		snapshot.OSVersion = hostIdentity.OSVersion
		snapshot.LastBootUTC = hostIdentity.LastBootUTC
		if hostIdentity.Hostname != "" {
			snapshot.Hostname = hostIdentity.Hostname
		}
	}

	processes, err := collectProcesses(ctx)
	if err != nil {
		snapshot.Notes = append(snapshot.Notes, fmt.Sprintf("process collection error: %v", err))
	} else {
		snapshot.Processes = processes
	}

	services, err := collectServices(ctx)
	if err != nil {
		snapshot.Notes = append(snapshot.Notes, fmt.Sprintf("service collection error: %v", err))
	} else {
		snapshot.Services = services
	}

	network, err := collectNetwork(ctx)
	if err != nil {
		snapshot.Notes = append(snapshot.Notes, fmt.Sprintf("network collection error: %v", err))
	} else {
		snapshot.Network = network
	}

	arp, err := collectARP(ctx)
	if err != nil {
		snapshot.Notes = append(snapshot.Notes, fmt.Sprintf("arp collection error: %v", err))
	} else {
		snapshot.ARP = arp
	}

	return snapshot, nil
}

func (c *Collector) CollectAndPersist(ctx context.Context) (*HostSnapshot, error) {
	snapshot, err := c.Collect(ctx)
	if err != nil {
		return nil, err
	}

	if err := shared.WriteJSONLine(c.SnapshotPath, snapshot); err != nil {
		return nil, err
	}

	return snapshot, nil
}

func runPowerShellJSON(ctx context.Context, script string, v any) error {
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", script)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("powershell failed: %w", err)
	}
	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		out = []byte("[]")
	}
	return json.Unmarshal(out, v)
}

type hostIdentity struct {
	Hostname    string
	Domain      string
	OSName      string
	OSVersion   string
	LastBootUTC string
}

func collectHostIdentity(ctx context.Context) (hostIdentity, error) {
	type psHost struct {
		CSName            string `json:"CSName"`
		Domain            string `json:"Domain"`
		Caption           string `json:"Caption"`
		Version           string `json:"Version"`
		LastBootUpTimeUTC string `json:"LastBootUpTimeUtc"`
	}

	var rows []psHost
	script := "@(Get-CimInstance Win32_OperatingSystem | Select-Object @{Name='CSName';Expression={$_.CSName}}, @{Name='Domain';Expression={(Get-CimInstance Win32_ComputerSystem).Domain}}, Caption, Version, @{Name='LastBootUpTimeUtc';Expression={$_.LastBootUpTime.ToUniversalTime().ToString('o')}}) | ConvertTo-Json -Depth 3 -Compress"
	if err := runPowerShellJSON(ctx, script, &rows); err != nil {
		return hostIdentity{}, err
	}
	if len(rows) == 0 {
		return hostIdentity{}, nil
	}

	return hostIdentity{
		Hostname:    rows[0].CSName,
		Domain:      rows[0].Domain,
		OSName:      rows[0].Caption,
		OSVersion:   rows[0].Version,
		LastBootUTC: rows[0].LastBootUpTimeUTC,
	}, nil
}

func collectProcesses(ctx context.Context) ([]ProcessRecord, error) {
	type psProcess struct {
		Name           string `json:"Name"`
		ProcessId      int    `json:"ProcessId"`
		ExecutablePath string `json:"ExecutablePath"`
	}

	var rows []psProcess
	script := "@(Get-CimInstance Win32_Process | Select-Object Name,ProcessId,ExecutablePath) | ConvertTo-Json -Depth 2 -Compress"
	if err := runPowerShellJSON(ctx, script, &rows); err != nil {
		return nil, err
	}

	records := make([]ProcessRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, ProcessRecord{
			Name: row.Name,
			PID:  fmt.Sprintf("%d", row.ProcessId),
			Path: row.ExecutablePath,
		})
	}
	return records, nil
}

func collectServices(ctx context.Context) ([]ServiceRecord, error) {
	type psService struct {
		Name        string `json:"Name"`
		DisplayName string `json:"DisplayName"`
		State       string `json:"State"`
		StartMode   string `json:"StartMode"`
	}

	var rows []psService
	script := "@(Get-CimInstance Win32_Service | Select-Object Name,DisplayName,State,StartMode) | ConvertTo-Json -Depth 2 -Compress"
	if err := runPowerShellJSON(ctx, script, &rows); err != nil {
		return nil, err
	}

	records := make([]ServiceRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, ServiceRecord{
			Name:        row.Name,
			DisplayName: row.DisplayName,
			Status:      row.State,
			StartMode:   row.StartMode,
		})
	}
	return records, nil
}

func collectNetwork(ctx context.Context) (NetworkSnapshot, error) {
	type psAdapter struct {
		InterfaceAlias string `json:"InterfaceAlias"`
		MacAddress     string `json:"MacAddress"`
		Status         string `json:"Status"`
	}

	type psIP struct {
		InterfaceAlias string `json:"InterfaceAlias"`
		IPAddress      string `json:"IPAddress"`
	}

	type psGateway struct {
		NextHop string `json:"NextHop"`
	}

	type psDNS struct {
		ServerAddresses []string `json:"ServerAddresses"`
	}

	var adapters []psAdapter
	if err := runPowerShellJSON(ctx, "@(Get-NetAdapter | Select-Object InterfaceAlias,MacAddress,Status) | ConvertTo-Json -Depth 2 -Compress", &adapters); err != nil {
		return NetworkSnapshot{}, err
	}

	var ips []psIP
	if err := runPowerShellJSON(ctx, "@(Get-NetIPAddress -AddressFamily IPv4 | Select-Object InterfaceAlias,IPAddress) | ConvertTo-Json -Depth 2 -Compress", &ips); err != nil {
		return NetworkSnapshot{}, err
	}

	var gateways []psGateway
	_ = runPowerShellJSON(ctx, "@(Get-NetRoute -DestinationPrefix '0.0.0.0/0' | Select-Object NextHop) | ConvertTo-Json -Depth 2 -Compress", &gateways)

	var dnsRows []psDNS
	_ = runPowerShellJSON(ctx, "@(Get-DnsClientServerAddress -AddressFamily IPv4 | Select-Object ServerAddresses) | ConvertTo-Json -Depth 3 -Compress", &dnsRows)

	byAlias := make(map[string]*InterfaceRecord, len(adapters))
	for _, adapter := range adapters {
		byAlias[adapter.InterfaceAlias] = &InterfaceRecord{
			Name:   adapter.InterfaceAlias,
			MAC:    adapter.MacAddress,
			Status: adapter.Status,
			Up:     strings.EqualFold(adapter.Status, "Up"),
		}
	}
	for _, ip := range ips {
		if record, ok := byAlias[ip.InterfaceAlias]; ok && strings.TrimSpace(ip.IPAddress) != "" {
			record.IPv4 = append(record.IPv4, ip.IPAddress)
		}
	}

	interfaces := make([]InterfaceRecord, 0, len(byAlias))
	for _, record := range byAlias {
		interfaces = append(interfaces, *record)
	}
	sort.Slice(interfaces, func(i, j int) bool {
		return interfaces[i].Name < interfaces[j].Name
	})

	gatewayList := make([]string, 0, len(gateways))
	for _, gateway := range gateways {
		if nextHop := strings.TrimSpace(gateway.NextHop); nextHop != "" && nextHop != "0.0.0.0" {
			gatewayList = append(gatewayList, nextHop)
		}
	}
	gatewayList = uniqueSorted(gatewayList)

	dnsServers := make([]string, 0)
	for _, row := range dnsRows {
		dnsServers = append(dnsServers, row.ServerAddresses...)
	}

	return NetworkSnapshot{
		Interfaces:     interfaces,
		DefaultGateway: gatewayList,
		DNSServers:     uniqueSorted(dnsServers),
	}, nil
}

func collectARP(ctx context.Context) (ARPObservation, error) {
	type psNeighbor struct {
		IPAddress        string `json:"IPAddress"`
		LinkLayerAddress string `json:"LinkLayerAddress"`
		State            string `json:"State"`
		InterfaceAlias   string `json:"InterfaceAlias"`
	}

	var rows []psNeighbor
	script := "@(Get-NetNeighbor -AddressFamily IPv4 | Select-Object IPAddress,LinkLayerAddress,State,InterfaceAlias) | ConvertTo-Json -Depth 2 -Compress"
	if err := runPowerShellJSON(ctx, script, &rows); err != nil {
		return ARPObservation{}, err
	}

	observation := ARPObservation{
		Entries: make([]ARPEntry, 0, len(rows)),
	}
	ipToMACs := make(map[string]map[string]struct{})
	macToIPs := make(map[string]map[string]struct{})
	for _, row := range rows {
		entry := ARPEntry{
			Interface:  row.InterfaceAlias,
			IPAddress:  row.IPAddress,
			MACAddress: row.LinkLayerAddress,
			State:      row.State,
			Source:     "Get-NetNeighbor",
		}
		observation.Entries = append(observation.Entries, entry)

		ipKey := strings.TrimSpace(strings.ToLower(entry.IPAddress))
		macKey := strings.TrimSpace(strings.ToLower(entry.MACAddress))
		if ipKey != "" && macKey != "" {
			if _, ok := ipToMACs[ipKey]; !ok {
				ipToMACs[ipKey] = map[string]struct{}{}
			}
			ipToMACs[ipKey][macKey] = struct{}{}
			if _, ok := macToIPs[macKey]; !ok {
				macToIPs[macKey] = map[string]struct{}{}
			}
			macToIPs[macKey][ipKey] = struct{}{}
		}

		if isARPAnomaly(entry) {
			observation.Anomalies = append(observation.Anomalies, fmt.Sprintf("%s on %s state=%s mac=%s", entry.IPAddress, entry.Interface, entry.State, entry.MACAddress))
		}
	}

	for ipAddress, macs := range ipToMACs {
		if len(macs) > 1 {
			observation.Anomalies = append(observation.Anomalies, fmt.Sprintf("ip %s resolved to multiple MAC addresses: %s", ipAddress, strings.Join(sortedKeys(macs), ", ")))
		}
	}
	for macAddress, ips := range macToIPs {
		if len(ips) > 3 {
			observation.Anomalies = append(observation.Anomalies, fmt.Sprintf("mac %s resolved to many IP addresses: %s", macAddress, strings.Join(sortedKeys(ips), ", ")))
		}
	}
	observation.Anomalies = uniqueSorted(observation.Anomalies)

	return observation, nil
}

func isARPAnomaly(entry ARPEntry) bool {
	state := strings.ToLower(strings.TrimSpace(entry.State))
	if state == "incomplete" || state == "unreachable" || state == "invalid" {
		return true
	}

	mac := strings.ToLower(strings.TrimSpace(entry.MACAddress))
	if mac == "" || mac == "00-00-00-00-00-00" || mac == "ff-ff-ff-ff-ff-ff" {
		return true
	}

	return false
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	sort.Strings(result)
	return result
}

func sortedKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
