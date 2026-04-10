package bizs

import (
	"swapngo-backend/internal/clients"
	"swapngo-backend/internal/ws"
	"time"
)

type PriceBiz interface {
	StartPushing(userID string)
}

type priceBiz struct {
	hub *ws.Hub
}

func NewPriceBiz(hub *ws.Hub) PriceBiz {
	return &priceBiz{hub: hub}
}

func (b *priceBiz) StartPushing(userID string) {
	ticker := time.NewTicker(2 * time.Second)

	go func() {
		for range ticker.C {
			// 1. 获取币安真实价格
			clients.PriceMux.RLock()
			ethPrice := clients.LatestPrices["ETHUSDT"]
			suiPrice := clients.LatestPrices["SUIUSDT"]
			solPrice := clients.LatestPrices["SOLUSDT"]
			usdMyrRate := clients.LatestPrices["USDMYR"]
			clients.PriceMux.RUnlock()

			// 2. 构造推送负载
			pushData := map[string]any{
				"prices": map[string]any{
					"ETH":  ethPrice,
					"SUI":  suiPrice,
					"SOL":  solPrice,
					"USDMYR": usdMyrRate,
				},
				"timestamp": time.Now().Unix(),
			}

			// 4. 推送
			b.hub.SendToUser(userID, pushData)
		}
	}()
}