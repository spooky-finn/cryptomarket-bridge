package kucoin

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"sync"

	"github.com/pkg/errors"
	"github.com/spooky-finn/go-cryptomarkets-bridge/domain/interfaces"
	"github.com/spooky-finn/go-cryptomarkets-bridge/helpers"

	"github.com/recws-org/recws"
)

const (
	kucoinAckTimeout               = time.Second * 5
	pingInterval                   = time.Second * 30
	kucoinDefaultWebsocketEndpoint = "wss://api.kucoin.com"
	DebugMode                      = false
)

// All message types of WebSocket.
const (
	WelcomeMessage     = "welcome"
	PingMessage        = "ping"
	PongMessage        = "pong"
	SubscribeMessage   = "subscribe"
	AckMessage         = "ack"
	UnsubscribeMessage = "unsubscribe"
	ErrorMessage       = "error"
	Message            = "message"
	Notice             = "notice"
	Command            = "command"
)

// A WebSocketMessage represents a message between the WebSocket client and server.
type WebSocketMessage struct {
	Id   string `json:"id"`
	Type string `json:"type"`
}

// A WebSocketSubscribeMessage represents a message to subscribe the public/private channel.
type WebSocketSubscribeMessage struct {
	*WebSocketMessage
	Topic          string `json:"topic"`
	PrivateChannel bool   `json:"privateChannel"`
	Response       bool   `json:"response"`
}

type WebSocketDownstreamMessage struct {
	*WebSocketMessage
	Sn      string          `json:"sn"`
	Topic   string          `json:"topic"`
	Subject string          `json:"subject"`
	RawData json.RawMessage `json:"data"`
}

// A WebSocketTokenModel contains a token and some servers for WebSocket feed.
type WebSocketTokenModel struct {
	Token             string                `json:"token"`
	Servers           WebSocketServersModel `json:"instanceServers"`
	AcceptUserMessage bool                  `json:"accept_user_message"`
}

type WebSocketServerModel struct {
	PingInterval int64  `json:"pingInterval"`
	Endpoint     string `json:"endpoint"`
	Protocol     string `json:"protocol"`
	Encrypt      bool   `json:"encrypt"`
	PingTimeout  int64  `json:"pingTimeout"`
}

// A WebSocketServersModel is the set of *WebSocketServerModel.
type WebSocketServersModel []*WebSocketServerModel

// RandomServer returns a server randomly.
func (s WebSocketServersModel) RandomServer() (*WebSocketServerModel, error) {
	l := len(s)
	if l == 0 {
		return nil, errors.New("No available server ")
	}
	return s[rand.Intn(l)], nil
}

// ReadData read the data in channel.
func (m *WebSocketDownstreamMessage) ReadData(v interface{}) error {
	return json.Unmarshal(m.RawData, v)
}

func NewSubscribeMessage(topic string, privateChannel bool) *WebSocketSubscribeMessage {
	return &WebSocketSubscribeMessage{
		WebSocketMessage: &WebSocketMessage{
			Id:   getMsgId(),
			Type: SubscribeMessage,
		},
		Topic:          topic,
		PrivateChannel: privateChannel,
		Response:       true,
	}
}

type KucoinStreamClient struct {
	// Wait all goroutines quit
	wg *sync.WaitGroup
	// Stop subscribing channel
	done chan struct{}
	// Pong channel to check pong message
	pongs chan string
	// ACK channel to check pong message
	acks chan string
	// Error channel
	errors chan error
	// Downstream message channel
	messages chan *WebSocketDownstreamMessage
	token    *WebSocketTokenModel

	conn       *recws.RecConn
	writeMutex sync.Mutex

	enableHeartbeat bool

	clients map[string]chan []byte
}

func NewKucoinStreamClient(token *WebSocketTokenModel) *KucoinStreamClient {
	return &KucoinStreamClient{
		wg:       &sync.WaitGroup{},
		errors:   make(chan error, 1),
		pongs:    make(chan string, 1),
		acks:     make(chan string, 1),
		messages: make(chan *WebSocketDownstreamMessage, 2048),
		token:    token,

		done:            make(chan struct{}),
		enableHeartbeat: false,
		clients:         make(map[string]chan []byte),
	}
}

