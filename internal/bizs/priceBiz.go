package bizs

import (
	"strconv"
	"swapngo-backend/internal/clients"
	"swapngo-backend/internal/ws"
	"time"
)

type PriceBiz interface {
	StartBroadcasting()
}

type priceBiz struct {
	hub     *ws.Hub
	started bool
}

func NewPriceBiz(hub *ws.Hub) PriceBiz {
	return &priceBiz{hub: hub}
}

func parsePrice(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func (b *priceBiz) StartBroadcasting() {
	if b.started {
		return
	}
	b.started = true

	ticker := time.NewTicker(3 * time.Second)
	go func() {
		for range ticker.C {
			clients.PriceMux.RLock()
			ethUSD := parsePrice(clients.LatestPrices["ETHUSDT"])
			suiUSD := parsePrice(clients.LatestPrices["SUIUSDT"])
			btcUSD := parsePrice(clients.LatestPrices["BTCUSDT"])
			usdMyr := parsePrice(clients.LatestPrices["USDMYR"])
			clients.PriceMux.RUnlock()

			prices := map[string]float64{
				"MYRC": 1.00,
				"USDT": usdMyr,
				"USDC": usdMyr,
				"BTC":  btcUSD * usdMyr,
				"ETH":  ethUSD * usdMyr,
				"SUI":  suiUSD * usdMyr,
			}

			b.hub.BroadcastAll(ws.Event("PRICE_UPDATE", prices))
		}
	}()
}