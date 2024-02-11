package kucoin

import (
	"log"
	"sync"
	"time"

	"github.com/gammazero/deque"
	"github.com/spooky-finn/go-cryptomarkets-bridge/config"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
	i "github.com/spooky-finn/go-cryptomarkets-bridge/domain/interfaces"
	"github.com/spooky-finn/go-cryptomarkets-bridge/helpers"
)

// A manager class that is responsible for maintaining the orderbook of a market.
type OrderbookMaintainer struct {
	orderBook *domain.OrderBook
	syncAPI   *KucoinSyncAPI
	streamAPI *KucoinStreamAPI

	depthUpdateQueue deque.Deque[DepthUpdateModel]
	mu               sync.Mutex
	done             chan struct{}

	OutOfSequeceErrCount int
	wg                   sync.WaitGroup
}

func NewOrderBookMaintainer(stream *KucoinStreamAPI) *OrderbookMaintainer {
	return &OrderbookMaintainer{
		syncAPI:   stream.SyncAPI,
		streamAPI: stream,

		depthUpdateQueue: deque.Deque[DepthUpdateModel]{},
		mu:               sync.Mutex{},
	}
}

func (m *OrderbookMaintainer) CreareOrderBook(symbol *domain.MarketSymbol, limit int) *i.CreareOrderBookResult {
	log.Printf("creating orderbook for %s on kucoin", symbol.String())
	firstUpd := m.runStreamSubscriber(symbol)
	<-firstUpd

	if config.DebugMode {
		log.Printf("subscribed to depth update stream for %s on kucoin", symbol.String())
	}

	snapshot, err := m.syncAPI.OrderBookSnapshot(symbol, limit)
	log.Printf("got snapshot for %s on kucoin", symbol.String())
	if err != nil {
		return &i.CreareOrderBookResult{
			Err: err,
		}
	}

	orderbook := domain.NewOrderBook("kucoin", symbol, snapshot)
	m.orderBook = orderbook
	go m.queueReader()

	return &i.CreareOrderBookResult{
		OrderBook: orderbook,
		Snapshot:  snapshot,
		Err:       nil,
	}
}

func (m *OrderbookMaintainer) Stop() {
	close(m.done)
}

func (m *OrderbookMaintainer) queueReader() {
	for {
		m.mu.Lock()
		if m.depthUpdateQueue.Len() > 0 {
			update := m.depthUpdateQueue.PopFront()

			// case when orderbook more freash that update just skip it
			if update.SequenceEnd <= m.orderBook.LastUpdateID {
				m.mu.Unlock()
				continue
			}

			// case when update is older than last update
			if update.SequenceStart <= m.orderBook.LastUpdateID+1 && update.SequenceEnd >= m.orderBook.LastUpdateID {
				m.orderBook.ApplyUpdate(domain.NewOrderBookUpdate(
					update.Changes.Bids, update.Changes.Asks, update.SequenceEnd,
				))
				m.mu.Unlock()
				continue
			} else {
				m.OutOfSequeceErrCount++
				if m.OutOfSequeceErrCount > 10 {
					panic("out of sequence")
				}
			}
			m.mu.Unlock()
		} else {
			m.mu.Unlock()
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (m *OrderbookMaintainer) runStreamSubscriber(symbol *domain.MarketSymbol) <-chan struct{} {
	fistUpdteProcessed := false
	onFirstUpdateCh := make(chan struct{}, 1)

	subscription, err := m.streamAPI.DepthDiffStream(symbol)
	if err != nil {
		logger.Fatalf("error while subscribing to depth update stream  " + err.Error())
	}

	m.wg.Add(2)
	go func() {
		for {
			select {
			case <-m.done:
				return
			case update := <-subscription.Stream:
				m.mu.Lock()
				m.depthUpdateQueue.PushBack(*update)
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
