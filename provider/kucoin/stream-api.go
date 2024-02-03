package kucoin

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain/interfaces"
)

type KucoinStreamAPI struct {
	wc          *KucoinStreamClient
	syncAPI     *KucoinSyncAPI
	apiTinmeout time.Duration
}

func NewKucoinStreamAPI(wc *KucoinStreamClient, syncAPI *KucoinSyncAPI) *KucoinStreamAPI {
	return &KucoinStreamAPI{
		wc:          wc,
		syncAPI:     syncAPI,
		apiTinmeout: time.Second * 5,
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

func (s *KucoinStreamAPI) DepthDiffStream(symbol *domain.MarketSymbol) (*interfaces.Subscription[*DepthUpdateModel], error) {
	topic := fmt.Sprintf("/market/level2:%s", strings.ToUpper(symbol.Join("-")))
	m := NewSubscribeMessage(topic, false)
	subscribtion, err := s.wc.Subscribe(m)
	out := make(chan *DepthUpdateModel)

	go func() {
		defer close(out)

		for msg := range subscribtion.Stream {
			message := &DepthUpdateModel{}
			err := json.Unmarshal(msg, &message)
			if err != nil {
				fmt.Printf("Error unmarshaling message: %s", err)
			}
			out <- message
		}
	}()

	return &interfaces.Subscription[*DepthUpdateModel]{
		Stream:      out,
		Topic:       topic,
		Unsubscribe: subscribtion.Unsubscribe,
	}, err
}

func (s *KucoinStreamAPI) GetOrderBook(symbol *domain.MarketSymbol, maxDepth int) *interfaces.CreareOrderBookResult {
	om := NewOrderbookMaintainer(s)

	result := om.CreareOrderBook(symbol, maxDepth)
	if result.Err != nil {
		return result
	}

	return &interfaces.CreareOrderBookResult{
		OrderBook: result.OrderBook,
		Snapshot:  result.Snapshot,
		Err:       nil,
	}
}
