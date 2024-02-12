package binance

import (
	"encoding/json"
	"fmt"

	"github.com/spooky-finn/cryptobridge/domain"
)

var baseEndpoints = []string{
	"wss://stream.binance.com:9443/stream",
	"wss://stream.binance.com:443",
}

type BinanceStreamAPI struct {
	endpoint     string
	streamClient *BinanceStreamClient
	syncAPI      *BinanceSyncAPI
}

type DethUpdateSubscribtion = domain.Subscription[Message[DepthUpdateData]]

type DepthUpdateData struct {
	Event         string     `json:"e"`
	EventTime     int64      `json:"E"`
	Symbol        string     `json:"s"`
	FirstUpdateId int64      `json:"U"`
	FinalUpdateId int64      `json:"u"`
	Bids          [][]string `json:"b"`
	Asks          [][]string `json:"a"`
}

func NewBinanceStreamAPI(client *BinanceStreamClient, syncAPI *BinanceSyncAPI) *BinanceStreamAPI {
	return &BinanceStreamAPI{
		endpoint:     baseEndpoints[0],
		streamClient: client,
		syncAPI:      syncAPI,
	}
}

func (bs *BinanceStreamAPI) DepthDiffStream(symbol *domain.MarketSymbol) (*domain.Subscription[*domain.OrderBookUpdate], error) {
	topic := fmt.Sprintf("%s@depth", symbol.Join(""))
	subscribtion, err := bs.streamClient.Subscribe(topic)
	if err != nil {
		return nil, err
	}
	s := make(chan *domain.OrderBookUpdate)

	go func() {
		defer close(s)

		for msg := range subscribtion.Stream {
			var message Message[DepthUpdateData]
			err := json.Unmarshal(msg, &message)

			if err != nil {
				fmt.Printf("Error unmarshaling message: %s", err)
			}

			update := domain.NewOrderBookUpdate(
				message.Data.Bids, message.Data.Asks,
				message.Data.FirstUpdateId, message.Data.FinalUpdateId,
				symbol,
			)
			s <- update
		}
	}()

	return &domain.Subscription[*domain.OrderBookUpdate]{
		Stream:      s,
		Unsubscribe: subscribtion.Unsubscribe,
		Topic:       topic,
	}, nil
}

func (bs *BinanceStreamAPI) GetOrderBook(symbol *domain.MarketSymbol) *domain.CreareOrderBookResult {
	validator := &BinanceDepthUpdateValidator{}
	maintainer := domain.NewOrderBookMaintainer(bs, bs.syncAPI, validator)

	result := maintainer.CreareOrderBook("binance", symbol)
	if result.Err != nil {
		return result
	}

	return result
}
