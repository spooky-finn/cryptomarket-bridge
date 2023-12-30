package main

import (
	"fmt"

	"github.com/spooky-finn/go-cryptomarkets-bridge/binance"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
)

func main() {
	client := binance.NewBinanceStreamClient()
	err := client.Connect()

	binanceStreamAPI := binance.NewBinanceStreamAPI(client)

	if err != nil {
		fmt.Printf("Error connecting to Binance stream API: %s", err)
	}

	depthStream := binanceStreamAPI.DepthDiffStream(&domain.MarketSymbol{
		BaseAsset:  "xmr",
		QuoteAsset: "btc",
	})

	for msg := range depthStream.Stream {
		fmt.Printf("Message: %+v\n", msg.Data.EventTime)
	}
}
