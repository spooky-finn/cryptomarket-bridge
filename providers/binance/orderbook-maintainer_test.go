package binance

import (
	"testing"

	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
)

func TestCreateOrderBook(t *testing.T) {
	streamClient := NewBinanceStreamClient()
	streamClient.Connect()

	m := &OrderbookMaintainer{
		api:    NewBinanceAPI(),
		stream: NewBinanceStreamAPI(streamClient),
	}

	symbol, err := domain.NewMarketSymbol("BTC", "USDT")
	if err != nil {
		t.Fatalf("Failed to create market symbol")
	}

	m.CreareOrderBook(symbol)
}
