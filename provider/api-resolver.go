package provider

import (
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain/interfaces"
	"github.com/spooky-finn/go-cryptomarkets-bridge/provider/binance"
	"github.com/spooky-finn/go-cryptomarkets-bridge/provider/kucoin"
)

type APIResolver struct {
	KucoinHttpApi    *kucoin.KucoinHttpAPI
	KucoinStreamAPI  *kucoin.KucoinStreamAPI
	BinanceHttpAPI   *binance.BinanceAPI
	BinanceStreamAPI *binance.BinanceStreamAPI
}

func NewAPIResolver() *APIResolver {
	binanceStreamClient := binance.NewBinanceStreamClient()
	kucoinHttpApi := kucoin.NewKucoinHttpAPI()
	KucoinStreamAPI := kucoin.NewKucoinStreamAPI(kucoinHttpApi)

	if err := KucoinStreamAPI.Connect(); err != nil {
		panic("failed to connect to kucoin stream: " + err.Error())
	}

	if err := binanceStreamClient.Connect(); err != nil {
		panic("failed to connect to binance stream: " + err.Error())
	}

	return &APIResolver{
		KucoinHttpApi:    kucoinHttpApi,
		KucoinStreamAPI:  KucoinStreamAPI,
		BinanceHttpAPI:   binance.NewBinanceAPI(),
		BinanceStreamAPI: binance.NewBinanceStreamAPI(binanceStreamClient),
	}
}

func (a *APIResolver) ByProvider(provider string) interfaces.ProviderStreamAPI {
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
