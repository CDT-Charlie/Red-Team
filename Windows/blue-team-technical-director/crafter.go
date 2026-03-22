// The Go "Packet Crafter"
// This script demonstrates how to take a 4-character string (like "SHUT" or "INIT"), pack it into the SourceProtAddress field of an ARP frame, and broadcast it over the wire.
package main

import (
	"log"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

func main() {
	// 1. Define the network interface (Update this to your actual interface, e.g., "eth0")
	device := "eth0"
	handle, err := pcap.OpenLive(device, 1024, false, pcap.BlockForever)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	// 2. The data we want to "smuggle" (Must be 4 bytes/chars)
	payload := "HELO" 
	payloadBytes := []byte(payload)

	// 3. Create the Ethernet Layer
	// We use a broadcast MAC so every machine on the segment "sees" it.
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeARP,
	}

	// 4. Create the ARP Layer (The "Payload")
	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest, // Masking as a request
		SourceHwAddress:   []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		
		// ENCODING STEP: We put our string bytes directly into the Source IP field
		SourceProtAddress: payloadBytes, 
		
		TargetHwAddress:   []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		TargetProtAddress: []byte{0x00, 0x00, 0x00, 0x00},
	}

	// 5. Serialize and Send
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	gopacket.SerializeLayers(buffer, opts, eth, arp)
	
	if err := handle.WritePacketData(buffer.Bytes()); err != nil {
		log.Fatal(err)
	}

	log.Printf("Sent ARP packet with encoded payload: %s", payload)
}