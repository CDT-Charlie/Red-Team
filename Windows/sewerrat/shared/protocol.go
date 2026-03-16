package shared

import (
	"bytes"
	"fmt"
)

// PayloadEncode chunks a command string into 20-byte blocks for transmission
// Each block will be placed in the Ethernet padding after the magic marker.
// For multi-packet responses, each frame carries up to 20 bytes + null terminator.
func PayloadEncode(command string) []byte {
	if command == "" {
		return nil
	}

	cmdBytes := []byte(command)
	if len(cmdBytes) > MaxResponseSize {
		cmdBytes = cmdBytes[:MaxResponseSize]
	}

	return cmdBytes
}

// PayloadDecode extracts command/output from Ethernet frame padding.
// Assumes frame data starts at the beginning of Ethernet frame.
// Magic marker is expected at offset 42-43, payload follows at offset 44+.
func PayloadDecode(frameData []byte, includeTerminator bool) (string, error) {
	if len(frameData) < ARPPacketSize+MagicMarkerSize {
		return "", fmt.Errorf(ErrInvalidPayload)
	}

	// Check magic marker at offset 42
	magicOffset := ARPPacketSize
	if frameData[magicOffset] != MagicMarkerByte0 || frameData[magicOffset+1] != MagicMarkerByte1 {
		return "", fmt.Errorf("magic marker not found")
	}

	// Extract payload after magic marker
	payloadStart := magicOffset + MagicMarkerSize
	payload := frameData[payloadStart:]

	// Remove null terminators if present
	if !includeTerminator {
		payload = bytes.TrimRight(payload, "\x00")
	}

	return string(payload), nil
}

// FramePadding constructs the Ethernet padding with magic marker + payload.
// This is placed at the end of the ARP packet to reach the 64-byte minimum.
func FramePadding(payload []byte) []byte {
	padding := make([]byte, MaxPaddingSize)

	// Place magic marker at start of padding
	padding[0] = MagicMarkerByte0
	padding[1] = MagicMarkerByte1

	// Copy payload
	payloadSize := len(payload)
	if payloadSize > MaxPayloadPerFrame {
		payloadSize = MaxPayloadPerFrame
	}
	copy(padding[MagicMarkerSize:], payload[:payloadSize])

	return padding
}

// ChunkPayload splits data into 20-byte chunks for sequential transmission.
// Returns a slice of byte slices, each suitable for embedding in Ethernet padding.
func ChunkPayload(data []byte) [][]byte {
	if len(data) == 0 {
		return nil
	}

	var chunks [][]byte
	for i := 0; i < len(data); i += MaxPayloadPerFrame {
		end := i + MaxPayloadPerFrame
		if end > len(data) {
			end = len(data)
		}
		chunks = append(chunks, data[i:end])
	}

	return chunks
}

// JoinChunks reassembles chunked payloads into original data.
// Handles variable-length final chunks.
func JoinChunks(chunks [][]byte) []byte {
	if len(chunks) == 0 {
		return nil
	}

	totalSize := 0
	for _, chunk := range chunks {
		totalSize += len(chunk)
	}

	result := make([]byte, 0, totalSize)
	for _, chunk := range chunks {
		result = append(result, chunk...)
	}

	return result
}

// ValidateMagicMarker checks if the provided frame data contains our magic marker.
func ValidateMagicMarker(frameData []byte) bool {
	if len(frameData) < ARPPacketSize+MagicMarkerSize {
		return false
	}

	magicOffset := ARPPacketSize
	return frameData[magicOffset] == MagicMarkerByte0 &&
		frameData[magicOffset+1] == MagicMarkerByte1
}
