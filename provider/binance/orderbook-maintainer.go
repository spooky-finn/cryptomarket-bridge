package binance

import (
	"fmt"
	"sync"
	"time"

	"github.com/gammazero/deque"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain/interfaces"
)

type status string

const (
	running status = "running"
	stopped status = "died"
)

const outOfSeqUpdatesLimit = 10

type OrderbookMaintainer struct {
	status    status
	syncAPI   *BinanceAPI
	streamAPI *BinanceStreamAPI

	depthUpdateQueue deque.Deque[Message[DepthUpdateData]]
	mu               *sync.RWMutex

	done                 chan struct{}
	firstUpdateApplied   bool
	OutOfSequeceErrCount int
}

func NewOrderBookMaintainer(stream *BinanceStreamAPI) *OrderbookMaintainer {
	return &OrderbookMaintainer{
		status:    running,
		syncAPI:   stream.syncAPI,
		streamAPI: stream,

		mu:               &sync.RWMutex{},
		depthUpdateQueue: deque.Deque[Message[DepthUpdateData]]{},
	}
}

func (m *OrderbookMaintainer) CreareOrderBook(symbol *domain.MarketSymbol, maxDepth int) *interfaces.CreareOrderBookResult {
	firstUpd := m.subscribeDepthUpdateStream(symbol)
	<-firstUpd

	snapshot, err := m.syncAPI.OrderBookSnapshot(symbol, maxDepth)
	if err != nil {
		return &interfaces.CreareOrderBookResult{
			Err: err,
		}
	}

	orderbook := domain.NewOrderBook("binance", symbol, snapshot)
	go m.startMsgPicker(orderbook)

	return &interfaces.CreareOrderBookResult{
		OrderBook: orderbook,
		Snapshot:  snapshot,
		Err:       nil,
	}
}

func (m *OrderbookMaintainer) startMsgPicker(orderbook *domain.OrderBook) {
	for {
		m.mu.Lock()
		if m.depthUpdateQueue.Len() > 0 {
			update := m.depthUpdateQueue.PopFront()
			if err := m.appyUpdate(orderbook, &update.Data); err != nil {
				logger.Printf("binance: orderbook maintainer: %s\n", err)
				m.status = stopped
			}

			m.mu.Unlock()
			return
		} else {
			m.mu.Unlock()

			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (m *OrderbookMaintainer) appyUpdate(orderbook *domain.OrderBook, update *DepthUpdateData) error {
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

func Stop(m *OrderbookMaintainer) {
	close(m.done)
	_ = m.syncAPI.conn.Close()
}

// Return buffered channel which triggers on first update
func (m *OrderbookMaintainer) subscribeDepthUpdateStream(symbol *domain.MarketSymbol) <-chan struct{} {
	counter := 0
	firstUpdate := make(chan struct{}, 1)
	subscribtion := m.streamAPI.DepthDiffStream(symbol)

	go func() {
		for {
			select {
			case <-m.done:
				return
			case update := <-subscribtion.Stream:
				m.mu.Lock()
				m.depthUpdateQueue.PushBack(update)
				m.mu.Unlock()

				if counter == 0 {
					firstUpdate <- struct{}{}
				}

				counter++
			}
		}
	}()

	return firstUpdate
}
