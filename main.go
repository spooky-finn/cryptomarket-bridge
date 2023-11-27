package main

import (
	"fmt"

	"github.com/spooky-finn/go-cryptomarkets-bridge/binance"
)

var (
	connected = make(chan bool, 1)
	block     = make(chan bool, 1)
)

func main() {

	client := binance.NewBinanceClient()

	client.Connect()

	ch := client.Subscribe("btcusdt@depth@100ms")

	println("subscribed")
	for {
		select {
		case msg := <-ch:
			m := msg.(*binance.Message[binance.DepthUpdateData])

			fmt.Printf("Received message: %+v\n", m.Data)
		}
	}
	<-block
}
