package provider

import (
	"log"
	"os"
	"sync"

	"github.com/spooky-finn/go-cryptomarkets-bridge/domain/interfaces"
	"github.com/spooky-finn/go-cryptomarkets-bridge/provider/binance"
	"github.com/spooky-finn/go-cryptomarkets-bridge/provider/kucoin"
)

var logger = log.New(os.Stdout, "[api-resolver] ", log.LstdFlags)

type ConnectionManager struct {
	KucoinWS        *kucoin.KucoinStreamClient
	KucoinHttpApi   *kucoin.KucoinSyncAPI
	KucoinStreamAPI *kucoin.KucoinStreamAPI

	BinanceWC        *binance.BinanceStreamClient
	BinanceHttpAPI   *binance.BinanceAPI
	BinanceStreamAPI *binance.BinanceStreamAPI
}

func NewConnectionManager() *ConnectionManager {
	binanceStreamClient := binance.NewBinanceStreamClient()
	binanceSyncAPI := binance.NewBinanceAPI()

	kucoinSyncAPI := kucoin.NewKucoinSyncAPI()
	wsConnOpts, err := kucoinSyncAPI.WsConnOpts()
	if err != nil {
		panic("failed to get ws connection options: " + err.Error())
	}

	kucoinStreamClient := kucoin.NewKucoinStreamClient(wsConnOpts)
	KucoinStreamAPI := kucoin.NewKucoinStreamAPI(kucoinStreamClient, kucoinSyncAPI)

	return &ConnectionManager{
		KucoinWS:         kucoinStreamClient,
		KucoinHttpApi:    kucoinSyncAPI,
		KucoinStreamAPI:  KucoinStreamAPI,
		BinanceWC:        binanceStreamClient,
		BinanceHttpAPI:   binanceSyncAPI,
		BinanceStreamAPI: binance.NewBinanceStreamAPI(binanceStreamClient, binanceSyncAPI),
	}
}

func (cm *ConnectionManager) Init() {
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go cm.DialBinance(wg)
	go cm.DialKucoin(wg)
	wg.Wait()
}

func (cm *ConnectionManager) StreamApi(provider string) interfaces.ProviderStreamAPI {
	switch provider {
	case "kucoin":
		return cm.KucoinStreamAPI
	case "binance":
		return cm.BinanceStreamAPI
	}

	panic("unknown provider: " + provider)
}

func (cm *ConnectionManager) HttpApi(provider string) interfaces.ProviderHttpAPI {
	switch provider {
	case "kucoin":
		return cm.KucoinHttpApi
	case "binance":
		return cm.BinanceHttpAPI
	}

	panic("unknown provider: " + provider)
}

func (cm *ConnectionManager) DialBinance(wg *sync.WaitGroup) {
	if err := cm.BinanceWC.Connect(); err != nil {
		logger.Printf("failed to connect to binance ws: " + err.Error())
	}
	wg.Done()
}

func (cm *ConnectionManager) DialKucoin(wg *sync.WaitGroup) {
	if err := cm.KucoinWS.Connect(); err != nil {
		logger.Printf("failed to connect to kucoin ws: " + err.Error())
	}
	wg.Done()
}

func (cm *ConnectionManager) Close() {
	cm.KucoinWS.Close()
	cm.BinanceWC.Close()
}
