package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewOrderBook(t *testing.T) {
	// Mock data for testing
	provider := "MockProvider"
	symbol, err := NewMarketSymbol("BTC", "USDT")
	if err != nil {
		t.Fatal(err)
	}

	snapshot := &OrderBookSnapshot{
		LastUpdateId: 123,
		Bids:         [][]string{{"10000", "1"}, {"9900", "2"}},
		Asks:         [][]string{{"10100", "1.5"}, {"10200", "2.5"}},
	}

	// Create a new OrderBook instance
	ob := NewOrderBook(provider, symbol, snapshot)

	// Assertions
	assert.Equal(t, provider, ob.Provider, "Provider should match")
	assert.Equal(t, symbol, ob.Symbol, "Symbol should match")
	assert.Equal(t, snapshot.LastUpdateId, ob.LastUpdateID, "LastUpdateID should match")
	assert.NotEmpty(t, ob.Asks, "Asks should not be empty")
	assert.NotEmpty(t, ob.Bids, "Bids should not be empty")
}

func TestOrderBook_ApplyUpdate(t *testing.T) {
	// Mock data for testing
	provider := "MockProvider"
	symbol, err := NewMarketSymbol("BTC", "USDT")
	if err != nil {
		t.Fatal(err)
	}

	snapshot := &OrderBookSnapshot{
		LastUpdateId: 123,
		Bids:         [][]string{{"10000", "1"}, {"9900", "2"}},
		Asks:         [][]string{{"10100", "1.5"}, {"10200", "2.5"}},
	}
	update := &OrderBookUpdate{
		LastUpdateID: 124,
		Bids:         [][]string{{"9800", "3"}},                  // adding new bid
		Asks:         [][]string{{"10300", "2"}, {"10100", "0"}}, // updating and removing ask
	}

	// Create a new OrderBook instance
	ob := NewOrderBook(provider, symbol, snapshot)

	// Apply the update
	ob.ApplyUpdate(update)

	// Assertions
	assert.Equal(t, update.LastUpdateID, ob.LastUpdateID, "LastUpdateID should match")
	assert.NotEmpty(t, ob.Asks, "Asks should not be empty")
	assert.NotEmpty(t, ob.Bids, "Bids should not be empty")
	assert.Equal(t, [][]float64{{10200, 2.5}, {10300.0, 2.0}}, ob.Asks, "Asks should match")
	assert.Equal(t, [][]float64{{10000, 1}, {9900.0, 2.0}, {9800.0, 3.0}}, ob.Bids, "Bids should match")
}

func TestOrderBook_TakeSnapshot(t *testing.T) {
	// Mock data for testing
	provider := "MockProvider"
	symbol, err := NewMarketSymbol("BTC", "USDT")
	if err != nil {
		t.Fatal(err)
	}

	snapshot := &OrderBookSnapshot{
		LastUpdateId: 123,
		Bids:         [][]string{{"10000", "1"}, {"9900", "2"}},
		Asks:         [][]string{{"10100", "1.5"}, {"10200", "2.5"}},
	}

	// Create a new OrderBook instance
	ob := NewOrderBook(provider, symbol, snapshot)

	// Take a snapshot
	result := ob.TakeSnapshot()

	// Assertions
	assert.Equal(t, snapshot.LastUpdateId, result.LastUpdateId, "LastUpdateID should match")
	assert.NotEmpty(t, result.Asks, "Asks should not be empty")
	assert.NotEmpty(t, result.Bids, "Bids should not be empty")
	assert.Equal(t, snapshot.Asks, result.Asks, "Asks should match")
	assert.Equal(t, snapshot.Bids, result.Bids, "Bids should match")
}

func TestParsePriceLevel(t *testing.T) {
	// Mock data for testing
	asksOrBids := [][]string{{"10000", "1"}, {"9900", "2"}}

	// Call parsePriceLevel
	result := parsePriceLevel(asksOrBids)

	assert.NotEmpty(t, result, "Result should not be empty")
	assert.Equal(t, [][]float64{{10000.0, 1.0}, {9900.0, 2.0}}, result, "Result should match")
}

func TestSerializePriceLevel(t *testing.T) {
	// Mock data for testing
	asksOrBids := [][]float64{{10000.0, 1.0}, {9900.0, 2.0}}

	// Call serializePriceLevel
	result := serializePriceLevel(asksOrBids)

	// Assertions
	assert.Equal(t, [][]string{{"10000", "1"}, {"9900", "2"}}, result, "Result should match")
	assert.NotEmpty(t, result, "Result should not be empty")
}
