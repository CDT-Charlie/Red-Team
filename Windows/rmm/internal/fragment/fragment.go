package fragment

import (
	"sort"
	"strings"
)

const (
	MaxDataSize    = 3
	FragmentSize   = 4
	MoreFragsBit   = 0x80
	SeqIDMask      = 0x7F
)

// FragmentCommand splits a command string into 4-byte ARP payload chunks.
// Each chunk: [ControlByte, Data0, Data1, Data2].
// ControlByte bit 7 = more-fragments flag, bits 0-6 = sequence ID.
func FragmentCommand(command string) [][]byte {
	cmdBytes := []byte(command)
	var fragments [][]byte

	for i := 0; i < len(cmdBytes); i += MaxDataSize {
		end := i + MaxDataSize
		if end > len(cmdBytes) {
			end = len(cmdBytes)
		}

		fragment := make([]byte, FragmentSize)

		seqID := byte(len(fragments))
		if end < len(cmdBytes) {
			fragment[0] = MoreFragsBit | seqID
		} else {
			fragment[0] = seqID
		}

		copy(fragment[1:], cmdBytes[i:end])
		fragments = append(fragments, fragment)
	}
	return fragments
}

// CommandBuffer holds reassembly state for incoming ARP fragments.
type CommandBuffer struct {
	Fragments     map[int][]byte
	IsComplete    bool
	ExpectedCount int
}

func NewCommandBuffer() *CommandBuffer {
	return &CommandBuffer{
		Fragments: make(map[int][]byte),
	}
}

// ProcessFragment ingests a 4-byte SPA payload and returns true when the
// full command has been received and is ready for reassembly.
func (cb *CommandBuffer) ProcessFragment(spa []byte) bool {
	if len(spa) < FragmentSize {
		return false
	}

	controlByte := spa[0]
	isFinal := (controlByte & MoreFragsBit) == 0
	seqID := int(controlByte & SeqIDMask)
	data := make([]byte, MaxDataSize)
	copy(data, spa[1:FragmentSize])

	cb.Fragments[seqID] = data

	if isFinal {
		cb.IsComplete = true
		cb.ExpectedCount = seqID + 1
	}

	return cb.IsComplete && len(cb.Fragments) == cb.ExpectedCount
}

// Reassemble sorts fragments by sequence ID, concatenates the data bytes,
// trims null padding, and returns the reconstructed command string.
func (cb *CommandBuffer) Reassemble() string {
	keys := make([]int, 0, len(cb.Fragments))
	for k := range cb.Fragments {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	var b strings.Builder
	for _, k := range keys {
		b.WriteString(strings.TrimRight(string(cb.Fragments[k]), "\x00"))
	}
	return b.String()
}

// Reset clears the buffer for the next command cycle.
func (cb *CommandBuffer) Reset() {
	cb.Fragments = make(map[int][]byte)
	cb.IsComplete = false
	cb.ExpectedCount = 0
}
