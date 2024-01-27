package usecase

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain/interfaces"
)

const STARTING = "starting"

var logger = log.New(os.Stdout, "[orderbook-snapshot-usecase] ", log.LstdFlags)

type OrderBookSnapshotUseCase struct {
	apiResolver interfaces.ApiResolver
	storage     *domain.OrderBookStorage

	waitingRoom map[string]string
	mu          sync.RWMutex
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
	o.mu.Lock()
	defer o.mu.Unlock()

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
	o.mu.Lock()
	defer o.mu.Unlock()

	waitingRoomKey := o.getWaitingRoomKey(provider, symbol)
	o.waitingRoom[waitingRoomKey] = STARTING

	result := o.apiResolver.StreamApi(provider).GetOrderBook(symbol)
	if result.Err != nil {
		return
	}

	o.storage.Add(provider, symbol, result.OrderBook)
	delete(o.waitingRoom, waitingRoomKey)
	logger.Printf("Orderbook snapshot for %s is added for to the runtime storage. Provider=%s", symbol.String(), provider)
}

func (o *OrderBookSnapshotUseCase) getWaitingRoomKey(provider string, symbol *domain.MarketSymbol) string {
	return fmt.Sprintf("%s-%s", provider, symbol.String())
}
