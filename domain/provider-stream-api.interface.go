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
	GetOrderBook(marketSymbol *MarketSymbol, maxDepth int) *CreareOrderBookResult
	DepthDiffStream(marketSymbol *MarketSymbol) (*Subscription[*OrderBookUpdate], error)
}
