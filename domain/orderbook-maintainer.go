package domain

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gammazero/deque"
	"github.com/spooky-finn/cryptobridge/config"
	"github.com/spooky-finn/cryptobridge/helpers"
)

// A manager class that is responsible for maintaining (i.e. updating) the orderbook of a market.
type OrderbookMaintainer struct {
	syncAPI              ProviderSyncAPI
	streamAPI            ProviderStreamAPI
	depthUpdateValidator IDepthUpdateValidator

	orderBook        *OrderBook
	depthUpdateQueue deque.Deque[*OrderBookUpdate]
	mu               sync.Mutex

	OutOfSequeceErrCount int
	wg                   sync.WaitGroup
	done                 chan struct{}
}

func NewOrderBookMaintainer(
	streamAPI ProviderStreamAPI,
	syncAPI ProviderSyncAPI,
	depthUpdateValidator IDepthUpdateValidator,
) *OrderbookMaintainer {
	return &OrderbookMaintainer{
		syncAPI:              syncAPI,
		streamAPI:            streamAPI,
		depthUpdateValidator: depthUpdateValidator,

		depthUpdateQueue:     deque.Deque[*OrderBookUpdate]{},
		OutOfSequeceErrCount: 0,
		mu:                   sync.Mutex{},
	}
}

func (m *OrderbookMaintainer) CreareOrderBook(provider string, symbol *MarketSymbol, limit int) *CreareOrderBookResult {
	firstUpd := m.runStreamSubscriber(symbol)
	<-firstUpd

	if config.DebugMode {
		log.Printf("subscribed to depth update stream, Symbol=%s on Provider=%s", symbol.String(), provider)
	}

	snapshot, err := m.syncAPI.OrderBookSnapshot(symbol, limit)
	if err != nil {
		return &CreareOrderBookResult{
			Err: err,
		}
	}

	orderbook := NewOrderBook(provider, symbol, snapshot)
	m.orderBook = orderbook

	m.wg.Add(1)
	go m.queueReader()

	return &CreareOrderBookResult{
		OrderBook: orderbook,
		Snapshot:  snapshot,
		Err:       nil,
	}
}

func (m *OrderbookMaintainer) Stop() {
	close(m.done)
	m.wg.Wait()
}

func (m *OrderbookMaintainer) queueReader() {
	for {
		m.mu.Lock()
		if m.depthUpdateQueue.Len() > 0 {
			update := m.depthUpdateQueue.PopFront()

			err := m.depthUpdateValidator.IsValidUpd(update, m.orderBook.LastUpdateID)
			if err != nil {
				// TODO: process what to do when update is invalid.
				if m.depthUpdateValidator.IsErrOutOfSequece(err) {
					m.OutOfSequeceErrCount++
				}

				if m.OutOfSequeceErrCount > config.OrderBookOutOfSequeceErrThreshold {
					fmt.Printf("orderbook outdated and stopped. Provider=%s, Symbol=%s", m.orderBook.Provider, m.orderBook.Symbol.String())
					m.orderBook.StatusOutdated()

					m.mu.Unlock()
					m.Stop()
					m.wg.Done()
					return
				}

				m.mu.Unlock()
				continue
			}

			m.orderBook.ApplyUpdate(update)
			m.mu.Unlock()
		} else {
			m.mu.Unlock()
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (m *OrderbookMaintainer) runStreamSubscriber(symbol *MarketSymbol) <-chan struct{} {
	fistUpdteProcessed := false
	onFirstUpdateCh := make(chan struct{}, 1)

	subscription, err := m.streamAPI.DepthDiffStream(symbol)
	if err != nil {
		logger.Fatalf("error while subscribing to depth update stream  " + err.Error())
	}

	m.wg.Add(1)
	go func() {
		for {
			select {
			case <-m.done:
				m.wg.Done()
				return
			case update := <-subscription.Stream:
				m.mu.Lock()
				m.depthUpdateQueue.PushBack(update)
				m.mu.Unlock()

				if !fistUpdteProcessed {
					onFirstUpdateCh <- struct{}{}
					close(onFirstUpdateCh)
					fistUpdteProcessed = true
				}
			}
		}
	}()

	return helpers.WithLatestFrom(onFirstUpdateCh, TimeToEmtyChan(time.After(1*time.Second)))
}

func TimeToEmtyChan(in <-chan time.Time) chan struct{} {
	out := make(chan struct{}, 1)

	go func() {
		for range in {
			out <- struct{}{}
		}
	}()

	return out
}
