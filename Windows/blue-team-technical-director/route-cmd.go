// Step-by-Step Build Sequence
// Step 1: Syncing the Sequence
// Create a "Handshake" packet. When the Admin starts, it sends a packet with Sequence 0. This resets the Windows Agent's counter so both sides start hopping from the same seed.

// Step 2: The Command Router
// On the Windows side, create a secure execution environment:
func RouteCommand(input string) string {
	// Map shorthand to long PowerShell scripts
	switch input {
	case "who":
		return RunPS("whoami /all")
	case "net":
		return RunPS("Get-NetIPAddress | Select-Object InterfaceAlias, IPAddress")
	default:
		return "Unknown Command"
	}
}
// Step 3: Implementing "Reliable" Delivery
// Since ARP has no "Retry" logic, add a 1-byte CRC8 checksum to your fragment.

// Control Byte: [More Bit (1) | Seq ID (7)] (1 byte)

// Checksum: [CRC8] (1 byte)

// Data: [Payload] (2 bytes)
// If the CRC doesn't match on the receiver side, the receiver ignores the fragment, and the Admin (not seeing an ACK) will re-transmit after a timeout.