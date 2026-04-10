package clients

import (
	"context"
	"fmt"
	"swapngo-backend/internal/clients/chains"
	"swapngo-backend/internal/models"
	config "swapngo-backend/pkg/configs"
)

type WalletClient interface {
	GenerateAddress(chain models.ChainName) (string, string, error)
	GetBalance(ctx context.Context, chain models.ChainName, address string) (string, error)
}

type walletClient struct {
	clients map[models.ChainName]chains.IChainClient
}

func NewWalletClient() WalletClient {
	// Initialize with all supported chain clients
	evmClient, _ := chains.NewEVMClient(config.Env.EVMChainURL)
	solClient := chains.NewSolanaClient(config.Env.SOLChainURL)
	suiClient := chains.NewSuiClient(config.Env.SUIChainURL)
	return &walletClient{
		clients: map[models.ChainName]chains.IChainClient{
			models.ChainEthereum: evmClient,
			models.ChainPolygon:  evmClient,
			models.ChainSolana:   solClient,
			models.ChainSui:      suiClient,
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

func (c *walletClient) GetBalance(ctx context.Context, chain models.ChainName, address string) (string, error) {
	client, exists := c.clients[chain]
	if !exists {
		return "0", fmt.Errorf("unsupported chain: %s", chain)
	}
	return client.GetBalance(ctx, address)

}
