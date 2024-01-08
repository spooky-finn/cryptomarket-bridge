package kucoin

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

type SubscribtionEntry struct {
	ch              chan []byte
	subscriberCount int
}

type KucoinStreamClient struct {
	conn          *websocket.Conn
	subscriptions map[string]*SubscribtionEntry
	api           *KucoinAPI
}

func NewKucoinStreamClient(api *KucoinAPI) *KucoinStreamClient {
	return &KucoinStreamClient{

		conn:          nil,
		subscriptions: make(map[string]*SubscribtionEntry),
	}
}

func (c *KucoinStreamClient) Connect() error {
	connectionOpts, err := c.api.WsConnOpts()
	if err != nil {
		return err
	}

	endpoint := connectionOpts.InstanceServers[0].Endpoint
	log.Printf("kucoin-stream-client: connecting to the %s \n", endpoint)

	conn, _, err := websocket.DefaultDialer.Dial(endpoint, nil)
	if err != nil {
		return err
	}

	c.conn = conn
	go c.listenConnection()
	return nil
}

func (c *KucoinStreamClient) Disconnect() error {
	return c.conn.Close()
}

func (c *KucoinStreamClient) listenConnection() {
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

		// if multiStreamData["stream"] != nil {
		// 	topic := multiStreamData["stream"].(string)
		// 	entry, ok := c.subscriptions[topic]
		// 	if ok {
		// 		entry.ch <- msg
		// 	}
		// }
	}
}
