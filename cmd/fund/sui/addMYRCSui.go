package main

import (
	"context"
	"log"
	"time"

	"swapngo-backend/internal/clients/chains"
	config "swapngo-backend/pkg/configs"
)

func main() {
	config.Load()
	suiClient := chains.NewSuiClient(config.Env.SUIChainURL)

	
	// 那个新用户的地址
	targetAddress := "0x53b43f418824509e49e9d3230c0bdebd7578b3b8d4854fb5e5ab4c8ff334d88b" 

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	adminAddress := config.Env.SUITreasuryAddress
	suiClient.TransferMYRC(ctx, adminAddress, config.Env.SUITreasuryPriv, targetAddress, 10.0)

	log.Println("✅ 成功给测试用户充值 10 MYRC!")
}