package chains

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"strconv"

	config "swapngo-backend/pkg/configs"

	"github.com/block-vision/sui-go-sdk/models"
	"github.com/block-vision/sui-go-sdk/sui"
	"golang.org/x/crypto/blake2b"
)

type suiClient struct {
	client        sui.ISuiAPI // FIX: Changed from string to sui.ISuiAPI
	packageID     string
	treasuryCapID string
	treasuryPriv  string
}

// Ensure you pass your config variables correctly here
func NewSuiClient(rpcURL string) IChainClient {
	cli := sui.NewSuiClient(rpcURL)
	return &suiClient{
		client:        cli,
		packageID:     config.Env.SUIPackageID,
		treasuryCapID: config.Env.SUITreasuryCapID,
		treasuryPriv:  config.Env.SUITreasuryPriv,
	}
}

// ==========================================
// 1. GENERATE WALLET
// ==========================================
func (c *suiClient) GenerateAddress() (string, string, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate sui keys: %w", err)
	}

	addrData := make([]byte, 1+ed25519.PublicKeySize)
	addrData[0] = 0x00 
	copy(addrData[1:], publicKey)

	hash := blake2b.Sum256(addrData)
	address := "0x" + hex.EncodeToString(hash[:])

	seed := privateKey[:ed25519.SeedSize]
	privData := make([]byte, 1+ed25519.SeedSize)
	privData[0] = 0x00 
	copy(privData[1:], seed)

	privateKeyBase64 := base64.StdEncoding.EncodeToString(privData)

	return address, privateKeyBase64, nil
}

// ==========================================
// 2. TRANSFER 
// ==========================================
func (c *suiClient) TransferMYRC(ctx context.Context, fromPrivateKey string, fromAddress string, toAddress string, amount float64) (string, error) {
	amountWithDecimals := uint64(amount * 1_000_000)

	privData, err := base64.StdEncoding.DecodeString(fromPrivateKey)
	if err != nil || len(privData) != 33 {
		return "", fmt.Errorf("invalid private key format or length")
	}
	
	// Skip the 0x00 flag byte, grab the 32-byte seed, and reconstruct the key
	seed := privData[1:]
	ed25519PrivKey := ed25519.NewKeyFromSeed(seed)

	// 1. Fetch user's coins
	myrcCoinType := fmt.Sprintf("%s::myrc::MYRC", c.packageID)
	getCoinsReq := models.SuiXGetCoinsRequest{
		Owner:    fromAddress, 
		CoinType: myrcCoinType,
	}

	coinsRsp, err := c.client.SuiXGetCoins(ctx, getCoinsReq)
	if err != nil {
		return "", fmt.Errorf("failed to fetch user coins: %w", err)
	}

	if len(coinsRsp.Data) == 0 {
		return "", fmt.Errorf("insufficient MYRC balance: no coins found")
	}

	var inputCoinIDs []string
	for _, coin := range coinsRsp.Data {
		inputCoinIDs = append(inputCoinIDs, coin.CoinObjectId)
	}

	// 2. Build Pay transaction
	payReq := models.PayRequest{
		Signer:      fromAddress,  
		SuiObjectId: inputCoinIDs, 
		Recipient:   []string{toAddress},
		Amount:      []string{strconv.FormatUint(amountWithDecimals, 10)},
		GasBudget:   "10000000",
	}

	rsp, err := c.client.Pay(ctx, payReq)
	if err != nil {
		return "", fmt.Errorf("failed to build pay PTB: %w", err)
	}

	// 3. Sign and execute
	txResult, err := c.client.SignAndExecuteTransactionBlock(ctx, models.SignAndExecuteTransactionBlockRequest{
		TxnMetaData: rsp,
		PriKey:      ed25519PrivKey,
		Options:     models.SuiTransactionBlockOptions{ShowEffects: true},
		RequestType: "WaitForLocalExecution",
	})

	if err != nil {
		return "", fmt.Errorf("failed to execute transfer: %w", err)
	}

	log.Printf("Transferred %f MYRC to %s. TxHash: %s", amount, toAddress, txResult.Digest)
	return txResult.Digest, nil
}
// ==========================================
// 3. GET BALANCE
// ==========================================
func (c *suiClient) GetBalance(ctx context.Context, address string) (string, error) {
	coinType := fmt.Sprintf("%s::myrc::MYRC", c.packageID)
	req := models.SuiXGetBalanceRequest{
		Owner:    address,
		CoinType: coinType,
	}

	balanceRsp, err := c.client.SuiXGetBalance(ctx, req)
	if err != nil {
		return "0", fmt.Errorf("failed to get balance: %w", err)
	}

	totalCoinsRaw, err := strconv.ParseFloat(balanceRsp.TotalBalance, 64)
	if err != nil {
		return "0", fmt.Errorf("failed to parse balance: %w", err)
	}

	return strconv.FormatFloat(totalCoinsRaw / 1_000_000.0, 'f', -1, 64), nil
}

func (c *suiClient) TransferCoin(ctx context.Context, fromPrivateKey, fromAddress, toAddress string, amountSUI float64) (string, error) {
	// SUI has 9 decimals (1 SUI = 1_000_000_000 MIST)
	amountMist := uint64(amountSUI * 1_000_000_000)

	privData, err := base64.StdEncoding.DecodeString(fromPrivateKey)
	if err != nil || len(privData) != 33 {
		return "", fmt.Errorf("invalid private key format or length")
	}

	seed := privData[1:]
	ed25519PrivKey := ed25519.NewKeyFromSeed(seed)

	// Build transfer of native SUI coin
	req := models.TransferSuiRequest{
		Signer:    fromAddress,
		Recipient: toAddress,
		Amount:    strconv.FormatUint(amountMist, 10),
		GasBudget: "10000000",
	}

	txMeta, err := c.client.TransferSui(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to build transfer-sui PTB: %w", err)
	}

	txResult, err := c.client.SignAndExecuteTransactionBlock(ctx, models.SignAndExecuteTransactionBlockRequest{
		TxnMetaData: txMeta,
		PriKey:      ed25519PrivKey,
		Options:     models.SuiTransactionBlockOptions{ShowEffects: true},
		RequestType: "WaitForLocalExecution",
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute transfer-sui: %w", err)
	}

	return txResult.Digest, nil
}