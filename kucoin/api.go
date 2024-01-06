package kucoin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
)

const ENDPOINT = "https://api.kucoin.com"

type KucoinAPI struct {
	endpoint string
}

func NewKucoinAPI() *KucoinAPI {
	return &KucoinAPI{
		endpoint: ENDPOINT,
	}
}

type KucoinWSConnOpts struct {
	Code string `json:"code"`
	Data struct {
		Token           string `json:"token"`
		InstanceServers []struct {
			Endpoint     string `json:"endpoint"`
			Encrypt      bool   `json:"encrypt"`
			Protocol     string `json:"protocol"`
			PingInterval int    `json:"pingInterval"`
			PingTimeout  int    `json:"pingTimeout"`
		} `json:"instanceServers"`
	} `json:"data"`
}

type OrderBookSnapshot struct {
	Sequence string     `json:"sequence"`
	Time     int64      `json:"time"`
	Bids     [][]string `json:"bids"`
	Asks     [][]string `json:"asks"`
}

func (api *KucoinAPI) WsConnOpts() (*KucoinWSConnOpts, error) {
	const url = "/api/v1/bullet-public"

	// send request to get token
	res, err := http.Post(api.endpoint+url, "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get ws connection options: %w", err)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w, data: %s", err, body)
	}

	data := &KucoinWSConnOpts{}

	if err = json.Unmarshal(body, data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w, data: %s", err, body)
	}

	return data, nil
}

func (api *KucoinAPI) OrderBookSnapshot(symbol *domain.MarketSymbol) (*domain.OrderBookSnapshot, error) {
	const url = "/api/v3/market/orderbook/level2?symbol=%s"

	res, err := http.Get(fmt.Sprintf(api.endpoint+url, symbol.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to get order book snapshot: %w", err)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	data := &OrderBookSnapshot{}
	if err = json.Unmarshal(body, data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w, data: %s", err, body)
	}

	lastUpdId, err := strconv.Atoi(data.Sequence)
	if err != nil {
		return nil, fmt.Errorf("failed to convert sequence to int: %w, data: %s", err, body)
	}

	domainSnapshot := &domain.OrderBookSnapshot{
		LastUpdateId: int64(lastUpdId),
		Bids:         data.Bids,
		Asks:         data.Asks,
	}
	return domainSnapshot, nil

}
