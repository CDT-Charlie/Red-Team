// To implement the reassembly logic on Windows Server 2022, the agent needs a stateful buffer. Because ARP packets can technically arrive out of order (though rare on a local switch), we use a map[int][]byte to store fragments by their sequence ID and a bool to track if we've seen the "final" packet.

// Windows Reassembly Logic (Go)
// This snippet assumes you are using gopacket to sniff the wire. It processes each incoming ARP packet, extracts the "smuggled" data, and reconstructs the string.
package main

import (
	"fmt"
	"sort"
	"strings"
)

// CommandBuffer holds the state of a fragmented command
type CommandBuffer struct {
	Fragments map[int][]byte
	IsComplete bool
	ExpectedCount int
}

var currentBuffer = CommandBuffer{
	Fragments: make(map[int][]byte),
}

func processARPPacket(spa []byte) {
	if len(spa) < 4 {
		return
	}

	// 1. Extract the Control Byte
	controlByte := spa[0]
	isFinal := (controlByte & 0x80) == 0 // If MSB is 0, it's the last one
	seqID := int(controlByte & 0x7F)     // Remaining 7 bits are the ID

	// 2. Extract Data (Bytes 1-3)
	data := spa[1:]

	// 3. Store in Buffer
	currentBuffer.Fragments[seqID] = data

	if isFinal {
		currentBuffer.IsComplete = true
		currentBuffer.ExpectedCount = seqID + 1
		fmt.Printf("[!] Received final fragment (Seq: %d). Reassembling...\n", seqID)
	}

	// 4. Attempt Reassembly
	if currentBuffer.IsComplete && len(currentBuffer.Fragments) == currentBuffer.ExpectedCount {
		finalizeCommand()
	}
}

func finalizeCommand() {
	// Sort keys to ensure correct order
	keys := make([]int, 0, len(currentBuffer.Fragments))
	for k := range currentBuffer.Fragments {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	var builder strings.Builder
	for _, k := range keys {
		// Trim null bytes (0x00) used for padding
		fragment := string(currentBuffer.Fragments[k])
		builder.WriteString(strings.TrimRight(fragment, "\x00"))
	}

	finalCmd := builder.String()
	// Security Mapping: Only allow specific commands
	// This prevents an attacker from sending "rm -rf C:\" via ARP
	allowedCommands := map[string]string{
		"RESTART_IIS": "Restart-Service W3SVC -Force",
		"GET_SERVICES": "Get-Service | Where-Object {$_.Status -eq 'Running'} | Select-Object -First 5",
		"DISK_USAGE":   "Get-PSDrive C | Select-Object Used, Free",
	}

	if psCode, exists := allowedCommands[finalCmd]; exists {
		ExecutePowerShell(psCode)
	} else {
		fmt.Printf("[!] Blocked: '%s' is not in the allow-list.\n", finalCmd)
	}
	
	// Reset buffer for next command
	currentBuffer.Fragments = make(map[int][]byte)
	currentBuffer.IsComplete = false
}

func main() {
	// Simulation of receiving the fragments for "RESTART_IIS"
	// Chunks: [0x80, R, E, S], [0x81, T, A, R], [0x82, T, _, I], [0x03, I, S, 0x00]
	// Note: In our logic, 0x80-0x82 have the 'More' bit set. 0x03 is the final Seq 3.
	
	packets := [][]byte{
		{0x80, 'R', 'E', 'S'},
		{0x81, 'T', 'A', 'R'},
		{0x82, 'T', '_', 'I'},
		{0x03, 'I', 'S', 0x00},
	}

	for _, p := range packets {
		processARPPacket(p)
	}
}