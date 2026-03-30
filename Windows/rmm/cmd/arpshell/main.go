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

	"arp-rmm/internal/fragment"
	"arp-rmm/internal/transport"
)

const responseTimeout = 30 * time.Second

func main() {
	target := flag.String("target", "192.168.0.24:9999", "Agent IP:Port (e.g. 192.168.0.24:9999)")
	psk := flag.String("psk", "S3cur3_Adm1n_K3y", "Pre-shared key (unused in UDP v2)")
	debug := flag.Bool("debug", false, "Enable debug output")
	flag.Parse()

	log.SetFlags(log.Ltime | log.Lshortfile)
	fmt.Println("=== UDP RMM Admin Shell v2.0 (Cloud-Ready) ===")
	fmt.Printf("[*] Target: %s\n", *target)
	fmt.Println("[*] Type 'quit' to exit\n")

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("UDP-Admin> ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "quit" || input == "exit" {
			fmt.Println("[*] Goodbye!")
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
		fmt.Printf("[TX] Fragmenting command into %d UDP packet(s)\n", len(fragments))
		
		if *debug {
			fmt.Printf("[DEBUG] Command: \"%s\" (%d bytes)\n", input, len(input))
		}

		// Parse target IP and port
		parts := strings.Split(*target, ":")
		if len(parts) != 2 {
			fmt.Printf("[!] Invalid target format. Use: IP:PORT (e.g. 192.168.0.24:9999)\n")
			continue
		}
		targetIP := parts[0]
		var targetPort int
		_, err := fmt.Sscanf(parts[1], "%d", &targetPort)
		if err != nil {
			fmt.Printf("[!] Invalid port: %s\n", parts[1])
			continue
		}

		sendErrors := 0
		for i, frag := range fragments {
			// Fragments are already properly formatted with control byte
			if err := transport.SendUDP(targetIP, targetPort, frag); err != nil {
				fmt.Printf("[!] Fragment %d/%d send error: %v\n", i+1, len(fragments), err)
				sendErrors++
			} else if *debug {
				fmt.Printf("[DEBUG] Sent fragment %d/%d (%d bytes)\n",
					i+1, len(fragments), len(frag))
			}
			time.Sleep(5 * time.Millisecond)
		}

		if sendErrors > 0 {
			fmt.Printf("[!] %d/%d fragments failed to send\n", sendErrors, len(fragments))
			continue
		}

		// --- LISTEN state ---
		fmt.Println("[*] Waiting for response (max 30 seconds)...")
		if *debug {
			fmt.Printf("[DEBUG] Listening on UDP for response from %s...\n", *target)
		}
		response := listenForUDPResponse(*psk, *debug)

		// --- DISPLAY state ---
		fmt.Printf("\n--- RESPONSE ---\n%s\n----------------\n\n", response)
	}
}

// listenForUDPResponse listens for UDP response packets and reassembles them
func listenForUDPResponse(psk string, debug bool) string {
	// Create a listener on any available UDP port
	listenerAddr, _ := net.ResolveUDPAddr("udp", ":0")
	listener, err := net.ListenUDP("udp", listenerAddr)
	if err != nil {
		return fmt.Sprintf("[ERROR] Failed to create UDP listener: %v", err)
	}
	defer listener.Close()

	if debug {
		fmt.Printf("[DEBUG] Listening on %s for response\n", listener.LocalAddr().String())
	}

	deadline := time.Now().Add(responseTimeout)
	listener.SetDeadline(deadline)

	respBuf := fragment.NewCommandBuffer()
	fragmentCount := 0
	packetCount := 0

	for {
		// Receive UDP packet
		buf := make([]byte, 4096)
		n, _, err := listener.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				if fragmentCount == 0 {
					return "[TIMEOUT] No response from agent within 30 seconds.\n" +
						"Verify:\n" +
						"  1. Agent is running on target\n" +
						"  2. Agent is listening on port 9999\n" +
						"  3. Network connectivity exists\n" +
						"  4. Firewall allows UDP:9999\n" +
						"  5. Target IP:port is correct\n"
				}
				return fmt.Sprintf("[TIMEOUT] Received %d fragments but incomplete\n", fragmentCount)
			}
			return fmt.Sprintf("[ERROR] Listen error: %v", err)
		}

		if n < 1 {
			continue
		}

		packetCount++
		data := buf[:n]

		if debug {
			fmt.Printf("[DEBUG] RX packet %d: len=%d\n", packetCount, len(data))
		}

		fragmentCount++
		
		// ProcessFragment returns true when the command is complete and ready to reassemble
		if respBuf.ProcessFragment(data) {
			result := respBuf.Reassemble()
			if debug {
				fmt.Printf("[DEBUG] Response complete (%d UDP packets)\n", packetCount)
			}
			return result
		}
	}
}

// DEPRECATED: Old ARP-based listener
// listenForResponse sniffs for ARP Reply packets from the agent and
// reassembles the fragmented response. In MVP mode it matches by a static
// target MAC; in hopping mode it validates against predicted MACs.
func listenForResponse(handle *pcap.Handle, psk string, mvpMode bool, targetMAC string, seq *int, debug bool) string {
	return "[DEPRECATED] Use UDP-based transport instead"
}
