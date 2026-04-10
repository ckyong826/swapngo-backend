package clients

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	// 全局最新价格缓存
	LatestPrices    = make(map[string]string)
	PriceMux        sync.RWMutex
)

func StartPriceWorker() {
	// 订阅 ETH, SOL, SUI 的实时价格流
	url := "wss://stream.binance.com:9443/ws/ethusdt@ticker/solusdt@ticker/suiusdt@ticker"
	
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
				json.Unmarshal(message, &data)

				PriceMux.Lock()
				LatestPrices[data.Symbol] = data.Price
				PriceMux.Unlock()
			}
		}
	}()
}