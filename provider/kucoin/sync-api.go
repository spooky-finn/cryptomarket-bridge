package kucoin

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/Kucoin/kucoin-go-sdk"
	"github.com/spooky-finn/cryptobridge/domain"
)

var logger = log.New(log.Writer(), "[kucoin] ", log.LstdFlags)

type KucoinSyncAPI struct {
	endpoint   string
	apiService *kucoin.ApiService
}

func NewKucoinSyncAPI() *KucoinSyncAPI {
	return &KucoinSyncAPI{
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

func (api *KucoinSyncAPI) WsConnOpts() (*WebSocketTokenModel, error) {
	resp, err := api.apiService.WebSocketPublicToken()
	if err != nil {
		return nil, fmt.Errorf("failed to get ws connection options: %w", err)
	}

	data := &kucoin.WebSocketTokenModel{}

	if err = json.Unmarshal([]byte(resp.RawData), data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w, response: %s", err, resp.Message)
	}

	servers := make([]*WebSocketServerModel, 0, len(data.Servers))
	for _, server := range data.Servers {
		servers = append(servers, &WebSocketServerModel{
			PingInterval: server.PingInterval,
			Endpoint:     server.Endpoint,
			Protocol:     server.Protocol,
			Encrypt:      server.Encrypt,
			PingTimeout:  server.PingTimeout,
		})
	}

	token := &WebSocketTokenModel{
		Token:             data.Token,
		AcceptUserMessage: data.AcceptUserMessage,
		Servers:           servers,
	}

	return token, nil
}

func (api *KucoinSyncAPI) OrderBookSnapshot(symbol *domain.MarketSymbol, limit int) (*domain.OrderBookSnapshot, error) {
	s := strings.ToUpper(symbol.Join("-"))
	resp, err := api.apiService.AggregatedFullOrderBookV3(s)
	if err != nil {
		return nil, fmt.Errorf("failed to get order book snapshot: %w", err)
	}

	if !resp.HttpSuccessful() {
		return nil, fmt.Errorf("failed to get order book snapshot: %s", resp.Message)
	}
	data := &OrderBookSnapshot{}
	if err = json.Unmarshal(resp.RawData, data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w, response: %s", err, resp.RawData)
	}

	lastUpdId, err := strconv.Atoi(data.Sequence)
	if err != nil {
		return nil, fmt.Errorf("failed to convert sequence to int: %w, response: %s", err, resp.RawData)
	}

	if len(data.Asks) > limit {
		data.Asks = data.Asks[:limit]
	}

	if len(data.Bids) > limit {
		data.Bids = data.Bids[:limit]
	}

	obSnapshot := &domain.OrderBookSnapshot{
		Source:       domain.OrderBookSource_Provider,
		LastUpdateId: int64(lastUpdId),
		Bids:         data.Bids,
		Asks:         data.Asks,
	}

	return obSnapshot, nil

}
