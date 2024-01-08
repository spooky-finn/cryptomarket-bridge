package binance

import (
	"log"
	"time"

	"github.com/gammazero/deque"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
)

type BinanceOrderbookMaintainer struct {
	api    *BinanceAPI
	stream *BinanceStreamAPI

	depthUpdateQueue deque.Deque[Message[DepthUpdateData]]
}

func NewBinanceOrderbookMaintainer(api *BinanceAPI, stream *BinanceStreamAPI) *BinanceOrderbookMaintainer {
	return &BinanceOrderbookMaintainer{
		api:    api,
		stream: stream,

		depthUpdateQueue: deque.Deque[Message[DepthUpdateData]]{},
	}
}

func (m *BinanceOrderbookMaintainer) CreareOrderBook(symbol *domain.MarketSymbol) *domain.OrderBook {
	firstUpd := m.subscribeDepthUpdate(symbol)
	<-firstUpd

	snapshot, err := m.api.OrderBookSnapshot(symbol, 5000)
	if err != nil {
		panic(err)
	}

	orderbook := domain.NewOrderBook("binance", symbol, snapshot)

	go m.updateSelector(orderbook)

	return orderbook
}

func (m *BinanceOrderbookMaintainer) updateSelector(orderbook *domain.OrderBook) {
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
				orderbook.ApplyUpdate(&domain.OrderBookUpdate{
					Asks:         update.Data.Asks,
					Bids:         update.Data.Bids,
					LastUpdateID: update.Data.FinalUpdateId,
				})
				firstUpdateApplied = true
				continue
			}

			if firstUpdateApplied {
				// While listening to the stream, each new event's U should be equal to the previous event's u+1
				if update.Data.FirstUpdateId != orderbook.LastUpdateID+1 {
					log.Fatalf("binance: orderbook maintainer: not sequential update: %d != %d+1\n", update.Data.FirstUpdateId, orderbook.LastUpdateID)
				}

				orderbook.ApplyUpdate(&domain.OrderBookUpdate{
					Asks:         update.Data.Asks,
					Bids:         update.Data.Bids,
					LastUpdateID: update.Data.FinalUpdateId,
				})
			}

		} else {
			time.Sleep(100 * time.Millisecond)
		}

	}
}

// Return buffered channel which triggers on first update
func (m *BinanceOrderbookMaintainer) subscribeDepthUpdate(symbol *domain.MarketSymbol) chan bool {
	counter := 0
	firstUpdate := make(chan bool, 1)
	subscribtion := m.stream.DepthDiffStream(symbol)

	go func() {
		for {
			select {
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
