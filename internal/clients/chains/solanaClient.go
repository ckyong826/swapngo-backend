package chains

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mr-tron/base58"
)

type solanaClient struct {
	rpcURL string
}

func NewSolanaClient(rpcURL string) IChainClient {
	return &solanaClient{rpcURL: rpcURL}
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

func (c *solanaClient) GetBalance(ctx context.Context, address string) (string, error) {
	// 构造 Solana JSON-RPC 请求
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getBalance",
		"params":  []any{address},
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", c.rpcURL, bytes.NewBuffer(body))
	if err != nil {
		return "0", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "0", err
	}
	defer resp.Body.Close()

	// 解析返回结果
	var rpcResponse struct {
		Result struct {
			Value uint64 `json:"value"` // 余额单位是 lamports
		} `json:"result"`
		Error any `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rpcResponse); err != nil {
		return "0", err
	}
	if rpcResponse.Error != nil {
		return "0", fmt.Errorf("rpc error: %v", rpcResponse.Error)
	}

	// 转换为 SOL (1 SOL = 1,000,000,000 lamports)
	solBalance := float64(rpcResponse.Result.Value) / 1e9
	return fmt.Sprintf("%.6f", solBalance), nil
}

func (c *solanaClient) TransferMYRC(ctx context.Context, fromPrivateKey, fromAddress, toAddress string, amount float64) (string, error) {
	return "", fmt.Errorf("TransferMYRC not implemented for Solana")
}
func (c *solanaClient) TransferCoin(ctx context.Context, fromPrivateKey, fromAddress, toAddress string, amount float64) (string, error) {
	return "", fmt.Errorf("TransferCoin not implemented for Solana")
}