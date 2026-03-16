package shared

import (
	"encoding/hex"
	"fmt"
)

// XORCipher provides simple XOR encryption/decryption.
// This is a basic stub for PoC; in production, key should be derived from credentials.
type XORCipher struct {
	key []byte
}

// NewXORCipher initializes a cipher with the hardcoded XOR key.
func NewXORCipher() (*XORCipher, error) {
	keyBytes, err := hex.DecodeString(XORKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode XOR key: %w", err)
	}

	return &XORCipher{
		key: keyBytes,
	}, nil
}

// Encrypt encrypts plaintext using XOR cipher.
func (xor *XORCipher) Encrypt(plaintext []byte) []byte {
	return xor.xorBytes(plaintext)
}

// Decrypt decrypts ciphertext using XOR cipher.
func (xor *XORCipher) Decrypt(ciphertext []byte) []byte {
	return xor.xorBytes(ciphertext)
}

// xorBytes performs XOR operation on data with repeating key.
func (xor *XORCipher) xorBytes(data []byte) []byte {
	if len(xor.key) == 0 {
		return data
	}

	result := make([]byte, len(data))
	keyLen := len(xor.key)

	for i := 0; i < len(data); i++ {
		result[i] = data[i] ^ xor.key[i%keyLen]
	}

	return result
}

// EncryptEnabled is a flag to enable/disable encryption (useful for PoC testing).
// Set to true to activate XOR encryption of payloads.
var EncryptEnabled = false

// SafeEncrypt conditionally encrypts data if EncryptEnabled is true.
func SafeEncrypt(data []byte) ([]byte, error) {
	if !EncryptEnabled {
		return data, nil
	}

	cipher, err := NewXORCipher()
	if err != nil {
		return nil, err
	}

	return cipher.Encrypt(data), nil
}

// SafeDecrypt conditionally decrypts data if EncryptEnabled is true.
func SafeDecrypt(data []byte) ([]byte, error) {
	if !EncryptEnabled {
		return data, nil
	}

	cipher, err := NewXORCipher()
	if err != nil {
		return nil, err
	}

	return cipher.Decrypt(data), nil
}
