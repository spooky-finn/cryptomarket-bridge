package main

import (
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
	"github.com/spooky-finn/go-cryptomarkets-bridge/providers/kucoin"
)

func main() {
	godotenv.Load()

	// streamClient := binance.NewBinanceStreamClient()
	// streamClient.Connect()
	// binanceAPI := binance.NewBinanceAPI()
	// binanceStream := binance.NewBinanceStreamAPI(streamClient)

	// m := binance.NewBinanceOrderbookMaintainer(binanceAPI, binanceStream)

	symbol, err := domain.NewMarketSymbol("BTC", "USDT")
	if err != nil {
		panic(err)
	}

	// orderbook := m.CreareOrderBook(symbol)

	// // every second pring the orderbook last update id
	// for {
	// 	log.Println(orderbook.LastUpdateID)
	// 	time.Sleep(1 * time.Second)
	// }

	kucoinStream := kucoin.NewKucoinStreamAPI()

	kucoinStream.Connect()

	fmt.Println("connected to kucoin stream api")

	// subscribtion, err := kucoinStream.DepthDiffStream(symbol)
	// if err != nil {
	// 	panic(err)
	// }

	maintainer := kucoin.NewOrderbookMaintainer(kucoin.NewKucoinAPI(), kucoinStream)

	orderbook, err := maintainer.CreareOrderBook(symbol)
	if err != nil {
		panic(err)
	}

	for {
		log.Println(orderbook.LastUpdateID)
		time.Sleep(1 * time.Second)
	}

}
