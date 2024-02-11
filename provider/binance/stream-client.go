package binance

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/recws-org/recws"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain/interfaces"
)

const (
	binanceDefaultWebsocketEndpoint = "wss://stream.binance.com:9443/stream"
	pingDelay                       = time.Minute * 9
)

type Message[T any] struct {
	Stream string `json:"stream"`
	Data   T      `json:"data"`
}

type SubscribtionEntry struct {
	ch              chan []byte
	subscriberCount int
}

type WebSocketRequestModel struct {
	ReqId  int      `json:"id"`
	Params []string `json:"params"`
	Method string   `json:"method"`
}

type WebSocketAckModel struct {
	Result string `json:"stream"`
	ReqId  int    `json:"id"`
}

type BinanceStreamClient struct {
	conn          *recws.RecConn
	subscriptions map[string]*SubscribtionEntry
	mu            sync.Mutex
}

type SubscibeResult = *interfaces.Subscription[[]byte]

func NewBinanceStreamClient() *BinanceStreamClient {
	return &BinanceStreamClient{
		conn:          nil,
		subscriptions: make(map[string]*SubscribtionEntry),
		mu:            sync.Mutex{},
	}
}

func (c *BinanceStreamClient) Connect() error {
	conn := &recws.RecConn{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 5 * time.Second,
		KeepAliveTimeout: pingDelay,
		Conn:             nil,
		NonVerbose:       false,
	}

	conn.Dial(binanceDefaultWebsocketEndpoint, nil)

	c.conn = conn

	go c.read()
	return nil
}

func (c *BinanceStreamClient) Subscribe(topic string) (SubscibeResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.conn.IsConnected() {
		panic("connection is not established")
	}

	entry, ok := c.subscriptions[topic]
	ch := make(chan []byte)

	if ok {
		entry.subscriberCount++
		c.subscriptions[topic] = entry

		ch = entry.ch
	} else {
		c.subscriptions[topic] = &SubscribtionEntry{
			ch:              ch,
			subscriberCount: 1,
		}

		logger.Println("subscribing to the ", topic)

		err := c.conn.WriteJSON(WebSocketRequestModel{
			Method: "SUBSCRIBE",
			ReqId:  getRandomReqID(),
			Params: []string{topic},
		})

		if err != nil {
			return nil, fmt.Errorf("failed to send subscribe msg for topic=%s", topic)
		}

	}

	return &interfaces.Subscription[[]byte]{
		Stream: ch,
		Unsubscribe: func() {
			c.unSubscribe(topic)
		},
		Topic: topic,
	}, nil
}

func (c *BinanceStreamClient) unSubscribe(topic string) error {
	logger.Println("unsubscribing from topic ", topic)
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.subscriptions[topic].subscriberCount > 1 {
		c.subscriptions[topic].subscriberCount -= 1
	} else if c.subscriptions[topic].subscriberCount == 1 {
		close(c.subscriptions[topic].ch)
		delete(c.subscriptions, topic)
	}

	err := c.conn.WriteJSON(WebSocketRequestModel{
		Method: "UNSUBSCRIBE",
		ReqId:  getRandomReqID(),
		Params: []string{topic},
	})

	if err != nil {
		return err
	}

	return nil
}

func (c *BinanceStreamClient) Close() error {
	return c.conn.Conn.Close()
}

func (c *BinanceStreamClient) read() {
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			fmt.Printf("error: while reading from connection %v \n", err)
			continue
		}

		var multiStreamData map[string]interface{}

		err = json.Unmarshal(msg, &multiStreamData)
		if err != nil {
			log.Fatalf("error: %v message %v", err, string(msg))
		}

		// FIXME: handle log of ack message
		// if message have id then it is a response to a subscription
		if multiStreamData["id"] != nil {
			c.mu.Lock()
			id := int(multiStreamData["id"].(float64))
			entry, ok := c.subscriptions[fmt.Sprintf("%v", id)]
			if ok {
				fmt.Println("receive ack")
				entry.ch <- msg
			}

			delete(c.subscriptions, fmt.Sprintf("%v", id))
			c.mu.Unlock()
		}

		if multiStreamData["stream"] != nil {
			topic := multiStreamData["stream"].(string)
			c.mu.Lock()
			entry, ok := c.subscriptions[topic]
			c.mu.Unlock()
			if ok {
				entry.ch <- msg
			}
		}
	}
}

func getRandomReqID() int {
	// write a function that returns a random number
	min := 10000
	max := 9999999
	v := min + rand.Intn(max-min)
	return v
}
