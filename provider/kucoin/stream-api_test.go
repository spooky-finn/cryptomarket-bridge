package kucoin

import (
	"fmt"
	"testing"

	"github.com/joho/godotenv"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
	"github.com/stretchr/testify/assert"
)

func createDeps() (*KucoinStreamAPI, *KucoinSyncAPI, *KucoinStreamClient) {
	// Create a new KucoinStreamAPI instance
	syncAPI := NewKucoinSyncAPI()
	wsConnectionOpts, err := syncAPI.WsConnOpts()
	if err != nil {
		fmt.Printf("Error while creating ws connection options %s", err.Error())
		panic(err)
	}

	streamClient := NewKucoinStreamClient(wsConnectionOpts)
	streamAPI := NewKucoinStreamAPI(streamClient, syncAPI)

	return streamAPI, syncAPI, streamClient
}

func TestKucoinStreamAPIGetOrderBook(t *testing.T) {
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatal(err)
	}
	// Create a new KucoinStrea	mAPI instance
	streamAPI, _, streamClient := createDeps()

	err = streamClient.Connect()
	if err != nil {
		t.Errorf("Error whule connecting to kucoin %s", err.Error())
	}

	// Assert that there is no error
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}

	// Call DepthDiffStream
	symbol, err := domain.NewMarketSymbol("btc", "usdt")
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}

	// Cleanup - close the connection
	result := streamAPI.GetOrderBook(symbol, 50)
	if result.Err != nil {
		t.Errorf("Error while creating orderBook %s", result.Err.Error())
	}

	if result.OrderBook == nil {
		t.Errorf("Order book should not be nil")
	}

	assert.Equal(t, 3, len(result.OrderBook.TakeSnapshot(3).Asks), "Asks should have the same length as the limit")
	assert.Equal(t, 3, len(result.OrderBook.TakeSnapshot(3).Bids), "Bids should have the same length as the limit")
	// assert.Equal(t, 2, result.OrderBook.OnSnapshotRecieved)
}

func TestKucoinStreamAPIDepthStream(t *testing.T) {
	// Create a new KucoinStreamAPI instance
	streamAPI, _, streamClient := createDeps()

	err := streamClient.Connect()
	if err != nil {
		t.Errorf("Error whule connecting to kucoin %s", err.Error())
	}

	// Call DepthDiffStream
	symbol, err := domain.NewMarketSymbol("btc", "usdt")
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}

	subscription, err := streamAPI.DepthDiffStream(symbol)
	if err != nil {
		t.Errorf("Error while creating depth stream %s", err.Error())
		return
	}

	if subscription.Stream == nil {
		t.Errorf("Subscription.Stream should not be nil")
		return
	}

	update := <-subscription.Stream

	fmt.Printf("Update: %#v\n", update)
	if update.Symbol != "BTC-USDT" {
		t.Errorf("Symbol should be btc-usdt")
	}

	// Cleanup - close the connection
	subscription.Unsubscribe()

	if update.SequenceEnd == 0 || update.SequenceStart == 0 {
		t.Errorf("SequenceEnd should not be 0")
	}
}
