package chains

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/blake2b"
)

type suiClient struct{}

func NewSuiClient() IChainClient {
	return &suiClient{}
}

func (c *suiClient) GenerateAddress() (string, string, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate sui keys: %w", err)
	}

	// For Sui using Ed25519, the address is the first 32 bytes of the Blake2b-256 hash of:
	// a 1-byte signature scheme flag (0x00 for Ed25519) + the 32-byte public key.
	data := make([]byte, 1+ed25519.PublicKeySize)
	data[0] = 0x00 // Signature Scheme Flag for Ed25519
	copy(data[1:], publicKey)

	hash := blake2b.Sum256(data)
	address := "0x" + hex.EncodeToString(hash[:])

	// Store private key as hex
	privateKeyHex := hex.EncodeToString(privateKey)

	return address, privateKeyHex, nil
}
