package kucoin

import (
	"testing"

	"github.com/joho/godotenv"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
	"github.com/stretchr/testify/assert"
)

func TestGerWsConnOpts(t *testing.T) {
	api := NewKucoinAPI()

	opts, err := api.WsConnOpts()
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, opts.Token)
}

func TestGetOrderBookSnapshot(t *testing.T) {
	godotenv.Load("../.env")

	api := NewKucoinAPI()

	symbol, _ := domain.NewMarketSymbol("BTC", "USDT")

	snapshot, err := api.OrderBookSnapshot(symbol)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, snapshot.LastUpdateId)
	assert.NotEmpty(t, snapshot.Bids)
	assert.NotEmpty(t, snapshot.Asks)
}
