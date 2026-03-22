// To handle a command like RESTART_IIS (11 characters) using a protocol that only gives us 4 bytes per "Source IP" field, we have to implement a Segmentation and Reassembly (SAR) strategy.Since we are using the Source Protocol Address (SPA) field for our payload, we will reserve the first byte as a Control Byte and the remaining 3 bytes for Data.The Control Byte Structure (8 bits)Bit 7 (MSB): "More Fragments" flag ($1$ = more coming, $0$ = last packet).Bits 0-6: Sequence ID (0 to 127). This allows the receiver to reorder packets if they arrive out of sync.1. The Fragmentation Logic (Go)This function takes a string and breaks it into a slice of 4-byte payloads ready for the ARP SourceProtAddress field.
func FragmentCommand(command string) [][]byte {
	const maxDataSize = 3 // We use 1 byte for header, 3 for data
	cmdBytes := []byte(command)
	var fragments [][]byte

	for i := 0; i < len(cmdBytes); i += maxDataSize {
		end := i + maxDataSize
		if end > len(cmdBytes) {
			end = len(cmdBytes)
		}

		// Create the 4-byte ARP field buffer
		fragment := make([]byte, 4)
		
		// Set Control Byte
		seqID := byte(len(fragments))
		if end < len(cmdBytes) {
			fragment[0] = 0x80 | seqID // Set "More Fragments" bit (0x80)
		} else {
			fragment[0] = seqID        // Last fragment
		}

		// Copy data (up to 3 bytes)
		copy(fragment[1:], cmdBytes[i:end])
		fragments = append(fragments, fragment)
	}
	return fragments
}