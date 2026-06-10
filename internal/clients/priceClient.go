package clients

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

var (
	LatestPrices = make(map[string]string)
	PriceMux     sync.RWMutex
)

// StartPriceWorker polls CoinGecko every 30s for BTC/ETH/SUI prices
// and polls open.er-api.com every 30s for USD/MYR rate.
func StartPriceWorker() {
	// Prime both immediately, then start tickers
	refreshCoinGeckoPrices()
	refreshUSDToMYR()

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			refreshCoinGeckoPrices()
		}
	}()

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			refreshUSDToMYR()
		}
	}()
}

func refreshCoinGeckoPrices() {
	url := "https://api.coingecko.com/api/v3/simple/price?ids=bitcoin,ethereum,sui&vs_currencies=usd"

	resp, err := http.Get(url)
	if err != nil {
		log.Println("CoinGecko fetch error:", err)
		return
	}
	defer resp.Body.Close()

	// Response shape: {"bitcoin":{"usd":65000},"ethereum":{"usd":3500},"sui":{"usd":1.2}}
	var data map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Println("CoinGecko decode error:", err)
		return
	}

	PriceMux.Lock()
	defer PriceMux.Unlock()

	if btc, ok := data["bitcoin"]["usd"]; ok {
		LatestPrices["BTCUSDT"] = fmt.Sprintf("%.2f", btc)
	}
	if eth, ok := data["ethereum"]["usd"]; ok {
		LatestPrices["ETHUSDT"] = fmt.Sprintf("%.2f", eth)
	}
	if sui, ok := data["sui"]["usd"]; ok {
		LatestPrices["SUIUSDT"] = fmt.Sprintf("%.6f", sui)
	}

	log.Println("CoinGecko prices updated")
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

	log.Println("USD/MYR rate updated:", myr)
}
