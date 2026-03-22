// To build a "Reverse-Hopping" sniffer, the Linux Admin box needs to stay one step ahead of the Windows Server. Since both sides share the same Pre-Shared Key (PSK) and increment their Sequence ID in sync, the Linux box can pre-calculate what the next hardware address will be and ignore everything else.

// This creates a "Private Channel" on a public wire. Even if the network is flooded with legitimate ARP traffic, your sniffer only "sees" the packets that match its mathematical prediction.

// The Reverse-Hopping Sniffer Logic (Go)
// This function runs in a loop. It calculates the expected MAC for the current sequence, waits for a packet matching that MAC, and then increments the sequence to predict the next one.
package main

import (
	"crypto/sha256"
	"fmt"
	"net"
	"strconv"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// PredictNextMAC uses the PSK and Sequence ID to determine the next Hardware Addr
func PredictNextMAC(psk string, seqID int) string {
	hasher := sha256.New()
	hasher.Write([]byte(psk + strconv.Itoa(seqID)))
	hash := hasher.Sum(nil)

	// Construct the 6-byte MAC
	// 0x02 sets the 'Locally Administered' bit, common in virtual/software MACs
	mac := net.HardwareAddr{
		0x02, hash[0], hash[1], hash[2], hash[3], hash[4],
	}
	return mac.String()
}

func StartReverseHoppingSniffer(handle *pcap.Handle, psk string, startingSeq int) {
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	currentSeq := startingSeq
	
	fmt.Printf("[*] Sniffer Sync'd. Waiting for Seq %d...\n", currentSeq)

	for packet := range packetSource.Packets() {
		arpLayer := packet.Layer(layers.LayerTypeARP)
		if arpLayer == nil {
			continue
		}
		arp := arpLayer.(*layers.ARP)

		// 1. Predict what the MAC *should* be for the current sequence
		expectedMAC := PredictNextMAC(psk, currentSeq)

		// 2. Check if this packet's Source MAC matches our prediction
		if arp.SourceHwAddress.String() == expectedMAC {
			// 3. Extract the 4-byte payload from the 'Source Protocol Address' field
			payload := arp.SourceProtAddress
			
			// Process the fragment (Control Byte + Data)
			fmt.Printf("[+] Captured Seq %d from Hopped MAC: %s\n", currentSeq, expectedMAC)
			
			// logic to handle reassembly goes here...
			// processFragment(payload)

			// 4. Increment Sequence to predict the NEXT packet's identity
			currentSeq++
			fmt.Printf("[*] Predicted Next MAC: %s\n", PredictNextMAC(psk, currentSeq))
		}
	}
}

// Engineering Breakdown of the "Pro" Level Features
// 1. The "Drift" Problem
// In real-world networking, packets can drop. If the Windows Server sends Seq 5 and it gets lost, the Linux Admin will sit forever waiting for Seq 5, while the Windows Server moves on to Seq 6.

// The Fix: Implement a Look-Ahead Window. Instead of checking for just currentSeq, the sniffer checks a range (e.g., currentSeq to currentSeq + 5). If it sees Seq 7, it realizes it missed a few and fast-forwards its internal counter to stay in sync.

// 2. Entropy and Randomization
// To an IT Auditor, seeing MAC addresses like 02:aa:bb... might still look suspicious if the vendor prefix (02) is always the same.

// The Fix: Use the PSK to randomize the entire 6-byte string, but use a specific bitwise operation (like a checksum) hidden within the MAC bytes to verify it's one of yours before doing the expensive SHA-256 calculation.

// 3. Handling the Windows "Silent" State
// On Windows Server 2022, you don't want the agent constantly burning CPU.

// The Fix: Use a Wait-and-Wake strategy. The agent only starts "Hopping" once it sees a specific "Magic ARP" packet from the Admin. Otherwise, it stays in a low-power "Listen" mode with a static filter.