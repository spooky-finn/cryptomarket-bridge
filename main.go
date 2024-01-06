package main

import (
	"log"
	"time"

	"github.com/spooky-finn/go-cryptomarkets-bridge/binance"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
)

func main() {
	streamClient := binance.NewBinanceStreamClient()
	streamClient.Connect()
	binanceAPI := binance.NewBinanceAPI()
	binanceStream := binance.NewBinanceStreamAPI(streamClient)

	m := binance.NewBinanceOrderbookMaintainer(binanceAPI, binanceStream)

	symbol, err := domain.NewMarketSymbol("BTC", "USDT")
	if err != nil {
		panic(err)
	}

	orderbook := m.CreareOrderBook(symbol)

	// every second pring the orderbook last update id
	for {
		log.Println(orderbook.LastUpdateID)
		time.Sleep(1 * time.Second)
	}
}
