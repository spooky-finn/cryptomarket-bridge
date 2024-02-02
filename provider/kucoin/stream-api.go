package kucoin

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Kucoin/kucoin-go-sdk"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain/interfaces"
)

const apiTimeout = time.Second * 5

type KucoinStreamAPI struct {
	wc           *kucoin.WebSocketClient
	sendingMutex sync.Mutex

	messageBus <-chan *kucoin.WebSocketDownstreamMessage
	errorBus   <-chan error
}

func NewKucoinStreamAPI(api *KucoinHttpAPI) *KucoinStreamAPI {
	wsConnOpts, err := api.WsConnOpts()

	if err != nil {
		panic("failed to get ws connection options: " + err.Error())
	}

	return &KucoinStreamAPI{
		wc: api.apiService.NewWebSocketClient(wsConnOpts),
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

func (s *KucoinStreamAPI) Connect() error {
	bus, errorBus, err := s.wc.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to kucoin websocket: %w", err)
	}

	logger.Println("connected to the kucoin stream websocket")
	s.messageBus = bus
	s.errorBus = errorBus
	return err
}

func (s *KucoinStreamAPI) DepthDiffStream(symbol *domain.MarketSymbol) (*interfaces.Subscription[DepthUpdateModel], error) {
	topic := fmt.Sprintf("/market/level2:%s", strings.ToUpper(symbol.Join("-")))
	m := kucoin.NewSubscribeMessage(topic, false)

	s.sendingMutex.Lock()
	defer s.sendingMutex.Unlock()

	err := s.wc.Subscribe(m)
	if err != nil {
		return nil, err
	}

	ch := s.startMsgPicker(topic)

	return &interfaces.Subscription[DepthUpdateModel]{
		Stream: ch,
		Topic:  topic,
		Unsubscribe: func() {
			s.wc.Unsubscribe(kucoin.NewUnsubscribeMessage(topic, false))
		},
	}, err
}

func (s *KucoinStreamAPI) startMsgPicker(topic string) chan DepthUpdateModel {
	ch := make(chan DepthUpdateModel)

	go func() {
		for {
			select {
			case <-time.After(apiTimeout):
				logger.Printf("error while reading from the websocket. Stream stopped for topic: %s\n", topic)
				return

			case msg := <-s.messageBus:
				if msg.Type == kucoin.Message && msg.Topic == topic {
					data := &DepthUpdateModel{}

					if err := json.Unmarshal(msg.RawData, data); err != nil {
						logger.Printf("kucoin: failed to unmarshal message: %s\n", err.Error())
					}

					ch <- *data
				}
			}
		}
	}()
	return ch
}

func (s *KucoinStreamAPI) GetOrderBook(symbol *domain.MarketSymbol, maxDepth int) *interfaces.CreareOrderBookResult {
	om := NewOrderbookMaintainer(NewKucoinHttpAPI(), s)
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
