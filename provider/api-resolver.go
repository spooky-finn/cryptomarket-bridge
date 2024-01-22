package provider

import (
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain/interfaces"
	"github.com/spooky-finn/go-cryptomarkets-bridge/provider/binance"
	"github.com/spooky-finn/go-cryptomarkets-bridge/provider/kucoin"
)

type APIResolver struct {
	KucoinStreamAPI  *kucoin.KucoinStreamAPI
	BinanceStreamAPI *binance.BinanceStreamAPI
}

func NewAPIResolver() *APIResolver {
	binanceStreamClient := binance.NewBinanceStreamClient()
	kucoinApi := kucoin.NewKucoinAPI()
	KucoinStreamAPI := kucoin.NewKucoinStreamAPI(kucoinApi)

	if err := KucoinStreamAPI.Connect(); err != nil {
		panic("failed to connect to kucoin stream: " + err.Error())
	}

	if err := binanceStreamClient.Connect(); err != nil {
		panic("failed to connect to binance stream: " + err.Error())
	}

	return &APIResolver{
		KucoinStreamAPI:  KucoinStreamAPI,
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
