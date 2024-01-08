package kucoin

import (
	"strconv"
	"time"

	"github.com/gammazero/deque"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
)

type OrderbookMaintainer struct {
	api    *KucoinAPI
	stream *KucoinStreamAPI

	depthUpdateQueue deque.Deque[DepthUpdateModel]
	done             chan struct{}
}

func NewOrderbookMaintainer(api *KucoinAPI, stream *KucoinStreamAPI) *OrderbookMaintainer {
	return &OrderbookMaintainer{
		api:    api,
		stream: stream,

		depthUpdateQueue: deque.Deque[DepthUpdateModel]{},
	}
}

func (m *OrderbookMaintainer) CreareOrderBook(symbol *domain.MarketSymbol) (*domain.OrderBook, error) {
	firstUpd := m.subscribeDepthUpdate(symbol)
	<-firstUpd

	snapshot, err := m.api.OrderBookSnapshot(symbol)
	if err != nil {
		return nil, err
	}

	orderbook := domain.NewOrderBook("kucoin", symbol, snapshot)

	go m.updateSelector(orderbook)
	return orderbook, nil
}

func (m *OrderbookMaintainer) Stop() {
	m.stream.wc.Stop()
	close(m.done)
}

func (m *OrderbookMaintainer) updateSelector(orderbook *domain.OrderBook) {
	firstUpdateApplied := false

	for {
		if m.depthUpdateQueue.Len() > 0 {
			update := m.depthUpdateQueue.PopFront()

			// FIXME: WTF WHY THIS EMTY UPDATES
			// fmt.Println("update", update)
			// Drop any event where u is <= lastUpdateId in the snapshot
			if update.SequenceStart <= orderbook.LastUpdateID {
				continue
			}

			// print the last update id and update sequence Ends
			// fmt.Println("last update id", orderbook.LastUpdateID, "update sequence end", update.SequenceStart)

			//  to the local snapshot to ensure that sequenceStart(new)<=sequenceEnd+1(old) and sequenceEnd(new) > sequenceEnd(old).
			if !firstUpdateApplied &&
				(update.SequenceStart <= orderbook.LastUpdateID+1 && update.SequenceEnd >= orderbook.LastUpdateID) {
				asks := make([][]string, 0)
				bids := make([][]string, 0)

				for _, ask := range update.Changes.Asks {
					seq, err := strconv.ParseInt(ask[2], 10, 64)
					if err != nil {
						panic(err)
					}

					if seq > orderbook.LastUpdateID {
						asks = append(asks, ask)
					}
				}

				for _, bid := range update.Changes.Bids {
					seq, err := strconv.ParseInt(bid[2], 10, 64)
					if err != nil {
						panic(err)
					}
					if seq > orderbook.LastUpdateID {
						bids = append(bids, bid)
					}
				}

				firstUpdateApplied = true
				orderbook.ApplyUpdate(domain.NewOrderBookUpdate(
					bids, asks, update.SequenceEnd,
				))
				continue
			}

			if firstUpdateApplied {
				orderbook.ApplyUpdate(domain.NewOrderBookUpdate(
					update.Changes.Bids, update.Changes.Asks, update.SequenceEnd,
				))
			}

		}
	}
}

func (m *OrderbookMaintainer) subscribeDepthUpdate(symbol *domain.MarketSymbol) <-chan time.Time {

	t := time.NewTimer(3 * time.Second)
	subscription, err := m.stream.DepthDiffStream(symbol)
	if err != nil {
		panic("error while subscribing to depth update stream  " + err.Error())
	}

	go func() {
		for {
			select {
			case <-m.done:
				return
			case update := <-subscription.Stream:
				m.depthUpdateQueue.PushBack(update)
			}
		}
	}()

	return t.C
}
