// To create a true "ARP Shell" experience, the application must behave like a state machine. It transitions from Command Entry (Admin typing) to Fragment Transmission (Sender) to Reverse-Hopping Sniffing (Receiver).

// Because the user is an IT admin (on the Linux side) and the target is a Windows Server 2022 instance, the code needs to be robust enough to handle the half-duplex nature of ARP (where only one side "talks" at a time to avoid collisions).

// 1. The High-Level Architecture (The State Machine)
// The "Main" loop follows a strict sequence to ensure the Admin and the Agent stay in sync.

// State IDLE: Wait for user input on the Linux terminal.

// State SEND: Fragment the string, calculate Hopped MACs, and broadcast.

// State LISTEN: Immediately switch to Promiscuous Mode, predict the next MACs, and reassemble the response.

// State DISPLAY: Print the PowerShell output and return to IDLE.

// 2. The Full "Main" Execution Loop (Go)
// This logic would be compiled into a single binary. For the "Admin" mode, you’d run ./arpshell --mode admin.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/gopacket/pcap"
)

var (
	psk         = "S3cur3_Adm1n_K3y" // Shared secret
	currentSeq  = 0                 // Global sequence tracker
	interfaceName = "eth0"
)

func main() {
	// 1. Setup Network Handle
	handle, err := pcap.OpenLive(interfaceName, 1024, true, pcap.BlockForever)
	if err != nil {
		fmt.Printf("Error opening interface: %v\n", err)
		return
	}
	defer handle.Close()

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== [ Layer 2 ARP-Shell Activated ] ===")
	fmt.Printf("[*] Interface: %s | PSK: Enabled\n", interfaceName)

	for {
		// --- SENDER MODE ---
		fmt.Print("\nARP-Admin> ")
		command, _ := reader.ReadString('\n')
		command = strings.TrimSpace(command)

		if command == "exit" {
			break
		}

		if command == "" {
			continue
		}

		// Fragment and Send
		fmt.Printf("[*] Sending command via %d fragments...\n", (len(command)/3)+1)
		fragments := FragmentCommand(command) // Uses logic from previous step
		
		for _, frag := range fragments {
			// Generate the hopped MAC for THIS specific fragment
			targetMAC := PredictNextMAC(psk, currentSeq)
			
			// Send the raw ARP packet
			SendARPPacket(handle, targetMAC, frag)
			
			currentSeq++ // Increment sequence for the next hop
			time.Sleep(10 * time.Millisecond) // Prevent packet flooding
		}

		// --- RECEIVER MODE ---
		fmt.Println("[*] Switching to Receiver Mode. Waiting for Windows response...")
		
		// This blocks until a "Final" fragment is received from the Windows Server
		response := ListenAndReassemble(handle, psk, &currentSeq)
		
		fmt.Printf("\n[RESPONSE]:\n%s\n", response)
		
		// The 'currentSeq' is now updated and ready for the next command cycle
	}
}
// 3. Step-by-Step Build Sequence for the "Pro" ShellStep 1: The "Predictive" BufferIn the ListenAndReassemble function, you cannot just listen for one MAC. You should pre-calculate the next 5 expected MACs. If the Windows Server sends a response and a packet is dropped, your Linux sniffer should see "Seq 7" and realize "Seq 6" was lost, allowing it to continue reassembling rather than hanging.Step 2: Handling the "Windows Hang"Windows Server 2022 might take 2 seconds to restart a service or 10 seconds to query a large log.Implementation: The Linux "Main" loop should have a Read Timeout. If no ARP packets are seen for 30 seconds, it should assume the command failed or the agent is offline and return to the ARP-Admin> prompt.Step 3: Optimization for Windows (Npcap)On the Windows side, the "Main" loop is the inverse. It stays in Receiver Mode indefinitely. Only after it reassembles a full command does it switch to Sender Mode to broadcast the results of the PowerShell execution.4. Advanced Feature: "Silent Heartbeats"To ensure the IT admin knows the Windows Server is actually "there" without sending loud traffic:Every 5 minutes, the Windows Agent sends one ARP packet.The MAC address for this packet is generated using PredictNextMAC(psk, "HEARTBEAT").The Linux box sniffer can run in the background. If it sees that specific MAC, it updates a "Last Seen" timestamp in the UI.Summary Table for the Software EngineerComponentFunctionLogical ValueCurrentSeqThe "Seed" for the next MAC.Ensures identity rotates.Control Byte0x80 or 0x00.Manages fragment flow.mTLS EquivalentThe PSK + SHA256.Ensures only the Admin can talk to the Agent.TransportRaw Ethernet (ARP).Bypasses the entire TCP/IP stack.