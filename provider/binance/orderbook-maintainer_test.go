package binance

import (
	"testing"

	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
)

func TestCreateOrderBook(t *testing.T) {
	streamClient := NewBinanceStreamClient()
	streamClient.Connect()
	syncAPI := NewBinanceAPI()

	m := &OrderbookMaintainer{
		syncAPI:   syncAPI,
		streamAPI: NewBinanceStreamAPI(streamClient, syncAPI),
	}

	symbol, err := domain.NewMarketSymbol("BTC", "USDT")
	if err != nil {
		t.Fatalf("Failed to create market symbol")
	}

	m.CreareOrderBook(symbol, 10)
}
