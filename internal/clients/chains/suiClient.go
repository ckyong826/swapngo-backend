package chains

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"golang.org/x/crypto/blake2b"
)

type suiClient struct {
	rpcURL string
}

func NewSuiClient(rpcURL string) IChainClient {
	return &suiClient{rpcURL: rpcURL}
}

func (c *suiClient) GenerateAddress() (string, string, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate sui keys: %w", err)
	}

	data := make([]byte, 1+ed25519.PublicKeySize)
	data[0] = 0x00 
	copy(data[1:], publicKey)

	hash := blake2b.Sum256(data)
	address := "0x" + hex.EncodeToString(hash[:])

	privateKeyHex := hex.EncodeToString(privateKey)

	return address, privateKeyHex, nil
}

func (c *suiClient) GetBalance(ctx context.Context, address string) (string, error) {
	// 构造 Sui JSON-RPC 请求 (查询原生 SUI 代币)
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "suix_getBalance",
		"params":  []any{address, "0x2::sui::SUI"},
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
			TotalBalance string `json:"totalBalance"` // Sui 节点返回的是字符串类型的 MIST
		} `json:"result"`
		Error any `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rpcResponse); err != nil {
		return "0", err
	}
	if rpcResponse.Error != nil {
		return "0", fmt.Errorf("rpc error: %v", rpcResponse.Error)
	}

	// 将字符串 MIST 解析并转换为 SUI (1 SUI = 1,000,000,000 MIST)
	mist, _ := strconv.ParseFloat(rpcResponse.Result.TotalBalance, 64)
	suiBalance := mist / 1e9

	return fmt.Sprintf("%.6f", suiBalance), nil
}

func (c *suiClient) TransferMYRC(ctx context.Context, fromAddress, toAddress string, amount float64) (string, error) {
	// TODO: Implement actual SUI transaction signing and execution for MYRC token transfer.
	// This requires an admin private key and the MYRC token contract address.
	// For now, we return a mock transaction hash to simulate a successful mint.
	return "mocked_sui_tx_hash_for_myrc_mint", nil
}