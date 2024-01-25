package usecase

import (
	"log"

	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain/interfaces"
)

type OrderBookSnapshotUseCase struct {
	apiResolver interfaces.ApiResolver
	storage     *domain.OrderBookStorage
}

func NewOrderBookSnapshotUseCase(
	apiResolver interfaces.ApiResolver,
) *OrderBookSnapshotUseCase {
	return &OrderBookSnapshotUseCase{
		apiResolver: apiResolver,
		storage:     domain.NewOrderBookStorage(),
	}
}

func (o *OrderBookSnapshotUseCase) GetOrderBookSnapshot(
	provider string, symbol *domain.MarketSymbol, limit int,
) (*domain.OrderBookSnapshot, error) {
	orderbook, err := o.storage.Get(provider, symbol)
	if err != nil {
		return o.createOrderBook(provider, symbol)
	}

	snapshot := orderbook.TakeSnapshot(limit)
	return snapshot, nil
}

func (o *OrderBookSnapshotUseCase) createOrderBook(
	provider string, symbol *domain.MarketSymbol,
) (*domain.OrderBookSnapshot, error) {
	result := o.apiResolver.ByProvider(provider).GetOrderBook(symbol)
	if result.Err != nil {
		return nil, result.Err
	}

	o.storage.Add(provider, symbol, result.OrderBook)
	log.Printf("Orderbook snapshot for %s is added for to the runtime storage. Provider=%s", symbol.String(), provider)
	return result.Snapshot, nil
}
