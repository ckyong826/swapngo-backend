package chains

import "context"

type IChainClient interface {
	GenerateAddress() (address string, privateKey string, err error)
	GetBalance(ctx context.Context, address string) (string, error)
	TransferMYRC(ctx context.Context, fromPrivateKey, fromAddress, toAddress string, amount float64) (txHash string, err error)
	TransferCoin(ctx context.Context, fromPrivateKey, fromAddress, toAddress string, amountSUI float64) (string, error) 
}
