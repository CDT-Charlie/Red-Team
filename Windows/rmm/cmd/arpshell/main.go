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
	debug := flag.Bool("debug", false, "Enable debug output")
	flag.Parse()

	log.SetFlags(log.Ltime | log.Lshortfile)
	fmt.Println("=== ARP RMM Admin Shell v1.1 ===")

	// List available interfaces
	devices, err := pcap.FindAllDevs()
	if err != nil {
		log.Fatalf("Failed to enumerate network interfaces: %v", err)
	}
	if len(devices) == 0 {
		log.Fatal("No network interfaces found. Ensure libpcap is installed.")
	}

	fmt.Println("[+] Available network interfaces:")
	for i, d := range devices {
		fmt.Printf("    [%d] %s", i, d.Name)
		if d.Description != "" {
			fmt.Printf(" (%s)", d.Description)
		}
		fmt.Println()
	}

	fmt.Printf("\n[*] Opening interface: %s\n", *iface)
	handle, err := craft.OpenHandle(*iface, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL] Failed to open interface %s: %v\n", *iface, err)
		fmt.Fprintf(os.Stderr, "[FATAL] Troubleshooting:\n")
		fmt.Fprintf(os.Stderr, "  1. Verify interface name (use: ip link show)\n")
		fmt.Fprintf(os.Stderr, "  2. Ensure you have root/sudo privileges\n")
		fmt.Fprintf(os.Stderr, "  3. Verify libpcap is installed: apt-get install libpcap-dev\n")
		os.Exit(1)
	}
	defer handle.Close()

	if err := handle.SetBPFFilter("arp"); err != nil {
		log.Fatalf("[FATAL] Failed to set BPF filter: %v", err)
	}

	reader := bufio.NewReader(os.Stdin)
	currentSeq := 0

	fmt.Println("=== [ Layer 2 ARP-Shell Ready ] ===")
	fmt.Printf("[*] Interface: %s\n", *iface)
	fmt.Printf("[*] MVP Mode: %v\n", *mvp)
	fmt.Printf("[*] PSK: [configured]\n")
	if *mvp {
		fmt.Printf("[*] Target MAC: %s\n", *targetMAC)
	}
	fmt.Println("[*] Type 'quit' to exit\n")

	for {
		fmt.Print("ARP-Admin> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "exit" || input == "quit" {
			fmt.Println("[*] Exiting ARP Shell.")
			break
		}
		if input == "" {
			continue
		}

		// Validate input
		if len(input) > 512 {
			fmt.Println("[!] Command too long (max 512 bytes)")
			continue
		}

		// --- SEND state ---
		fragments := fragment.FragmentCommand(input)
		fmt.Printf("[TX] Fragmenting command into %d ARP packet(s)\n", len(fragments))

		sendErrors := 0
		for i, frag := range fragments {
			var srcMAC net.HardwareAddr
			if *mvp {
				srcMAC = net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
			} else {
				srcMAC = mac.GenerateHoppedMAC(*psk, currentSeq)
			}

			if err := craft.SendARPRequest(handle, srcMAC, frag); err != nil {
				fmt.Printf("[!] Fragment %d/%d send error: %v\n", i+1, len(fragments), err)
				sendErrors++
			} else if *debug {
				fmt.Printf("[DEBUG] Sent fragment %d/%d from MAC %s\n", i+1, len(fragments), srcMAC.String())
			}
			currentSeq++
			time.Sleep(10 * time.Millisecond)
		}

		if sendErrors > 0 {
			fmt.Printf("[!] %d/%d fragments failed to send\n", sendErrors, len(fragments))
			continue
		}

		// --- LISTEN state ---
		fmt.Println("[*] Waiting for response (max 30 seconds)...")
		response := listenForResponse(handle, *psk, *mvp, *targetMAC, &currentSeq, *debug)

		// --- DISPLAY state ---
		fmt.Printf("\n--- RESPONSE ---\n%s\n----------------\n\n", response)
	}
}

// listenForResponse sniffs for ARP Reply packets from the agent and
// reassembles the fragmented response. In MVP mode it matches by a static
// target MAC; in hopping mode it validates against predicted MACs.
func listenForResponse(handle *pcap.Handle, psk string, mvpMode bool, targetMAC string, seq *int, debug bool) string {
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	respBuf := fragment.NewCommandBuffer()
	deadline := time.After(responseTimeout)
	fragmentCount := 0

	for {
		select {
		case <-deadline:
			if fragmentCount == 0 {
				return "[TIMEOUT] No response from agent within 30 seconds.\n" +
					"Verify:\n" +
					"  1. Agent is running on Windows server\n" +
					"  2. Agent MAC matches or Npcap MAC hopping is configured\n" +
					"  3. Network connectivity between admin and agent\n" +
					"  4. PSK (pre-shared key) is identical on both sides"
			}
			return fmt.Sprintf("[TIMEOUT] Received %d fragments but response incomplete after 30s", fragmentCount)

		case packet, ok := <-packetSource.Packets():
			if !ok {
				return "[ERROR] Packet source closed unexpectedly"
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
					if debug {
						fmt.Printf("[DEBUG] MAC mismatch: got %s, expected %s\n", srcMAC, expected)
					}
					continue
				}
			} else if srcMAC != targetMAC {
				if debug {
					fmt.Printf("[DEBUG] Rejected SPA from %s (target: %s)\n", srcMAC, targetMAC)
				}
				continue
			}

			spa := arp.SourceProtAddress
			if len(spa) < 4 {
				fmt.Printf("[WARN] Invalid SPA length: %d\n", len(spa))
				continue
			}

			seqID := int(spa[0] & fragment.SeqIDMask)
			moreFragments := (spa[0] & 0x80) != 0
			fragmentCount++

			if debug {
				fmt.Printf("[DEBUG] RX fragment seq=%d more=%v from MAC %s\n", seqID, moreFragments, srcMAC)
			}

			*seq++

			if respBuf.ProcessFragment(spa) {
				result := respBuf.Reassemble()
				fmt.Printf("[RX] Response complete (%d fragments received)\n", fragmentCount)
				return result
			}
		}
	}
}
