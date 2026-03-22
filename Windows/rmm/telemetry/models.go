package telemetry

import "time"

type HostSnapshot struct {
	Timestamp    time.Time       `json:"timestamp"`
	Hostname     string          `json:"hostname"`
	Username     string          `json:"username"`
	Platform     string          `json:"platform"`
	Architecture string          `json:"architecture"`
	Domain       string          `json:"domain,omitempty"`
	OSName       string          `json:"os_name,omitempty"`
	OSVersion    string          `json:"os_version,omitempty"`
	LastBootUTC  string          `json:"last_boot_utc,omitempty"`
	Processes    []ProcessRecord `json:"processes"`
	Services     []ServiceRecord `json:"services"`
	Network      NetworkSnapshot `json:"network"`
	ARP          ARPObservation  `json:"arp"`
	Notes        []string        `json:"notes,omitempty"`
}

type ProcessRecord struct {
	Name string `json:"name"`
	PID  string `json:"pid"`
	Path string `json:"path,omitempty"`
}

type ServiceRecord struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name,omitempty"`
	Status      string `json:"status,omitempty"`
	StartMode   string `json:"start_mode,omitempty"`
}

type NetworkSnapshot struct {
	Interfaces     []InterfaceRecord `json:"interfaces"`
	DefaultGateway []string          `json:"default_gateway,omitempty"`
	DNSServers     []string          `json:"dns_servers,omitempty"`
}

type InterfaceRecord struct {
	Name   string   `json:"name"`
	IPv4   []string `json:"ipv4,omitempty"`
	MAC    string   `json:"mac,omitempty"`
	Status string   `json:"status,omitempty"`
	Up     bool     `json:"up"`
}

type ARPObservation struct {
	Entries   []ARPEntry `json:"entries"`
	Anomalies []string   `json:"anomalies,omitempty"`
}

type ARPEntry struct {
	Interface  string `json:"interface,omitempty"`
	IPAddress  string `json:"ip_address"`
	MACAddress string `json:"mac_address,omitempty"`
	State      string `json:"state,omitempty"`
	Source     string `json:"source,omitempty"`
}
