package binance

import (
	"encoding/json"
	"math/rand"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type StreamName struct {
	Stream string `json:"stream"`
}

type Message[T any] struct {
	StreamName
	Data T `json:"data"`
}

type BinanceStreamClient struct {
	conn        *websocket.Conn
	multiplexor *Multiplexor
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

type DepthUpdateData struct {
	Event       string     `json:"e"`
	EventTime   int64      `json:"E"`
	Symbol      string     `json:"s"`
	FirstUpdate int        `json:"U"`
	FinalUpdate int        `json:"u"`
	Bids        [][]string `json:"b"`
	Asks        [][]string `json:"a"`
}

func NewBinanceClient() *BinanceStreamClient {
	return &BinanceStreamClient{
		conn: nil,
	}
}

func (c *BinanceStreamClient) Connect() error {
	conn, _, err := websocket.DefaultDialer.Dial("wss://stream.binance.com:9443/stream", nil)
	if err != nil {
		return err
	}
	c.conn = conn
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	c.multiplexor = NewMultiplexor(conn)
	return nil
}

func (c *BinanceStreamClient) Subscribe(topic string) chan interface{} {
	ch := make(chan interface{})

	conf := MultiplexorRegistration{
		handleFunc: func(msg []byte) error {
			message := make(map[string]interface{})

			err := json.Unmarshal(msg, &message)
			if err != nil {
				panic(err)
			}

			if message["stream"] == topic {
				data := &Message[DepthUpdateData]{}

				err = json.Unmarshal(msg, data)
				if err != nil {
					panic(err)
				}

				ch <- data
				return nil
			}

			return nil
		},
		onSubscribe: func() ReqMessage {
			return ReqMessage{
				Method: "SUBSCRIBE",
				ReqId:  getRandomReqID(),
				Params: []string{topic},
			}
		},
		onUnsubscribe: func() ReqMessage {
			return ReqMessage{
				Method: "UNSUBSCRIBE",
				ReqId:  getRandomReqID(),
				Params: []string{topic},
			}
		},
	}

	c.multiplexor.Register(&conf)

	return ch
}

func (c *BinanceStreamClient) Close() error {
	return c.conn.Close()
}

func getRandomReqID() int {
	// write a function that returns a random number
	min := 10000
	max := 9999999
	v := min + rand.Intn(max-min)
	return v
}
