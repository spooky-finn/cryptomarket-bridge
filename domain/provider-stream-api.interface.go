package domain

type ProviderSyncAPI interface {
	OrderBookSnapshot(symbol *MarketSymbol, limit int) (*OrderBookSnapshot, error)
}

type CreareOrderBookResult struct {
	OrderBook *OrderBook
	Snapshot  *OrderBookSnapshot
	Err       error
}

type ProviderStreamAPI interface {
	GetOrderBook(marketSymbol *MarketSymbol) *CreareOrderBookResult
	DepthDiffStream(marketSymbol *MarketSymbol) (*Subscription[*OrderBookUpdate], error)
}
