package binance

import (
	"testing"

	"github.com/spooky-finn/cryptobridge/domain"
	"github.com/stretchr/testify/assert"
)

func TestDepthUpdateValidator(t *testing.T) {
	v := &BinanceDepthUpdateValidator{}

	upd := &domain.OrderBookUpdate{
		SequenceStart: 123,
		SequenceEnd:   124,
		Bids:          [][]string{{"10000", "1"}, {"9900", "2"}},
		Asks:          [][]string{{"10100", "1.5"}, {"10200", "2.5"}},
		Symbol:        &domain.MarketSymbol{BaseAsset: "BTC", QuoteAsset: "USDT"},
	}

	// if sequenceEnd is <= lastUpdateID, it should be domain.ErrOrderBookUpdateIsOutOfSequece
	err := v.IsValidUpd(upd, 124)
	assert.Equal(t, domain.ErrOrderBookUpdateIsOutdated, err, "Error should match")

	// if U <= lastUpdateId+1 AND u >= lastUpdateId+1 (from docs)
	// if SequenceStart <= lastUpdateId+1 && SequenceEnd >= lastUpdateId+1
	// if 123 <= 123 +1 && 124 >= 123 +1
	// 123 <= 124 && 124 >= 124
	err = v.IsValidUpd(upd, 123)
	assert.Nil(t, err, "Error should be nil")
}

func TestDepthUpdateValidator2(t *testing.T) {
	v := &BinanceDepthUpdateValidator{}
	upd := &domain.OrderBookUpdate{
		SequenceStart: 123,
		SequenceEnd:   140,
		Bids:          [][]string{{"10000", "1"}, {"9900", "2"}},
		Asks:          [][]string{{"10100", "1.5"}, {"10200", "2.5"}},
		Symbol:        &domain.MarketSymbol{BaseAsset: "BTC", QuoteAsset: "USDT"},
	}

	// if U <= lastUpdateId+1 AND u >= lastUpdateId+1 (from docs)
	// if SequenceStart <= lastUpdateId+1 && SequenceEnd >= lastUpdateId+1
	// if 123 <= 123 +1 && 140 >= 123 +1
	// 123 <= 124 && 140 >= 124
	err := v.IsValidUpd(upd, 123)
	assert.Nil(t, err, "Error should be nil")
}

func TestDepthUpdateValidator_OutOfSeq(t *testing.T) {
	v := &BinanceDepthUpdateValidator{}

	upd := &domain.OrderBookUpdate{
		SequenceStart: 125,
		SequenceEnd:   136,
		Bids:          [][]string{{"10000", "1"}, {"9900", "2"}},
		Asks:          [][]string{{"10100", "1.5"}, {"10200", "2.5"}},
		Symbol:        &domain.MarketSymbol{BaseAsset: "BTC", QuoteAsset: "USDT"},
	}

	// if sequenceEnd is <= lastUpdateID, it should be domain.ErrOrderBookUpdateIsOutOfSequece
	err := v.IsValidUpd(upd, 122)
	assert.Equal(t, domain.ErrOrderBookUpdateIsOutOfSequece, err, "Error should match")
}
