package chains

import (
	"crypto/ed25519"
	"crypto/rand"
	"fmt"

	"github.com/mr-tron/base58"
)

type solanaClient struct{}

func NewSolanaClient() IChainClient {
	return &solanaClient{}
}

func (c *solanaClient) GenerateAddress() (string, string, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate solana keys: %w", err)
	}

	address := base58.Encode(publicKey)
	privateKeyBase58 := base58.Encode(privateKey)

	return address, privateKeyBase58, nil
}
