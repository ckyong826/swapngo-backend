package chains

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type evmClient struct {
	rpcClient *ethclient.Client
}

// NewEVMClient 现在需要传入节点的 URL (例如 Infura 或 Alchemy)
func NewEVMClient(rpcURL string) (IChainClient, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to EVM RPC: %w", err)
	}
	return &evmClient{rpcClient: client}, nil
}

func (c *evmClient) GenerateAddress() (string, string, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate evm private key: %w", err)
	}

	privateKeyBytes := crypto.FromECDSA(privateKey)
	privateKeyHex := hex.EncodeToString(privateKeyBytes)

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return "", "", fmt.Errorf("failed to cast evm public key to ECDSA")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()

	return address, privateKeyHex, nil
}

func (c *evmClient) GetBalance(ctx context.Context, address string) (string, error) {
	account := common.HexToAddress(address)
	
	// 获取 Wei 单位的余额 (1 ETH = 10^18 Wei)
	balanceWei, err := c.rpcClient.BalanceAt(ctx, account, nil)
	if err != nil {
		return "0", fmt.Errorf("failed to get evm balance: %w", err)
	}

	// 将 Wei 转换为 ETH
	fbalance := new(big.Float)
	fbalance.SetString(balanceWei.String())
	ethValue := new(big.Float).Quo(fbalance, big.NewFloat(math.Pow10(18)))

	return ethValue.Text('f', 6), nil // 保留 6 位小数
}

func (c *evmClient) TransferMYRC(ctx context.Context, toAddress string, amount float64) (string, error) {
	return "", fmt.Errorf("TransferMYRC not implemented for EVM")
}