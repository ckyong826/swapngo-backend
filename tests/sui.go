//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"swapngo-backend/internal/clients/chains"
	config "swapngo-backend/pkg/configs"
)

func main() {
	config.Load()
	suiClient := chains.NewSuiClient(config.Env.SUIChainURL)

ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	log.Println("==================================================")
	log.Println("🚀 阶段 1：模拟新用户注册，生成平台托管钱包...")
	senderAddress := "0x53b43f418824509e49e9d3230c0bdebd7578b3b8d4854fb5e5ab4c8ff334d88b"
	senderPrivateKey := "AGw7RYifM39nIMKGntwyXUNOZxtr/LGulRIEfDdV5lf+"
	
	log.Printf("✅ 新用户 SUI 地址: %s", senderAddress)
	log.Printf("🔑 新用户 Base64 私钥 (将存入数据库): %s", senderPrivateKey)
	log.Println("==================================================")

	// 2. 交互式暂停：等待资金就位
	log.Println("⏸️  请不要关闭终端！我们需要给这个新用户准备测试资金：")
	log.Printf("👉 1. 请前往 Sui Discord 水龙头，给 %s 发送一点测试网 SUI 作为 Gas 费。", senderAddress)
	log.Printf("👉 2. 请使用你的 Admin 钱包，调用 Minting 函数（或在网页浏览器里）给 %s 发送 10 个 MYRC。", senderAddress)
	log.Println("💰 确认上述两步完成后，请在终端按【回车键 (Enter)】继续执行转账测试...")
	
	// 阻塞等待用户按回车
	fmt.Scanln() 

	// 3. 执行转账 (从新用户转给 Admin)
	recipientAddress := "0x0833e3b64a20e7a90ba6720962cc853f61af97d71547e2bb6f16e9f00d5ad1fa" // 你的地址
	transferAmount := 5.5

	log.Println("==================================================")
	log.Printf("🚀 阶段 2：执行 Web3 转账 (提现模拟)...")
	log.Printf("从: %s", senderAddress)
	log.Printf("到: %s", recipientAddress)
	log.Printf("金额: %f MYRC", transferAmount)

	// 这里使用的是绝对符合我们系统标准的 Base64 私钥！
	txHash, err := suiClient.TransferMYRC(ctx, senderPrivateKey, senderAddress, recipientAddress, transferAmount)
	if err != nil {
		log.Fatalf("❌ 转账失败: %v", err)
	}

	log.Printf("✅ 转账成功!")
	log.Printf("🔗 交易哈希: %s", txHash)
	log.Printf("👉 在浏览器查看: https://suiscan.xyz/testnet/tx/%s", txHash)
	log.Println("==================================================")
}