package bizs

import (
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

func (b *priceBiz) StartBroadcasting() {
	if b.started {
		return
	}
	b.started = true

	ticker := time.NewTicker(3 * time.Second)
	go func() {
		for range ticker.C {
			clients.PriceMux.RLock()
			ethUSD := clients.PriceOrFallback("ETHUSDT", clients.FallbackETHUSDT)
			suiUSD := clients.PriceOrFallback("SUIUSDT", clients.FallbackSUIUSDT)
			btcUSD := clients.PriceOrFallback("BTCUSDT", clients.FallbackBTCUSDT)
			usdMyr := clients.PriceOrFallback("USDMYR", clients.FallbackUSDMYR)
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