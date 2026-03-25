package mac

import (
	"crypto/sha256"
	"net"
	"strconv"
)

// GenerateHoppedMAC derives a deterministic MAC address from a pre-shared key
// and sequence ID. The first byte is 0x02 (locally administered bit) so the
// address looks like a valid virtual/software MAC on the wire.
func GenerateHoppedMAC(psk string, sequenceID int) net.HardwareAddr {
	hasher := sha256.New()
	hasher.Write([]byte(psk + strconv.Itoa(sequenceID)))
	hash := hasher.Sum(nil)

	return net.HardwareAddr{
		0x02,
		hash[0], hash[1], hash[2], hash[3], hash[4],
	}
}

// PredictNextMAC returns the string representation of the hopped MAC for the
// given sequence ID, used by the receiver to filter matching packets.
func PredictNextMAC(psk string, seqID int) string {
	return GenerateHoppedMAC(psk, seqID).String()
}
