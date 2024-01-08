package kucoin

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Kucoin/kucoin-go-sdk"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
)

const ENDPOINT = "https://api.kucoin.com"

type KucoinAPI struct {
	endpoint   string
	apiService *kucoin.ApiService
}

func NewKucoinAPI() *KucoinAPI {
	return &KucoinAPI{
		endpoint: ENDPOINT,
		apiService: kucoin.NewApiService(
			kucoin.ApiKeyOption(os.Getenv("KUCOIN_API_KEY")),
			kucoin.ApiSecretOption(os.Getenv("KUCOIN_SECRET_KEY")),
			kucoin.ApiPassPhraseOption(os.Getenv("KUCOIN_PASSPHRASE")),
		),
	}
}

type KucoinWSConnOpts struct {
	Token           string `json:"token"`
	InstanceServers []struct {
		Endpoint     string `json:"endpoint"`
		Encrypt      bool   `json:"encrypt"`
		Protocol     string `json:"protocol"`
		PingInterval int    `json:"pingInterval"`
		PingTimeout  int    `json:"pingTimeout"`
	} `json:"instanceServers"`
}

type OrderBookSnapshot struct {
	Sequence string     `json:"sequence"`
	Time     int64      `json:"time"`
	Bids     [][]string `json:"bids"`
	Asks     [][]string `json:"asks"`
}

func (api *KucoinAPI) WsConnOpts() (*kucoin.WebSocketTokenModel, error) {
	resp, err := api.apiService.WebSocketPublicToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get ws connection options: %w", err)
	}

	data := &kucoin.WebSocketTokenModel{}

	if err = json.Unmarshal([]byte(resp.RawData), data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w, response: %s", err, resp.Message)
	}

	return data, nil
}

func (api *KucoinAPI) OrderBookSnapshot(symbol *domain.MarketSymbol) (*domain.OrderBookSnapshot, error) {
	s := strings.ToUpper(symbol.Join("-"))
	resp, err := api.apiService.AggregatedFullOrderBookV3(s)
	if err != nil {
		return nil, fmt.Errorf("failed to get order book snapshot: %w", err)
	}

	data := &OrderBookSnapshot{}
	if err = json.Unmarshal(resp.RawData, data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w, response: %s", err, resp.RawData)
	}

	lastUpdId, err := strconv.Atoi(data.Sequence)
	if err != nil {
		return nil, fmt.Errorf("failed to convert sequence to int: %w, response: %s", err, resp.RawData)
	}

	domainSnapshot := &domain.OrderBookSnapshot{
		LastUpdateId: int64(lastUpdId),
		Bids:         data.Bids,
		Asks:         data.Asks,
	}

	return domainSnapshot, nil

}
