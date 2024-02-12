package binance

import "github.com/spooky-finn/go-cryptomarkets-bridge/domain"

type BinanceDepthUpdateValidator struct{}

func (v *BinanceDepthUpdateValidator) IsValidUpd(update *domain.OrderBookUpdate, orderBookLastUpdId int64) error {
	// Drop any event where u is <= lastUpdateId in the snapshot
	if update.SequenceEnd <= orderBookLastUpdId {
		return domain.ErrOrderBookUpdateIsOutdated
	}

	// The first processed event should have U <= lastUpdateId+1 AND u >= lastUpdateId+1
	if update.SequenceStart <= orderBookLastUpdId+1 && update.SequenceEnd >= orderBookLastUpdId+1 {
		return nil
	}

	if update.SequenceStart > orderBookLastUpdId+1 {
		return domain.ErrOrderBookUpdateIsOutOfSequece
	}

	return nil
}

func (v *BinanceDepthUpdateValidator) IsErrOutOfSequece(err error) bool {
	return err == domain.ErrOrderBookUpdateIsOutOfSequece
}

func (v *BinanceDepthUpdateValidator) IsErrOutdated(err error) bool {
	return err == domain.ErrOrderBookUpdateIsOutdated
}

// Drop any event where u is <= lastUpdateId in the snapshot
// if update.SequenceEnd <= orderbook.LastUpdateID {
// 	return nil
// }

// // The first processed event should have U <= lastUpdateId+1 AND u >= lastUpdateId+1
// if !m.firstUpdateApplied &&
// 	(update.SequenceStart <= orderbook.LastUpdateID+1 && update.SequenceEnd >= orderbook.LastUpdateID+1) {
// 	orderbook.ApplyUpdate(update)
// 	m.firstUpdateApplied = true
// 	return nil
// }

// if m.firstUpdateApplied {
// 	// While listening to the stream, each new event's U should be equal to the previous event's u+1
// 	if update.SequenceStart != orderbook.LastUpdateID+1 {
// 		m.OutOfSequeceErrCount++
// 		logger.Printf("binance: orderbook maintainer: droped update: %d <= %d\n", update.SequenceEnd, orderbook.LastUpdateID)

// 		// if outOfSeqUpdatesLimit is reached, stop the maintainer
// 		if m.OutOfSequeceErrCount > outOfSeqUpdatesLimit {
// 			return fmt.Errorf("binance: orderbook maintainer: out of sequence updates limit reached")
// 		}
// 		return nil
// 	}

// 	orderbook.ApplyUpdate(update)
// }
