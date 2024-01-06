package domain

import (
	"sort"
	"strconv"
	"sync"
	"time"
)

type OrderBookSnapshot struct {
	LastUpdateId int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

type OrderBookUpdate struct {
	Bids         [][]string
	Asks         [][]string
	LastUpdateID int64
}

type OrderBook struct {
	Provider       string
	Symbol         *MarketSymbol
	Asks           [][]float64
	Bids           [][]float64
	LastUpdateID   int64
	LastUpdateTime int64

	updateMx sync.Mutex
}

func NewOrderBook(provider string, symbol *MarketSymbol, snapshot *OrderBookSnapshot) *OrderBook {
	return &OrderBook{
		Provider:       provider,
		Symbol:         symbol,
		Asks:           parsePriceLevel(snapshot.Asks),
		Bids:           parsePriceLevel(snapshot.Bids),
		LastUpdateID:   snapshot.LastUpdateId,
		LastUpdateTime: time.Now().Unix(),
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

func (ob *OrderBook) TakeSnapshot() *OrderBookSnapshot {
	ob.updateMx.Lock()
	defer ob.updateMx.Unlock()

	return &OrderBookSnapshot{
		LastUpdateId: ob.LastUpdateID,
		Bids:         serializePriceLevel(ob.Bids),
		Asks:         serializePriceLevel(ob.Asks),
	}
}

func (ob *OrderBook) updateDepth(updateDepth [][]float64, isAsks bool) {
	var temp [][]float64

	if isAsks {
		temp = ob.Asks
	} else {
		temp = ob.Bids
	}

	for _, level := range updateDepth {
		price := level[0]
		quantity := level[1]

		if quantity == 0 {
			// remove price level
			for i, level := range temp {
				if level[0] == price {
					temp[i] = temp[len(temp)-1]
					temp = temp[:len(temp)-1]
					break
				}
			}
		} else {
			// if price level exists, update quantity
			// otherwise, add price level
			for i, level := range temp {
				if level[0] == price {
					temp[i][1] = quantity
					break
				}
			}

			temp = append(temp, []float64{price, quantity})
		}
	}

	if isAsks {
		sort.Slice(temp, func(i, j int) bool {
			return temp[i][0] < temp[j][0]
		})
	} else {
		sort.Slice(temp, func(i, j int) bool {
			return temp[i][0] > temp[j][0]
		})
	}

	if isAsks {
		ob.Asks = temp
	} else {
		ob.Bids = temp
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
