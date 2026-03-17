package implant

import (
	"fmt"
	"net"
	"strings"

	"github.com/google/gopacket/pcap"
)

// InterfaceInfo holds information about the active network interface
type InterfaceInfo struct {
	Name    string
	HardwareAddr string
	IP      string
	Device  *pcap.Handle
}

// FindActiveInterface detects the primary active network interface
// that will be used for ARP sniffing and broadcasting.
func FindActiveInterface() (*InterfaceInfo, error) {
	// Get all interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to list interfaces: %w", err)
	}

	// Get all pcap devices to map Windows names properly (\Device\NPF_...)
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, fmt.Errorf("failed to list pcap devices: %w", err)
	}

	for _, iface := range interfaces {
		// Skip loopback and non-active interfaces
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		// Skip interfaces without hardware address
		if len(iface.HardwareAddr) == 0 {
			continue
		}

		// Get IP address
		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}

		var ipAddr string
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				ipAddr = ipnet.IP.String()
				break
			}
		}

		if ipAddr == "" {
			continue
		}

		// Now find the matching device in pcap to get the \Device\NPF_ name
		pcapName := iface.Name
		for _, dev := range devices {
			for _, devAddr := range dev.Addresses {
				if devAddr.IP.String() == ipAddr {
					pcapName = dev.Name
					break
				}
			}
		}

		return &InterfaceInfo{
			Name:         pcapName,
			HardwareAddr: iface.HardwareAddr.String(),
			IP:           ipAddr,
		}, nil
	}

	return nil, fmt.Errorf("no suitable network interface found")
}

// GetDeviceHandle opens a pcap handle for the interface
func (ii *InterfaceInfo) GetDeviceHandle() (*pcap.Handle, error) {
	handle, err := pcap.OpenLive(ii.Name, 1600, true, pcap.BlockForever)
	if err != nil {
		return nil, fmt.Errorf("failed to open device %s: %w", ii.Name, err)
	}

	// Set BPF filter to only capture ARP traffic
	err = handle.SetBPFFilter("arp")
	if err != nil {
		handle.Close()
		return nil, fmt.Errorf("failed to set BPF filter: %w", err)
	}

	ii.Device = handle
	return handle, nil
}

// Close gracefully closes the device handle
func (ii *InterfaceInfo) Close() error {
	if ii.Device != nil {
		ii.Device.Close()
	}
	return nil
}

// ParseInterfaceName attempts to get interface name from device string
// Useful for cross-platform compatibility (eth0 on Linux, \Device\NPF_... on Windows)
func ParseInterfaceName(deviceName string) string {
	// On Windows, device names are like \Device\NPF_{UUID}
	// On Linux, they're like eth0, wlan0, etc.
	
	if strings.Contains(deviceName, "NPF") {
		// Windows format - extract the interface info
		return deviceName
	}
	
	return deviceName
}
