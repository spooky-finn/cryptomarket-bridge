package binance

import (
	"encoding/json"
	"fmt"

	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
)

var baseEndpoints = []string{
	"wss://stream.binance.com:9443/stream",
	"wss://stream.binance.com:443",
}

type BinanceStreamAPI struct {
	endpoint string
	client   *BinanceStreamClient
}

type DepthUpdateData struct {
	Event         string     `json:"e"`
	EventTime     int64      `json:"E"`
	Symbol        string     `json:"s"`
	FirstUpdateId int64      `json:"U"`
	FinalUpdateId int64      `json:"u"`
	Bids          [][]string `json:"b"`
	Asks          [][]string `json:"a"`
}

func NewBinanceStreamAPI(client *BinanceStreamClient) *BinanceStreamAPI {
	return &BinanceStreamAPI{
		endpoint: baseEndpoints[0],
		client:   client,
	}
}

func (bs *BinanceStreamAPI) DepthDiffStream(symbol *domain.MarketSymbol) *domain.Subscription[Message[DepthUpdateData]] {
	topic := fmt.Sprintf("%s@depth", symbol.Join(""))
	subscribtion := bs.client.Subscribe(topic)

	// unmarschal the message
	s := make(chan Message[DepthUpdateData])

	go func() {
		for msg := range subscribtion.Stream {
			var message Message[DepthUpdateData]
			err := json.Unmarshal(msg, &message)

			if err != nil {
				fmt.Printf("Error unmarshaling message: %s", err)
			}

			s <- message
		}
	}()

	return &domain.Subscription[Message[DepthUpdateData]]{
		Stream:      s,
		Unsubscribe: subscribtion.Unsubscribe,
		Topic:       topic,
	}
}
