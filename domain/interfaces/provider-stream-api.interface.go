package interfaces

import "github.com/spooky-finn/go-cryptomarkets-bridge/domain"

type CreareOrderBookResult struct {
	OrderBook *domain.OrderBook
	Snapshot  *domain.OrderBookSnapshot
	Err       error
}

type ProviderStreamAPI interface {
	GetOrderBook(marketSymbol *domain.MarketSymbol) *CreareOrderBookResult
}
