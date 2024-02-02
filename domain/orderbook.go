package domain

import (
	"sort"
	"strconv"
	"sync"
	"time"
)

type OrderBookSource string
type OrderBookStatus string

const (
	OrderBookSource_Provider       OrderBookSource = "Provider"
	OrderBookSource_LocalOrderBook OrderBookSource = "LocalOrderBook"

	OrderBookStatus_Ok      OrderBookStatus = "Ok"
	OrderBookStatus_Oudated OrderBookStatus = "Outdated"
)

type OrderBookSnapshot struct {
	Source       OrderBookSource `json:"source"`
	LastUpdateId int64           `json:"lastUpdateId"`
	Bids         [][]string      `json:"bids"`
	Asks         [][]string      `json:"asks"`
}

type OrderBookUpdate struct {
	Bids         [][]string
	Asks         [][]string
	LastUpdateID int64
}

func NewOrderBookUpdate(bids [][]string, asks [][]string, lastUpdateID int64) *OrderBookUpdate {
	return &OrderBookUpdate{
		Bids:         bids,
		Asks:         asks,
		LastUpdateID: lastUpdateID,
	}
}

type OrderBook struct {
	Provider       string
	Symbol         *MarketSymbol
	Asks           [][]float64
	Bids           [][]float64
	LastUpdateID   int64
	LastUpdateTime int64

	status OrderBookStatus
	// MessageBus chan interface{}
	OnSnapshotRecieved chan *OrderBookSnapshot
	updateMx           *sync.Mutex
}

func NewOrderBook(provider string, symbol *MarketSymbol, snapshot *OrderBookSnapshot) *OrderBook {
	return &OrderBook{
		Provider:       provider,
		Symbol:         symbol,
		Asks:           parsePriceLevel(snapshot.Asks),
		Bids:           parsePriceLevel(snapshot.Bids),
		LastUpdateID:   snapshot.LastUpdateId,
		LastUpdateTime: time.Now().Unix(),

		status: OrderBookStatus_Ok,

		updateMx: &sync.Mutex{},
	}
}

func (ob *OrderBook) ApplyUpdate(update *OrderBookUpdate) {
	updateAsks := parsePriceLevel(update.Asks)
	updateBids := parsePriceLevel(update.Bids)

	// TODO: is mutex here necessary?
	ob.updateMx.Lock()
	defer ob.updateMx.Unlock()

	if update.LastUpdateID <= ob.LastUpdateID {
		return
	}

	// TODO: add check for sequenciality of updates
	// // The first processed event should have U <= lastUpdateId+1 AND u >= lastUpdateId+1

	ob.LastUpdateID = update.LastUpdateID
	ob.LastUpdateTime = time.Now().Unix()

	ob.updateDepth(updateAsks, true)
	ob.updateDepth(updateBids, false)
}

func (ob *OrderBook) Stop() {
	ob.status = OrderBookStatus_Oudated
}

func (ob *OrderBook) TakeSnapshot(limit int) *OrderBookSnapshot {
	ob.updateMx.Lock()
	defer ob.updateMx.Unlock()

	bids := make([][]float64, len(ob.Bids))
	asks := make([][]float64, len(ob.Asks))

	copy(bids, ob.Bids)
	copy(asks, ob.Asks)

	bids = ob.limitDepth(bids, limit)
	asks = ob.limitDepth(asks, limit)

	return &OrderBookSnapshot{
		Source:       OrderBookSource_LocalOrderBook,
		LastUpdateId: ob.LastUpdateID,
		Bids:         serializePriceLevel(bids),
		Asks:         serializePriceLevel(asks),
	}
}

func (ob *OrderBook) limitDepth(depth [][]float64, limit int) [][]float64 {
	if limit > 0 && len(depth) > limit {
		return depth[:limit]
	}

	return depth
}

func (ob *OrderBook) updateDepth(updateDepth [][]float64, isAsks bool) {
	var depth [][]float64

	if isAsks {
		depth = ob.Asks
	} else {
		depth = ob.Bids
	}

	for _, level := range updateDepth {
		price := level[0]
		quantity := level[1]

		if quantity == 0 {
			// remove price level
			for i, level := range depth {
				if level[0] == price {
					depth[i] = depth[len(depth)-1]
					depth = depth[:len(depth)-1]
					break
				}
			}
		} else {
			// if price level exists, update quantity
			// otherwise, add price level
			updated := false
			for i, level := range depth {
				if level[0] == price {
					depth[i][1] = quantity
					updated = true
					break
				}
			}

			if !updated {
				depth = append(depth, []float64{price, quantity})
			}
		}
	}

	if isAsks {
		sort.Slice(depth, func(i, j int) bool {
			return depth[i][0] < depth[j][0]
		})
	} else {
		sort.Slice(depth, func(i, j int) bool {
			return depth[i][0] > depth[j][0]
		})
	}

	if isAsks {
		ob.Asks = depth
	} else {
		ob.Bids = depth
	}
}

func parsePriceLevel(depth [][]string) [][]float64 {
	result := make([][]float64, len(depth))
	for i, level := range depth {
		price, err := strconv.ParseFloat(level[0], 64)
		if err != nil {
			panic(err)
		}
		quantity, err := strconv.ParseFloat(level[1], 64)
		if err != nil {
			panic(err)
		}

		result[i] = []float64{price, quantity}
	}

	return result
}

func serializePriceLevel(depth [][]float64) [][]string {
	result := make([][]string, len(depth))
	for i, level := range depth {
		result[i] = []string{
			strconv.FormatFloat(level[0], 'f', -1, 64),
			strconv.FormatFloat(level[1], 'f', -1, 64),
		}
	}

	return result
}
