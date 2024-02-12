package kucoin

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
)

type KucoinStreamAPI struct {
	WebSocket *KucoinStreamClient
	SyncAPI   *KucoinSyncAPI

	apiTimeout time.Duration
}

func NewKucoinStreamAPI(wc *KucoinStreamClient, syncAPI *KucoinSyncAPI) *KucoinStreamAPI {
	return &KucoinStreamAPI{
		WebSocket:  wc,
		SyncAPI:    syncAPI,
		apiTimeout: time.Second * 10,
	}
}

type DepthUpdateModel struct {
	Changes       OrderBookChanges `json:"changes"`
	SequenceEnd   int64            `json:"sequenceEnd"`
	SequenceStart int64            `json:"sequenceStart"`
	Symbol        string           `json:"symbol"`
	Time          int64            `json:"time"`
}

type OrderBookChanges struct {
	Asks [][]string `json:"asks"`
	Bids [][]string `json:"bids"`
}

type DethUpdateSubscribtion = *domain.Subscription[*domain.OrderBookUpdate]

func (s *KucoinStreamAPI) DepthDiffStream(symbol *domain.MarketSymbol) (*domain.Subscription[*domain.OrderBookUpdate], error) {
	topic := fmt.Sprintf("/market/level2:%s", strings.ToUpper(symbol.Join("-")))
	m := NewSubscribeMessage(topic, false)
	subscribtion, err := s.WebSocket.Subscribe(m)
	out := make(chan *domain.OrderBookUpdate)

	go func() {
		defer close(out)

		for msg := range subscribtion.Stream {
			message := &DepthUpdateModel{}

			err := json.Unmarshal(msg, &message)
			if err != nil {
				logger.Printf("Error unmarshaling message: %s", err)
			}

			out <- domain.NewOrderBookUpdate(
				message.Changes.Bids, message.Changes.Asks,
				message.SequenceStart, message.SequenceEnd,
				symbol,
			)
		}
	}()

	return &domain.Subscription[*domain.OrderBookUpdate]{
		Stream:      out,
		Topic:       topic,
		Unsubscribe: subscribtion.Unsubscribe,
	}, err
}

func (s *KucoinStreamAPI) GetOrderBook(symbol *domain.MarketSymbol, maxDepth int) *domain.CreareOrderBookResult {
	validator := &KucoinDepthUpdateValidator{}

	maintainer := domain.NewOrderBookMaintainer(s, s.SyncAPI, validator)

	result := maintainer.CreareOrderBook("kucoin", symbol, maxDepth)
	if result.Err != nil {
		return result
	}

	return &domain.CreareOrderBookResult{
		OrderBook: result.OrderBook,
		Snapshot:  result.Snapshot,
		Err:       nil,
	}
}
