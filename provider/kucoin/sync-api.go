package kucoin

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/Kucoin/kucoin-go-sdk"
	pb "github.com/spooky-finn/go-cryptomarkets-bridge/cryptobridge"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
)

var logger = log.New(log.Writer(), "[kucoin]", log.LstdFlags)

type KucoinHttpAPI struct {
	endpoint   string
	apiService *kucoin.ApiService
}

func NewKucoinHttpAPI() *KucoinHttpAPI {
	return &KucoinHttpAPI{
		endpoint: os.Getenv("KUCOIN_BASE_URL"),
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

func (api *KucoinHttpAPI) WsConnOpts() (*kucoin.WebSocketTokenModel, error) {
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

func (api *KucoinHttpAPI) OrderBookSnapshot(symbol *domain.MarketSymbol, limit int) (*domain.OrderBookSnapshot, error) {
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
		Source:       pb.OrderBookSource_Provider,
		LastUpdateId: int64(lastUpdId),
		Bids:         data.Bids,
		Asks:         data.Asks,
	}

	return domainSnapshot, nil

}
