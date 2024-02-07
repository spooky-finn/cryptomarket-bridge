package binance

import (
	"fmt"
	"sync"

	"github.com/gammazero/deque"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain/interfaces"
)

const outOfSeqUpdatesLimit = 10

type OrderBookMaintainer struct {
	syncAPI   *BinanceAPI
	streamAPI *BinanceStreamAPI

	depthUpdateQueue deque.Deque[Message[DepthUpdateData]]
	mu               *sync.RWMutex

	done                 chan struct{}
	firstUpdateApplied   bool
	OutOfSequeceErrCount int
}

func NewOrderBookMaintainer(stream *BinanceStreamAPI) *OrderBookMaintainer {
	return &OrderBookMaintainer{
		syncAPI:   stream.syncAPI,
		streamAPI: stream,

		mu:               &sync.RWMutex{},
		depthUpdateQueue: deque.Deque[Message[DepthUpdateData]]{},
	}
}

func (m *OrderBookMaintainer) CreareOrderBook(symbol *domain.MarketSymbol, maxDepth int) *interfaces.CreareOrderBookResult {
	firstUpd, err := m.streamSubscriber(symbol)
	if err != nil {
		return &interfaces.CreareOrderBookResult{
			Err: err,
		}

	}

	<-firstUpd
	snapshot, err := m.syncAPI.OrderBookSnapshot(symbol, maxDepth)
	if err != nil {
		return &interfaces.CreareOrderBookResult{
			Err: err,
		}
	}

	orderbook := domain.NewOrderBook("binance", symbol, snapshot)
	go m.queueReader(orderbook)

	return &interfaces.CreareOrderBookResult{
		OrderBook: orderbook,
		Snapshot:  snapshot,
		Err:       nil,
	}
}

func (m *OrderBookMaintainer) queueReader(orderbook *domain.OrderBook) {
	for {
		m.mu.Lock()
		if m.depthUpdateQueue.Len() > 0 {
			update := m.depthUpdateQueue.PopFront()

			if err := m.appyUpdate(orderbook, &update.Data); err != nil {
				logger.Printf("binance: orderbook maintainer: fail to apply update: %s\n", err)
				orderbook.StatusOutdated()
			}

			m.mu.Unlock()
			return
		} else {
			m.mu.Unlock()
		}
	}
}

func (m *OrderBookMaintainer) appyUpdate(orderbook *domain.OrderBook, update *DepthUpdateData) error {
	upd := domain.NewOrderBookUpdate(
		update.Bids, update.Asks, update.FinalUpdateId,
	)
	// Drop any event where u is <= lastUpdateId in the snapshot
	if update.FinalUpdateId <= orderbook.LastUpdateID {
		return nil
	}

	// The first processed event should have U <= lastUpdateId+1 AND u >= lastUpdateId+1
	if !m.firstUpdateApplied &&
		(update.FirstUpdateId <= orderbook.LastUpdateID+1 && update.FinalUpdateId >= orderbook.LastUpdateID+1) {
		orderbook.ApplyUpdate(upd)
		m.firstUpdateApplied = true
		return nil
	}

	if m.firstUpdateApplied {
		// While listening to the stream, each new event's U should be equal to the previous event's u+1
		if update.FirstUpdateId != orderbook.LastUpdateID+1 {
			m.OutOfSequeceErrCount++
			logger.Printf("binance: orderbook maintainer: droped update: %d <= %d\n", update.FinalUpdateId, orderbook.LastUpdateID)

			// if outOfSeqUpdatesLimit is reached, stop the maintainer
			if m.OutOfSequeceErrCount > outOfSeqUpdatesLimit {
				return fmt.Errorf("binance: orderbook maintainer: out of sequence updates limit reached")
			}
			return nil
		}

		orderbook.ApplyUpdate(upd)
	}

	return nil
}

func Stop(m *OrderBookMaintainer) {
	close(m.done)
	_ = m.syncAPI.conn.Close()
}

// Return channel which triggers on first update
func (m *OrderBookMaintainer) streamSubscriber(symbol *domain.MarketSymbol) (<-chan struct{}, error) {
	onFirstUpdate := make(chan struct{}, 1)
	subscribtion, err := m.streamAPI.DepthDiffStream(symbol)

	if err != nil {
		return nil, err
	}

	go func() {

		for {
			select {
			case <-m.done:
				return // stop the loop
			case update := <-subscribtion.Stream:
				m.mu.Lock()
				m.depthUpdateQueue.PushBack(update)
				m.mu.Unlock()

				if !m.firstUpdateApplied {
					onFirstUpdate <- struct{}{}
					close(onFirstUpdate)
				}
			}
		}
	}()

	return onFirstUpdate, nil
}
