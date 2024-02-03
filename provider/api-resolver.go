package provider

import (
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain/interfaces"
	"github.com/spooky-finn/go-cryptomarkets-bridge/provider/binance"
	"github.com/spooky-finn/go-cryptomarkets-bridge/provider/kucoin"
)

type APIResolver struct {
	KucoinHttpApi    *kucoin.KucoinSyncAPI
	KucoinStreamAPI  *kucoin.KucoinStreamAPI
	BinanceHttpAPI   *binance.BinanceAPI
	BinanceStreamAPI *binance.BinanceStreamAPI
}

func NewAPIResolver() *APIResolver {
	binanceStreamClient := binance.NewBinanceStreamClient()
	binanceSyncAPI := binance.NewBinanceAPI()

	kucoinSyncAPI := kucoin.NewKucoinSyncAPI()
	wsConnOpts, err := kucoinSyncAPI.WsConnOpts()
	if err != nil {
		panic("failed to get ws connection options: " + err.Error())
	}

	kucoinStreamClient := kucoin.NewKucoinStreamClient(wsConnOpts)
	KucoinStreamAPI := kucoin.NewKucoinStreamAPI(kucoinStreamClient)

	if _, _, err := kucoinStreamClient.Connect(); err != nil {
		panic("failed to connect to kucoin stream: " + err.Error())
	}

	if err := binanceStreamClient.Connect(); err != nil {
		panic("failed to connect to binance stream: " + err.Error())
	}

	return &APIResolver{
		KucoinHttpApi:    kucoinSyncAPI,
		KucoinStreamAPI:  KucoinStreamAPI,
		BinanceHttpAPI:   binanceSyncAPI,
		BinanceStreamAPI: binance.NewBinanceStreamAPI(binanceStreamClient, binanceSyncAPI),
	}
}

func (a *APIResolver) StreamApi(provider string) interfaces.ProviderStreamAPI {
	switch provider {
	case "kucoin":
		return a.KucoinStreamAPI
	case "binance":
		return a.BinanceStreamAPI
	}

	panic("unknown provider: " + provider)
}

func (a *APIResolver) HttpApi(provider string) interfaces.ProviderHttpAPI {
	switch provider {
	case "kucoin":
		return a.KucoinHttpApi
	case "binance":
		return a.BinanceHttpAPI
	}

	panic("unknown provider: " + provider)
}
