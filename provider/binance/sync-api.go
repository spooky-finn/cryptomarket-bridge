package binance

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spooky-finn/cryptobridge/domain"
)

var logger = log.New(log.Writer(), "[binance] ", log.LstdFlags)

const ENDPOINT = ""

// Get OrderBookSnapshot (Depth)
type BinanceSyncAPI struct {
	conn       *websocket.Conn
	writeMutex sync.Mutex
	in         chan []byte
}

type GenericMessage[T any] struct {
	ID     int `json:"id"`
	Status int `json:"status"`
	Result T   `json:"result"`
}

func NewBinanceAPI() *BinanceSyncAPI {
	logger.Println("instantiating binance websocket api")
	instance := &BinanceSyncAPI{
		in: make(chan []byte),
	}

	Dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 5 * time.Second,
	}

	endpoint := os.Getenv("BINANCE_WS_API_ENDPOINT")
	conn, _, err := Dialer.Dial(endpoint, nil)
	if err != nil {
		logger.Printf("error dialing binance sync ws api: %s", err.Error())
	}
	instance.conn = conn

	go instance.listener(conn)
	return instance
}

func (api *BinanceSyncAPI) OrderBookSnapshot(symbol *domain.MarketSymbol, limit int) (*domain.OrderBookSnapshot, error) {
	reqId := getRandomReqID()

	// params is a object of symbol and limit
	params := map[string]interface{}{
		"symbol": strings.ToUpper(symbol.Join("")),
		"limit":  fmt.Sprintf("%d", limit),
	}

	api.writeMutex.Lock()
	err := api.conn.WriteJSON(map[string]interface{}{
		"method": "depth",
		"params": params,
		"id":     reqId,
	})
	api.writeMutex.Unlock()

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

func (api *BinanceSyncAPI) listener(conn *websocket.Conn) {
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

func (api *BinanceSyncAPI) waitForResponse(messageId int) ([]byte, error) {
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