func (c *KucoinStreamClient) Connect() (<-chan *WebSocketDownstreamMessage, <-chan error, error) {
	conn := &recws.RecConn{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 5 * time.Second,
		Conn:             nil,
		NonVerbose:       false,
	}

	url := fmt.Sprintf("%s?token=%s", c.token.Servers[0].Endpoint, c.token.Token)
	conn.Dial(url, nil)
	c.conn = conn

	// Must read the first welcome message
	for {
		m := &WebSocketDownstreamMessage{}
		if err := c.conn.ReadJSON(m); err != nil {
			return c.messages, c.errors, err
		}

		if m.Type == ErrorMessage {
			if DebugMode {
				logger.Println("Kucoin error:", m)
			}
			return c.messages, c.errors, errors.Errorf("Error message: %s", helpers.ToJsonString(m))
		}
		if m.Type == WelcomeMessage {
			logger.Println("connected to the kucoin stream websocket")
			break
		}
	}

	c.wg.Add(2)
	go c.read()
	go c.heartbeat()
	return c.messages, c.errors, nil
}

func (c *KucoinStreamClient) Subscribe(channel *WebSocketSubscribeMessage) (*interfaces.Subscription[[]byte], error) {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()

	ch := make(chan []byte, 2048)
	c.clients[channel.Topic] = ch

	if DebugMode {
		logger.Println("Subscribe to channel", channel.Topic)
	}
	if err := c.conn.WriteJSON(channel); err != nil {
		return nil, err
	}

	if err := c.waitForAck(channel.Id); err != nil {
		return nil, err
	}

	return &interfaces.Subscription[[]byte]{
		Stream: ch,
		Unsubscribe: func() {
			// TODO: Implement unsubscribe
			// c.Unsubscribe(NewUnsubscribeMessage(channel.Topic, channel.PrivateChannel))
		},
		Topic: channel.Topic,
	}, nil
}

func (c *KucoinStreamClient) Close() error {
	return c.conn.NetConn().Close()
}

func (c *KucoinStreamClient) CreateMultiplexTunnel(tunnelId string) error {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()

	if DebugMode {
		logger.Println("Create multiplex tunnel")
	}

	if err := c.conn.WriteJSON(map[string]interface{}{
		"id":       getMsgId(),
		"type":     "openTunnel",
		"response": true,
	}); err != nil {
		return err
	}

	return c.waitForAck(tunnelId)
}

func (c *KucoinStreamClient) waitForAck(msgId string) error {
	for {
		select {
		case id := <-c.acks:
			if id != msgId {
				return errors.Errorf("Invalid ack id %s, expect %s", id, msgId)
			}
			if DebugMode {
				logger.Printf("Received ack message %s\n", id)
			}
			return nil
		case err := <-c.errors:
			return errors.Errorf("Subscribe failed, %s", err.Error())
		case <-time.After(kucoinAckTimeout):
			return errors.Errorf("Wait ack message timeout in %v", kucoinAckTimeout)
		}
	}
}

func (c *KucoinStreamClient) read() {
	defer func() {
		close(c.messages)
		close(c.errors)
		c.wg.Done()
	}()

	for {
		select {
		case <-c.done:
			return
		default:
			m := &WebSocketDownstreamMessage{}
			if err := c.conn.ReadJSON(m); err != nil {
				c.errors <- err
				return
			}

			switch m.Type {
			case WelcomeMessage:
			case PongMessage:
				if c.enableHeartbeat {
					c.pongs <- m.Id
				}
			case AckMessage:
				c.acks <- m.Id
			case ErrorMessage:
				c.errors <- errors.Errorf("Error message: %s", helpers.ToJsonString(m))
				return
			case Message, Notice, Command:
				c.messages <- m

				outCh, ok := c.clients[m.Topic]
				if !ok {
					c.errors <- errors.Errorf("Unknown topic: %s", m.Topic)
					logger.Printf("Received message for not required topic: %s", m.Topic)
					continue
				}
				outCh <- m.RawData
			default:
				c.errors <- errors.Errorf("Unknown message type: %s", m.Type)
			}
		}
	}
}

func (c *KucoinStreamClient) heartbeat() {
	c.enableHeartbeat = true
	pt := time.NewTicker(pingInterval)
	defer c.wg.Done()
	defer pt.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-pt.C:
			if DebugMode {
				logger.Println("send ping message")
			}
			c.writeMutex.Lock()
			if err := c.conn.WriteJSON(WebSocketMessage{
				Id:   getMsgId(),
				Type: PingMessage,
			}); err != nil {
				c.errors <- err
				return
			}
			c.writeMutex.Unlock()

			select {
			case <-time.After(kucoinAckTimeout):
				c.errors <- errors.Errorf("Wait pong message timeout in %v", kucoinAckTimeout)
				return
			case <-c.pongs:
			}
		}
	}
}

func getMsgId() string {
	return helpers.IntToString(time.Now().UnixNano())
}
