package binance

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
)

var logger = log.New(log.Writer(), "[binance]", log.LstdFlags)

const ENDPOINT = ""

// Get OrderBookSnapshot (Depth)
type BinanceAPI struct {
	conn *websocket.Conn
	in   chan []byte
}

type GenericMessage[T any] struct {
	ID     int `json:"id"`
	Status int `json:"status"`
	Result T   `json:"result"`
}

func NewBinanceAPI() *BinanceAPI {
	logger.Println("instantiating binance websocket api")
	instance := &BinanceAPI{
		in: make(chan []byte),
	}

	Dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 5 * time.Second,
	}

	endpoint := os.Getenv("BINANCE_STREAM_ENDPOINT")
	conn, _, err := Dialer.Dial(endpoint, nil)
	if err != nil {
		panic(err)
	}
	instance.conn = conn

	go instance.listener(conn)
	return instance
}

func (api *BinanceAPI) OrderBookSnapshot(symbol *domain.MarketSymbol, limit int) (*domain.OrderBookSnapshot, error) {
	reqId := getRandomReqID()

	// params is a object of symbol and limit
	params := map[string]interface{}{
		"symbol": strings.ToUpper(symbol.Join("")),
		"limit":  fmt.Sprintf("%d", limit),
	}

	err := api.conn.WriteJSON(map[string]interface{}{
		"method": "depth",
		"params": params,
		"id":     reqId,
	})

	if err != nil {
		return nil, err
	}

	msg, err := api.waitForResponse(reqId)
	if err != nil {
		return nil, err
	}

	// unmarshal message
	var response GenericMessage[domain.OrderBookSnapshot]
	err = json.Unmarshal(msg, &response)
	if err != nil {
		return nil, err
	}

	snapshot := &domain.OrderBookSnapshot{
		Source:       domain.OrderBookSource_Provider,
		LastUpdateId: response.Result.LastUpdateId,
		Bids:         response.Result.Bids,
		Asks:         response.Result.Asks,
	}

	return snapshot, nil
}

func (api *BinanceAPI) listener(conn *websocket.Conn) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			logger.Println(err)
			return
		}
		api.in <- message
	}
}

var ErrTimeout = errors.New("timeout error")

func (api *BinanceAPI) waitForResponse(messageId int) ([]byte, error) {
	for {
		select {
		case msg := <-api.in:
			// unmarshal message
			var response map[string]interface{}
			err := json.Unmarshal(msg, &response)
			if err != nil {
				return nil, err
			}

			// try to get id from response message and compare it with messageId
			if response["id"] != nil {
				id := int(response["id"].(float64))
				if id != messageId {
					continue
				}
				return msg, nil
			}

			continue

		case <-time.After(10 * time.Second):
			return nil, ErrTimeout
		}
	}
}
