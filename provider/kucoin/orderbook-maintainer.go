package kucoin

import (
	"log"
	"strconv"
	"sync"

	"github.com/gammazero/deque"
	"github.com/spooky-finn/go-cryptomarkets-bridge/config"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
	i "github.com/spooky-finn/go-cryptomarkets-bridge/domain/interfaces"
)

// A manager class that is responsible for maintaining the orderbook of a market.
type OrderbookMaintainer struct {
	ob        *domain.OrderBook
	syncAPI   *KucoinSyncAPI
	streamAPI *KucoinStreamAPI

	depthUpdateQueue deque.Deque[DepthUpdateModel]
	mu               sync.Mutex
	done             chan struct{}

	firstUpdateApplied   bool
	OutOfSequeceErrCount int
}

func NewOrderBookMaintainer(stream *KucoinStreamAPI) *OrderbookMaintainer {
	return &OrderbookMaintainer{
		syncAPI:   stream.syncAPI,
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
	m.ob = orderbook
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

			// FIXME: WTF WHY THIS EMTY UPDATES
			// Drop any event where u is <= lastUpdateId in the snapshot

			//  to the local snapshot to ensure that sequenceStart(new)<=sequenceEnd+1(old) and sequenceEnd(new) > sequenceEnd(old).
			if !m.firstUpdateApplied &&
				(update.SequenceStart <= m.ob.LastUpdateID+1 && update.SequenceEnd >= m.ob.LastUpdateID) {
				m.applyFirstUpdate(m.ob, &update)
				m.mu.Unlock()
				continue
			}

			if m.firstUpdateApplied {
				// TODO: add seq
				//  to the local snapshot to ensure that sequenceStart(new)<=sequenceEnd+1(old) and sequenceEnd(new) > sequenceEnd(old).
				m.ob.ApplyUpdate(domain.NewOrderBookUpdate(
					update.Changes.Bids, update.Changes.Asks, update.SequenceEnd,
				))
			}
		}
		m.mu.Unlock()
	}
}

func (m *OrderbookMaintainer) applyFirstUpdate(orderbook *domain.OrderBook, update *DepthUpdateModel) {
	asks := m.selectFromUpdatEventsLaterThan(update.Changes.Asks, orderbook.LastUpdateID)
	bids := m.selectFromUpdatEventsLaterThan(update.Changes.Bids, orderbook.LastUpdateID)

	orderbook.ApplyUpdate(domain.NewOrderBookUpdate(
		bids, asks, update.SequenceEnd,
	))
	m.firstUpdateApplied = true
}

func (m *OrderbookMaintainer) selectFromUpdatEventsLaterThan(depth [][]string, lastUpdateID int64) [][]string {
	new := make([][]string, 0)

	for _, ask := range depth {
		seq, err := strconv.ParseInt(ask[2], 10, 64)
		if err != nil {
			panic(err)
		}

		if seq > lastUpdateID {
			new = append(new, ask)
		}
	}

	return new
}

func (m *OrderbookMaintainer) runStreamSubscriber(symbol *domain.MarketSymbol) <-chan struct{} {
	fistUpdteProcessed := false
	onFirstUpdateCh := make(chan struct{}, 1)

	subscription, err := m.streamAPI.DepthDiffStream(symbol)
	if err != nil {
		logger.Fatalf("error while subscribing to depth update stream  " + err.Error())
	}

	go func() {
		for {
			select {
			case <-m.done:
				return
			case update := <-subscription.Stream:
				m.mu.Lock()
				m.depthUpdateQueue.PushBack(*update)

				if !fistUpdteProcessed {
					onFirstUpdateCh <- struct{}{}
					fistUpdteProcessed = true
					close(onFirstUpdateCh)
				}
				m.mu.Unlock()
			}
		}
	}()

	return onFirstUpdateCh
}
