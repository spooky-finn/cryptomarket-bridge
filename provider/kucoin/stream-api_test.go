package kucoin

import (
	"testing"

	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
	"github.com/stretchr/testify/assert"
)

func TestKucoinStreamAPI(t *testing.T) {
	// Create a new KucoinStreamAPI instance
	client := NewKucoinAPI()
	streamAPI := NewKucoinStreamAPI(client)

	// Call Connect
	err := streamAPI.Connect()

	// Assert that there is no error
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}

	// Call DepthDiffStream
	symbol, err := domain.NewMarketSymbol("xmr", "btc")
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}

	// Cleanup - close the connection
	result := streamAPI.GetOrderBook(symbol)
	if result.Err != nil {
		t.Errorf("Unexpected error: %s", result.Err.Error())
	}

	if result.OrderBook == nil {
		t.Errorf("Order book should not be nil")
	}

	assert.Equal(t, 3, len(result.OrderBook.TakeSnapshot(3).Asks), "Asks should have the same length as the limit")

}
