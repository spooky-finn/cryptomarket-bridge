package kucoin

import (
	"testing"

	"github.com/joho/godotenv"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
	"github.com/stretchr/testify/assert"
)

func TestGerWsConnOpts(t *testing.T) {
	api := NewKucoinSyncAPI()

	opts, err := api.WsConnOpts()
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, opts.Token)
	assert.GreaterOrEqual(t, len(opts.Servers), 1)
}

func TestGetOrderBookSnapshot(t *testing.T) {
	err := godotenv.Load("../../.env")

	if err != nil {
		t.Fatal(err)
	}

	api := NewKucoinSyncAPI()

	symbol, _ := domain.NewMarketSymbol("BTC", "USDT")

	snapshot, err := api.OrderBookSnapshot(symbol, 5)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEmpty(t, snapshot.LastUpdateId)
	assert.NotEmpty(t, snapshot.Bids)
	assert.NotEmpty(t, snapshot.Asks)

	// assert len of bids and asks
	assert.Equal(t, 5, len(snapshot.Bids))
	assert.Equal(t, 5, len(snapshot.Asks))

}
