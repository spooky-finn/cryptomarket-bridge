package binance

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain"
)

const (
	binanceDefaultWebsocketEndpoint = "wss://stream.binance.com:9443/stream"
	pingDelay                       = time.Minute * 9
	logTag                          = "binance-stream-client"
)

type Message[T any] struct {
	Stream string `json:"stream"`
	Data   T      `json:"data"`
}

type SubscribtionEntry struct {
	ch              chan []byte
	subscriberCount int
}

type ReqMessage struct {
	ReqId  int      `json:"id"`
	Params []string `json:"params"`
	Method string   `json:"method"`
}

type ReqMessageAck struct {
	Result string `json:"stream"`
	ReqId  int    `json:"id"`
}

type BinanceStreamClient struct {
	conn          *websocket.Conn
	subscriptions map[string]*SubscribtionEntry
	mu            sync.Mutex
}

func NewBinanceStreamClient() *BinanceStreamClient {
	return &BinanceStreamClient{
		conn:          nil,
		subscriptions: make(map[string]*SubscribtionEntry),
	}
}

func (c *BinanceStreamClient) Connect() error {
	log.Printf("[%s] connecting to the %s \n", logTag, binanceDefaultWebsocketEndpoint)

	Dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 5 * time.Second,
	}

	conn, _, err := Dialer.Dial(binanceDefaultWebsocketEndpoint, nil)
	conn.SetReadLimit(655350)

	if err != nil {
		return err
	}
	c.conn = conn

	go c.listenConnection()
	return nil
}

func (c *BinanceStreamClient) Subscribe(topic string) *domain.Subscription[[]byte] {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.subscriptions[topic]
	if ok {
		entry.subscriberCount++
		c.subscriptions[topic] = entry

		return &domain.Subscription[[]byte]{
			Stream: entry.ch,
			Unsubscribe: func() {
				c.unsubscribe(topic)
			},
			Topic: topic,
		}
	}

	ch := make(chan []byte)
	c.subscriptions[topic] = &SubscribtionEntry{
		ch:              ch,
		subscriberCount: 1,
	}

	log.Printf("[%s] subscribing to the %s \n", logTag, topic)

	err := c.conn.WriteJSON(ReqMessage{
		Method: "SUBSCRIBE",
		ReqId:  getRandomReqID(),
		Params: []string{topic},
	})

	if err != nil {
		panic(err)
	}

	return &domain.Subscription[[]byte]{
		Stream: ch,
		Unsubscribe: func() {
			c.unsubscribe(topic)
		},
		Topic: topic,
	}
}

func (c *BinanceStreamClient) unsubscribe(topic string) error {
	log.Printf("[%s] unsubscribing from the %s \n", logTag, topic)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.subscriptions[topic].subscriberCount > 1 {
		c.subscriptions[topic].subscriberCount -= 1
	} else if c.subscriptions[topic].subscriberCount == 1 {
		close(c.subscriptions[topic].ch)
		delete(c.subscriptions, topic)
	}

	err := c.conn.WriteJSON(ReqMessage{
		Method: "UNSUBSCRIBE",
		ReqId:  getRandomReqID(),
		Params: []string{topic},
	})

	if err != nil {
		return err
	}

	return nil
}

func (c *BinanceStreamClient) Disconnect() error {
	return c.conn.Close()
}

func (c *BinanceStreamClient) listenConnection() {
	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			panic(err)
		}

		var multiStreamData map[string]interface{}

		err = json.Unmarshal(msg, &multiStreamData)
		if err != nil {
			log.Fatalf("error: %v message %v", err, string(msg))
		}

		// FIXME: handle log of ack message
		// if message have id then it is a response to a subscription
		if multiStreamData["id"] != nil {
			id := int(multiStreamData["id"].(float64))
			entry, ok := c.subscriptions[fmt.Sprintf("%v", id)]
			if ok {
				fmt.Println("receive ack")
				entry.ch <- msg
			}

			delete(c.subscriptions, fmt.Sprintf("%v", id))
		}

		if multiStreamData["stream"] != nil {
			topic := multiStreamData["stream"].(string)
			entry, ok := c.subscriptions[topic]
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
