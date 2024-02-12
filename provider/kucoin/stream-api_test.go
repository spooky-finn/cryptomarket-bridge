package kucoin

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
	"github.com/spooky-finn/go-cryptomarkets-bridge/helpers"
	"github.com/stretchr/testify/assert"
)

func createDeps() (*KucoinStreamAPI, *KucoinSyncAPI) {
	// Create a new KucoinStreamAPI instance
	syncAPI := NewKucoinSyncAPI()
	wsConnectionOpts, err := syncAPI.WsConnOpts()
	if err != nil {
		fmt.Printf("Error while creating ws connection options %s", err.Error())
		panic(err)
	}

	streamClient := NewKucoinStreamClient(wsConnectionOpts)

	if err := streamClient.Connect(); err != nil {
		fmt.Printf("Error while connecting to kucoin %s", err.Error())
	}

	streamAPI := NewKucoinStreamAPI(streamClient, syncAPI)

	return streamAPI, syncAPI
}

func TestKucoinStreamAPIGetOrderBook(t *testing.T) {
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatal(err)
	}
	// Create a new KucoinStrea	mAPI instance
	streamAPI, _ := createDeps()

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
}

func TestKucoinStreamAPIDepthStream(t *testing.T) {
	// Create a new KucoinStreamAPI instance
	streamAPI, _ := createDeps()

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
	if update.Symbol.String() != "btc_usdt" {
		t.Errorf("Symbol should be btc_usdt")
	}

	// Cleanup - close the connection
	// subscription.Unsubscribe()

	if update.SequenceEnd == 0 || update.SequenceStart == 0 {
		t.Errorf("SequenceEnd should not be 0")
	}
}

func TestWaitForSync(t *testing.T) {
	startT := time.Now()
	ch1 := make(chan struct{}, 1)
	ch2 := make(chan struct{}, 1)
	wg := sync.WaitGroup{}

	wg.Add(2)
	go func() {
		<-time.After(150 * time.Millisecond)
		ch1 <- struct{}{}
		wg.Done()
	}()

	go func() {
		<-time.After(2 * time.Second)
		ch2 <- struct{}{}
		wg.Done()
	}()

	wg.Wait()

	<-helpers.WithLatestFrom(ch1, ch2)
	elapsed := time.Since(startT)

	assert.True(t, elapsed > 2*time.Second, "Elapsed time should be greater than 2 seconds")
	assert.True(t, elapsed < 3*time.Second, "Elapsed time should be less than 3 seconds")
}

func TestKucoinCreateMultiplexTunnel(t *testing.T) {
	// Create a new KucoinStreamAPI instance
	streamAPI, _ := createDeps()
	err := streamAPI.WebSocket.CreateMultiplexTunnel("main-tunnel")
	if err != nil {
		t.Errorf("Error while creating multiplex tunnel %s", err.Error())
		return
	}
}
