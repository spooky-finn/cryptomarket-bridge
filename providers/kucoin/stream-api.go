package kucoin

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/Kucoin/kucoin-go-sdk"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
)

type KucoinStreamAPI struct {
	wc *kucoin.WebSocketClient

	messageBus <-chan *kucoin.WebSocketDownstreamMessage
	errorBus   <-chan error
}

func NewKucoinStreamAPI() *KucoinStreamAPI {
	api := NewKucoinAPI()
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

	s.messageBus = bus
	s.errorBus = errorBus
	return err
}

func (s *KucoinStreamAPI) DepthDiffStream(symbol *domain.MarketSymbol) (*domain.Subscription[DepthUpdateModel], error) {
	ch := make(chan DepthUpdateModel)
	topic := fmt.Sprintf("/market/level2:%s", strings.ToUpper(symbol.Join("-")))
	m := kucoin.NewSubscribeMessage(topic, false)
	err := s.wc.Subscribe(m)

	go func() {
		for msg := range s.messageBus {
			if msg.Type == kucoin.Message && msg.Topic == topic {
				data := &DepthUpdateModel{}

				if err := json.Unmarshal(msg.RawData, data); err != nil {
					log.Printf("kucoin: failed to unmarshal message: %s\n", err.Error())
				}

				ch <- *data
			}
		}
	}()

	return &domain.Subscription[DepthUpdateModel]{
		Stream: ch,
		Topic:  topic,
		Unsubscribe: func() {
			s.wc.Unsubscribe(kucoin.NewUnsubscribeMessage(topic, false))
		},
	}, err
}
