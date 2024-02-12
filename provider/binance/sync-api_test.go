package binance

import (
	"testing"

	"github.com/joho/godotenv"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
	"github.com/stretchr/testify/assert"
)

func TestBinanceWSAPI_GetOrderBookSnapshot(t *testing.T) {
	if err := godotenv.Load("../../.env"); err != nil {
		t.Fatal(err)
	}
	// Create a new BinanceWSAPI instance
	api := NewBinanceAPI()

	// Define a market symbol and limit for the test
	symbol, err := domain.NewMarketSymbol("xmr", "btc")
	assert.NoError(t, err, "Unexpected error")
	limit := 3

	// Call GetOrderBookSnapshot
	orderBook, err := api.OrderBookSnapshot(symbol, limit)

	assert.NoError(t, err, "Unexpected error")
	assert.NotNil(t, orderBook, "Order book should not be nil")
	assert.NotEmpty(t, orderBook.Asks, "Asks should not be empty")
	assert.NotEmpty(t, orderBook.Bids, "Bids should not be empty")

	assert.Equal(t, limit, len(orderBook.Asks), "Asks should have the same length as the limit")
	assert.Equal(t, limit, len(orderBook.Bids), "Bids should have the same length as the limit")
	// Cleanup - close the connection
	api.conn.Close()
}
