package domain

import (
	"errors"
	"log"
	"os"
	"time"

	promclient "github.com/spooky-finn/go-cryptomarkets-bridge/infrastructure/prometheus"
)

var logger = log.New(os.Stdout, "[orderbook-storage] ", log.LstdFlags)

var ErrOrderBookNotFound = errors.New("order book not found")
var ErrProviderNotFound = errors.New("provider not found")

type OrderBookStorage struct {
	storage map[string]map[string]*OrderBook
}

func NewOrderBookStorage() *OrderBookStorage {
	s := &OrderBookStorage{
		storage: make(map[string]map[string]*OrderBook),
	}

	go s.startOutdetedOrderBookCleaner()
	return s
}

func (o *OrderBookStorage) Add(provider string, symbol *MarketSymbol, orderBook *OrderBook) {
	if _, ok := o.storage[provider]; !ok {
		o.storage[provider] = make(map[string]*OrderBook)
	}

	o.storage[provider][symbol.String()] = orderBook

	promclient.BinanceOpenOrderBookGauge.Set(float64(o.OrderBookCount("binance")))
	promclient.KucoinOpenOrderBookGauge.Set(float64(o.OrderBookCount("kucoin")))
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

func (o *OrderBookStorage) Remove(provider string, symbol *MarketSymbol) {
	if _, ok := o.storage[provider]; !ok {
		return
	}

	delete(o.storage[provider], symbol.String())

	promclient.BinanceOpenOrderBookGauge.Set(float64(o.OrderBookCount("binance")))
	promclient.KucoinOpenOrderBookGauge.Set(float64(o.OrderBookCount("kucoin")))
}

func (o *OrderBookStorage) startOutdetedOrderBookCleaner() {
	for {
		for provider, symbols := range o.storage {
			for symbol, orderBook := range symbols {
				if orderBook.status == OrderBookStatus_Oudated {
					logger.Printf("cleaning outdated order book: %s %s\n", provider, symbol)
					o.Remove(provider, orderBook.Symbol)
				}
			}
		}

		<-time.After(10 * time.Second)
	}
}
