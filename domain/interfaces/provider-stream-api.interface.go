package interfaces

import "github.com/spooky-finn/go-cryptomarkets-bridge/domain"

type ProviderHttpAPI interface {
	OrderBookSnapshot(symbol *domain.MarketSymbol, limit int) (*domain.OrderBookSnapshot, error)
}

type CreareOrderBookResult struct {
	OrderBook *domain.OrderBook
	Snapshot  *domain.OrderBookSnapshot
	Err       error
}

type ProviderStreamAPI interface {
	GetOrderBook(marketSymbol *domain.MarketSymbol) *CreareOrderBookResult
}
