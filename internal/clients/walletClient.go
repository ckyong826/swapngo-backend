package clients

import (
	"fmt"
	"swapngo-backend/internal/clients/chains"
	"swapngo-backend/internal/models"
)

type WalletClient interface {
	GenerateAddress(chain models.ChainName) (string, string, error)
}

type walletClient struct {
	clients map[models.ChainName]chains.IChainClient
}

func NewWalletClient() WalletClient {
	// Initialize with all supported chain clients
	evmClient := chains.NewEVMClient()
	return &walletClient{
		clients: map[models.ChainName]chains.IChainClient{
			models.ChainEthereum: evmClient,
			models.ChainPolygon:  evmClient,
			models.ChainSolana:   chains.NewSolanaClient(),
			models.ChainSui:      chains.NewSuiClient(),
			},
	}
}

func (c *walletClient) GenerateAddress(chain models.ChainName) (string, string, error) {
	client, exists := c.clients[chain]
	if !exists {
		return "", "", fmt.Errorf("unsupported chain: %s", chain)
	}

	return client.GenerateAddress()
}
