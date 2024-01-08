package binance

import (
	"log"
	"time"

	"github.com/gammazero/deque"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
)

type OrderbookMaintainer struct {
	api    *BinanceAPI
	stream *BinanceStreamAPI

	depthUpdateQueue deque.Deque[Message[DepthUpdateData]]
	done             chan struct{}
}

func NewOrderbookMaintainer(api *BinanceAPI, stream *BinanceStreamAPI) *OrderbookMaintainer {
	return &OrderbookMaintainer{
		api:    api,
		stream: stream,

		depthUpdateQueue: deque.Deque[Message[DepthUpdateData]]{},
	}
}

func (m *OrderbookMaintainer) CreareOrderBook(symbol *domain.MarketSymbol) (*domain.OrderBook, error) {
	firstUpd := m.subscribeDepthUpdate(symbol)
	<-firstUpd

	snapshot, err := m.api.OrderBookSnapshot(symbol, 5000)
	if err != nil {
		return nil, err
	}

	orderbook := domain.NewOrderBook("binance", symbol, snapshot)

	go m.updateSelector(orderbook)
	return orderbook, nil
}

func (m *OrderbookMaintainer) updateSelector(orderbook *domain.OrderBook) {
	firstUpdateApplied := false

	for {
		if m.depthUpdateQueue.Len() > 0 {
			update := m.depthUpdateQueue.PopFront()

			// Drop any event where u is <= lastUpdateId in the snapshot
			if update.Data.FinalUpdateId <= orderbook.LastUpdateID {
				continue
			}

			// The first processed event should have U <= lastUpdateId+1 AND u >= lastUpdateId+1
			if !firstUpdateApplied &&
				(update.Data.FirstUpdateId <= orderbook.LastUpdateID+1 && update.Data.FinalUpdateId >= orderbook.LastUpdateID+1) {
				orderbook.ApplyUpdate(domain.NewOrderBookUpdate(
					update.Data.Bids, update.Data.Asks, update.Data.FinalUpdateId,
				))
				firstUpdateApplied = true
				continue
			}

			if firstUpdateApplied {
				// While listening to the stream, each new event's U should be equal to the previous event's u+1
				if update.Data.FirstUpdateId != orderbook.LastUpdateID+1 {
					log.Fatalf("binance: orderbook maintainer: not sequential update: %d != %d+1\n", update.Data.FirstUpdateId, orderbook.LastUpdateID)
				}

				orderbook.ApplyUpdate(domain.NewOrderBookUpdate(
					update.Data.Bids, update.Data.Asks, update.Data.FinalUpdateId,
				))
			}
		} else {
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func Stop(m *OrderbookMaintainer) {
	close(m.done)
	_ = m.api.conn.Close()
}

// Return buffered channel which triggers on first update
func (m *OrderbookMaintainer) subscribeDepthUpdate(symbol *domain.MarketSymbol) <-chan bool {
	counter := 0
	firstUpdate := make(chan bool, 1)
	subscribtion := m.stream.DepthDiffStream(symbol)

	go func() {
		for {
			select {
			case <-m.done:
				return
			case update := <-subscribtion.Stream:
				m.depthUpdateQueue.PushBack(update)

				if counter == 0 {
					firstUpdate <- true
				}

				counter++
			}
		}
	}()

	return firstUpdate
}
