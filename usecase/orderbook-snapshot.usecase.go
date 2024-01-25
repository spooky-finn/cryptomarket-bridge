package usecase

import (
	"fmt"
	"log"

	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain/interfaces"
)

const STARTING = "starting"

type OrderBookSnapshotUseCase struct {
	apiResolver interfaces.ApiResolver
	storage     *domain.OrderBookStorage

	waitingRoom map[string]string
}

func NewOrderBookSnapshotUseCase(
	apiResolver interfaces.ApiResolver,
) *OrderBookSnapshotUseCase {
	return &OrderBookSnapshotUseCase{
		apiResolver: apiResolver,
		storage:     domain.NewOrderBookStorage(),

		waitingRoom: make(map[string]string),
	}
}

// GetOrderBookSnapshot returns the orderbook snapshot from the runtime storage or from provider api.
func (o *OrderBookSnapshotUseCase) GetOrderBookSnapshot(
	provider string, symbol *domain.MarketSymbol, limit int,
) (*domain.OrderBookSnapshot, error) {
	waitingRoomKey := o.getWaitingRoomKey(provider, symbol)
	if _, ok := o.waitingRoom[waitingRoomKey]; ok {
		return o.apiResolver.HttpApi(provider).OrderBookSnapshot(symbol, limit)
	}

	orderbook, err := o.storage.Get(provider, symbol)
	if err != nil {
		go o.createOrderBook(provider, symbol)
		return o.apiResolver.HttpApi(provider).OrderBookSnapshot(symbol, limit)
	}

	snapshot := orderbook.TakeSnapshot(limit)
	return snapshot, nil
}

func (o *OrderBookSnapshotUseCase) createOrderBook(
	provider string, symbol *domain.MarketSymbol,
) {
	waitingRoomKey := o.getWaitingRoomKey(provider, symbol)
	o.waitingRoom[waitingRoomKey] = STARTING

	result := o.apiResolver.StreamApi(provider).GetOrderBook(symbol)
	if result.Err != nil {
		log.Fatalf("Failed to create orderbook for %s. Provider=%s. Err=%s", symbol.String(), provider, result.Err)
		return
	}

	o.storage.Add(provider, symbol, result.OrderBook)
	delete(o.waitingRoom, waitingRoomKey)
	log.Printf("Orderbook snapshot for %s is added for to the runtime storage. Provider=%s", symbol.String(), provider)
}

func (o *OrderBookSnapshotUseCase) getWaitingRoomKey(provider string, symbol *domain.MarketSymbol) string {
	return fmt.Sprintf("%s-%s", provider, symbol.String())
}
