package domain

import (
	"errors"
	"log"
	"os"
	"sync"
	"time"

	promclient "github.com/spooky-finn/cryptobridge/infrastructure/prometheus"
)

var logger = log.New(os.Stdout, "[orderbook-storage] ", log.LstdFlags)

var ErrOrderBookNotFound = errors.New("order book not found")
var ErrProviderNotFound = errors.New("provider not found")

type OrderBookStorage struct {
	storage map[string]map[string]*OrderBook
	mu      sync.RWMutex
}

func NewOrderBookStorage() *OrderBookStorage {
	s := &OrderBookStorage{
		storage: make(map[string]map[string]*OrderBook),
	}

	go s.runGC()
	return s
}

func (o *OrderBookStorage) Add(provider string, symbol *MarketSymbol, orderBook *OrderBook) {
	o.mu.Lock()
	if _, ok := o.storage[provider]; !ok {
		o.storage[provider] = make(map[string]*OrderBook)
	}

	o.storage[provider][symbol.String()] = orderBook
	o.mu.Unlock()

	promclient.BinanceOpenOrderBookGauge.Set(float64(o.OrderBookCount("binance")))
	promclient.KucoinOpenOrderBookGauge.Set(float64(o.OrderBookCount("kucoin")))
}

func (o *OrderBookStorage) Get(provider string, symbol *MarketSymbol) (*OrderBook, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if _, ok := o.storage[provider]; !ok {
		return nil, ErrProviderNotFound
	}

	if _, ok := o.storage[provider][symbol.String()]; !ok {
		return nil, ErrOrderBookNotFound
	}

	return o.storage[provider][symbol.String()], nil
}

func (o *OrderBookStorage) OrderBookCount(provider string) int {
	o.mu.RLock()
	defer o.mu.RUnlock()

	if _, ok := o.storage[provider]; !ok {
		return -1
	}

	return len(o.storage[provider])
}

func (o *OrderBookStorage) Remove(provider string, symbol *MarketSymbol) error {
	o.mu.Lock()
	if _, ok := o.storage[provider]; !ok {
		return errors.New("provider not found")
	}

	delete(o.storage[provider], symbol.String())
	o.mu.Unlock()

	promclient.BinanceOpenOrderBookGauge.Set(float64(o.OrderBookCount("binance")))
	promclient.KucoinOpenOrderBookGauge.Set(float64(o.OrderBookCount("kucoin")))
	return nil
}

func (o *OrderBookStorage) runGC() {
	for {
		o.mu.RLock()
		for provider, symbols := range o.storage {
			for symbol, orderBook := range symbols {
				if orderBook.status == OrderBookStatus_Oudated {
					logger.Printf("cleaning outdated order book: %s %s\n", provider, symbol)
					err := o.Remove(provider, orderBook.Symbol)

					if err != nil {
						logger.Printf("failed to remove outdated order book: %s %s, %s\n", provider, symbol, err)
					}
				}
			}

		}

		o.logStat()
		o.mu.RUnlock()
		<-time.After(10 * time.Second)
	}
}

// logStat the information about the order book count in the memeory
func (o *OrderBookStorage) logStat() {
	o.mu.RLock()
	defer o.mu.RUnlock()

	for provider, orderBookMap := range o.storage {
		logger.Printf(`order book count: %s: %d`, provider, len(orderBookMap))
	}
}
