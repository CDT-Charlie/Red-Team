func GenerateHoppedMAC(psk string, sequenceID int) net.HardwareAddr {
	// Use a hash of the PSK + SeqID to create a deterministic "random" MAC
	hasher := sha256.New()
	hasher.Write([]byte(psk + strconv.Itoa(sequenceID)))
	hash := hasher.Sum(nil)

	// Construct a MAC (6 bytes)
	// We set the 'Locally Administered' bit (0x02) to look like a valid virtual MAC
	mac := net.HardwareAddr{
		0x02, 
		hash[0], hash[1], hash[2], hash[3], hash[4],
	}
	return mac
}