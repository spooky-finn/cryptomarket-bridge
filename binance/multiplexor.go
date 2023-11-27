package binance

import (
	"log"

	"github.com/gorilla/websocket"
)

// LogTag is a constant representing the log tag for binance package.
const LogTag = "binance"

// MultiplexorRegistration defines the registration details for the Multiplexor.
type MultiplexorRegistration struct {
	handleFunc    func(msg []byte) error // Function to handle messages
	onSubscribe   func() ReqMessage      // Function to subscribe to a topic
	onUnsubscribe func() ReqMessage      // Function to unsubscribe from a topic
}

// Multiplexor manages multiple clients and handles message multiplexing.
type Multiplexor struct {
	conn    *websocket.Conn
	clients []MultiplexorRegistration
}

// NewMultiplexor creates a new instance of Multiplexor.
func NewMultiplexor(conn *websocket.Conn) *Multiplexor {
	return &Multiplexor{
		conn:    conn,
		clients: make([]MultiplexorRegistration, 0),
	}
}

// Register adds a new client to the Multiplexor and returns a channel for receiving messages.
func (m *Multiplexor) Register(conf *MultiplexorRegistration) error {
	m.clients = append(m.clients, *conf)
	acked := make(chan bool)

	topic := conf.onSubscribe().Params[0]

	// Subscriber
	go func() {
		log.Default().Printf("[%s] subscribing to the %s \n", LogTag, topic)
		err := m.conn.WriteJSON(conf.onSubscribe())
		if err != nil {
			panic(err)
		}

		// Wait for acknowledgment
		_, ackMsg, err := m.conn.ReadMessage()
		if err != nil {
			panic(err)
		}

		if ackMsg != nil {
			log.Default().Printf("[%s] subscription to %s is acknowledged \n", LogTag, topic)
		}

		acked <- true
	}()

	// Reader
	go func() {
		<-acked

		for {
			_, msg, err := m.conn.ReadMessage()
			if err != nil {
				panic(err)
			}

			if len(m.clients) == 0 {
				panic("no clients")
			}

			// Iterate through clients and send message to those interested
			for _, client := range m.clients {
				client.handleFunc(msg)
			}
		}
	}()

	return nil
}
