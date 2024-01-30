package domain

import (
	"errors"
	"log"
	"os"
)

type OrderBookStorage struct {
	storage map[string]map[string]*OrderBook
}

var logger = log.New(os.Stdout, "[orderbook-storage] ", log.LstdFlags)
var ErrOrderBookNotFound = errors.New("order book not found")
var ErrProviderNotFound = errors.New("provider not found")

func NewOrderBookStorage() *OrderBookStorage {
	return &OrderBookStorage{
		storage: make(map[string]map[string]*OrderBook),
	}
}

func (o *OrderBookStorage) Add(provider string, symbol *MarketSymbol, orderBook *OrderBook) {
	if _, ok := o.storage[provider]; !ok {
		o.storage[provider] = make(map[string]*OrderBook)
	}

	o.storage[provider][symbol.String()] = orderBook
}

func (o *OrderBookStorage) Get(provider string, symbol *MarketSymbol) (*OrderBook, error) {
	if _, ok := o.storage[provider]; !ok {
		return nil, ErrProviderNotFound
	}

	if _, ok := o.storage[provider][symbol.String()]; !ok {
		return nil, ErrOrderBookNotFound
	}

	return o.storage[provider][symbol.String()], nil
}

func (o *OrderBookStorage) OrderBookCount(provider string) int {
	if _, ok := o.storage[provider]; !ok {
		logger.Println("provider not found")
		return -1
	}

	return len(o.storage[provider])
}
