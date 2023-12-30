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
	binanceDefaultWebsocketURL = "wss://stream.binance.com:9443/stream"
	pingDelay                  = time.Minute * 9
	logTag                     = "binance-stream-client"
)

type Message[T any] struct {
	Stream string `json:"stream"`
	Data   T      `json:"data"`
}

type OutgoingTableEntry struct {
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
	conn                 *websocket.Conn
	subscriptionRegistry map[string]*OutgoingTableEntry
	mu                   sync.Mutex
}

func NewBinanceStreamClient() *BinanceStreamClient {
	return &BinanceStreamClient{
		conn:                 nil,
		subscriptionRegistry: make(map[string]*OutgoingTableEntry),
	}
}

func (c *BinanceStreamClient) Connect() error {
	log.Printf("[%s] connecting to the %s \n", logTag, binanceDefaultWebsocketURL)

	Dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 5 * time.Second,
	}

	conn, _, err := Dialer.Dial(binanceDefaultWebsocketURL, nil)
	conn.SetReadLimit(655350)
	conn.SetReadDeadline(time.Now().Add(pingDelay))
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

	entry, ok := c.subscriptionRegistry[topic]
	if ok {
		entry.subscriberCount++
		c.subscriptionRegistry[topic] = entry

		return &domain.Subscription[[]byte]{
			Stream: entry.ch,
			Unsubscribe: func() {
				c.unsubscribe(topic)
			},
			Topic: topic,
		}
	}

	ch := make(chan []byte)

	c.subscriptionRegistry[topic] = &OutgoingTableEntry{
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

	if c.subscriptionRegistry[topic].subscriberCount > 1 {
		c.subscriptionRegistry[topic].subscriberCount -= 1
	} else if c.subscriptionRegistry[topic].subscriberCount == 1 {
		close(c.subscriptionRegistry[topic].ch)
		delete(c.subscriptionRegistry, topic)
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

		var unknownStreamMessage map[string]interface{}

		err = json.Unmarshal(msg, &unknownStreamMessage)
		if err != nil {
			log.Fatalf("error: %v message %v", err, string(msg))
		}

		// if message have id then it is a response to a subscription
		if unknownStreamMessage["id"] != nil {
			id := int(unknownStreamMessage["id"].(float64))
			entry, ok := c.subscriptionRegistry[fmt.Sprintf("%v", id)]
			if ok {
				fmt.Println("receive ack")
				entry.ch <- msg
			}

			delete(c.subscriptionRegistry, fmt.Sprintf("%v", id))
		}

		if unknownStreamMessage["stream"] != nil {
			topic := unknownStreamMessage["stream"].(string)
			entry, ok := c.subscriptionRegistry[topic]
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
