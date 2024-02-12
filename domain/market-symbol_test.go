package domain_test

import (
	"testing"

	"github.com/spooky-finn/cryptobridge/domain"
	"github.com/stretchr/testify/assert"
)

func TestNewMarketSymbol(t *testing.T) {
	tests := []struct {
		name        string
		base, quote string
		expectError bool
	}{
		{"ValidSymbol", "BTC", "USDT", false},
		{"EqualBaseQuote", "ETH", "ETH", true},
		{"EmptyBase", "", "USDT", true},
		{"EmptyQuote", "BTC", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewMarketSymbol(tt.base, tt.quote)

			if tt.expectError {
				assert.Error(t, err, "NewMarketSymbol() should return an error")
			} else {
				assert.NoError(t, err, "NewMarketSymbol() should not return an error")
			}
		})
	}
}

func TestNewSymbolFromString(t *testing.T) {
	tests := []struct {
		name        string
		symbol      string
		expectError bool
	}{
		{"ValidString", "BTC_USDT", false},
		{"InvalidString", "ETH-USD", true},
		{"EmptyString", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := domain.NewMarketSymbolFromString(tt.symbol)

			if tt.expectError {
				assert.Error(t, err, "NewSymbolFromString() should return an error")
			} else {
				assert.NoError(t, err, "NewSymbolFromString() should not return an error")
			}
		})
	}
}

func TestMarketSymbol_Join(t *testing.T) {
	ms := domain.MarketSymbol{BaseAsset: "BTC", QuoteAsset: "USDT"}
	separator := "_"

	result := ms.Join(separator)

	expected := "BTC_USDT"
	assert.Equal(t, expected, result, "Join() result should be equal to expected")
}

func TestMarketSymbol_String(t *testing.T) {
	ms := domain.MarketSymbol{BaseAsset: "BTC", QuoteAsset: "USDT"}

	result := ms.String()

	expected := "BTC_USDT"
	assert.Equal(t, expected, result, "String() result should be equal to expected")
}

func TestMarketSymbol_Equal(t *testing.T) {
	ms1 := domain.MarketSymbol{BaseAsset: "BTC", QuoteAsset: "USDT"}
	ms2 := domain.MarketSymbol{BaseAsset: "BTC", QuoteAsset: "USDT"}
	ms3 := domain.MarketSymbol{BaseAsset: "ETH", QuoteAsset: "USDT"}

	assert.True(t, ms1.Equal(&ms2), "Equal() should return true for equal symbols")
	assert.False(t, ms1.Equal(&ms3), "Equal() should return false for different symbols")
}

func TestMarketSymbol_LovercaseConvertion(t *testing.T) {
	ms, err := domain.NewMarketSymbol("BTC", "USDT")
	if err != nil {
		t.Errorf("NewMarketSymbol() should not return an error")
	}

	result := ms.String()

	expected := "btc_usdt"
	assert.Equal(t, expected, result, "String() result should be equal to expected")
}
