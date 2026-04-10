package clients

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	// 全局最新价格缓存
	LatestPrices = make(map[string]string)
	PriceMux     sync.RWMutex
)

func StartPriceWorker() {
	// 订阅 ETH, SOL, SUI 的实时价格流
	url := "wss://stream.binance.com:9443/ws/ethusdt@ticker/solusdt@ticker/suiusdt@ticker"

	// USD -> MYR 汇率轮询
	go startUSDToMYRWorker()

	go func() {
		for {
			conn, _, err := websocket.DefaultDialer.Dial(url, nil)
			if err != nil {
				log.Println("Binance WS Dial error, retrying...", err)
				continue
			}

			for {
				_, message, err := conn.ReadMessage()
				if err != nil {
					break
				}
				var data struct {
					Symbol string `json:"s"`
					Price  string `json:"c"`
				}
				if err := json.Unmarshal(message, &data); err != nil {
					continue
				}

				PriceMux.Lock()
				LatestPrices[data.Symbol] = data.Price
				PriceMux.Unlock()
			}
		}
	}()
}

func startUSDToMYRWorker() {
	// 先拉一次，避免首次推送无汇率
	refreshUSDToMYR()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		refreshUSDToMYR()
	}
}

func refreshUSDToMYR() {
	resp, err := http.Get("https://open.er-api.com/v6/latest/USD")
	if err != nil {
		log.Println("USD/MYR rate fetch error:", err)
		return
	}
	defer resp.Body.Close()

	var data struct {
		Result string             `json:"result"`
		Rates  map[string]float64 `json:"rates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Println("USD/MYR decode error:", err)
		return
	}

	if data.Result != "success" {
		log.Println("USD/MYR api result not success:", data.Result)
		return
	}

	myr, ok := data.Rates["MYR"]
	if !ok {
		log.Println("USD/MYR not found in rates")
		return
	}

	PriceMux.Lock()
	LatestPrices["USDMYR"] = fmt.Sprintf("%.6f", myr)
	PriceMux.Unlock()
}