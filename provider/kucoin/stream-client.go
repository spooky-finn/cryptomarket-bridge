package kucoin

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"sync"

	"github.com/pkg/errors"
	"github.com/spooky-finn/cryptobridge/config"
	"github.com/spooky-finn/cryptobridge/domain"
	"github.com/spooky-finn/cryptobridge/helpers"

	"github.com/gorilla/websocket"
)

const (
	kucoinDefaultTimeout           = time.Second * 5
	kucoinDefaultWebsocketEndpoint = "wss://api.kucoin.com"
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
	// // Pong channel to check pong message
	pongs chan string
	// // ACK channel to check pong message
	acks chan string
	// // Error channel
	// errors chan error
	// // Downstream message channel
	// messages chan *WebSocketDownstreamMessage
	token *WebSocketTokenModel

	conn       *websocket.Conn
	writeMutex sync.Mutex

	enableHeartbeat bool
	pintInterval    time.Duration
	pingTimeout     time.Duration

	clients  map[string]chan []byte
	tunnelId string
}

func NewKucoinStreamClient(token *WebSocketTokenModel) *KucoinStreamClient {
	return &KucoinStreamClient{
		wg: &sync.WaitGroup{},
		// errors:   make(chan error, 1),
		pongs: make(chan string, 1),
		acks:  make(chan string, 1),
		// messages: make(chan *WebSocketDownstreamMessage, 2048),
		token: token,

		done:            make(chan struct{}),
		enableHeartbeat: false,
		pintInterval:    10 * time.Second,
		pingTimeout:     time.Duration(token.Servers[0].PingTimeout) * time.Millisecond,
		clients:         make(map[string]chan []byte),
	}
}

func (c *KucoinStreamClient) Connect() error {
	dialer := &websocket.Dialer{
		Proxy:           http.ProxyFromEnvironment,
		ReadBufferSize:  512,
		WriteBufferSize: 256,
	}

	url := fmt.Sprintf("%s?token=%s", c.token.Servers[0].Endpoint, c.token.Token)
	if config.DebugMode {
		logger.Printf("connection to the kucoin stream websocket: %s\n", url)
	}
	conn, httpResp, err := dialer.Dial(url, nil)
	if err != nil {
		return errors.Wrapf(err, "Failed to dial to the kucoin stream websocket. status: %s", httpResp.Status)
	}
	c.conn = conn

	// Must read the first welcome message
	for {
		m := &WebSocketDownstreamMessage{}
		if err := c.conn.ReadJSON(m); err != nil {
			return err
		}

		if m.Type == ErrorMessage {
			logger.Println("Kucoin error:", m)
			return errors.Errorf("Error message: %s", helpers.ToJsonString(m))
		}
		if m.Type == WelcomeMessage {
			logger.Println("connected to the kucoin stream websocket", m.WebSocketMessage)
			break
		}
	}

	c.wg.Add(2)
	go c.read()
	go c.keepHeartbeat()
	return nil
}

func (c *KucoinStreamClient) Subscribe(channel *WebSocketSubscribeMessage) (*domain.Subscription[[]byte], error) {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()

	ch := make(chan []byte, 2048)
	c.clients[channel.Topic] = ch

	if config.DebugMode {
		logger.Println("Subscribing to channel", channel.Topic)
	}
	if err := c.conn.WriteJSON(channel); err != nil {
		return nil, err
	}

	if err := c.waitForAck(channel.Id); err != nil {
		return nil, err
	}

	if config.DebugMode {
		logger.Println("succesfully subscribet to the topic=", channel.Topic)
	}

	return &domain.Subscription[[]byte]{
		Stream: ch,
		Unsubscribe: func() {
			// TODO: Implement unsubscribe
			// c.Unsubscribe(NewUnsubscribeMessage(channel.Topic, channel.PrivateChannel))
			panic("Unsubscribe not implemented")
		},
		Topic: channel.Topic,
	}, nil
}

func (c *KucoinStreamClient) Close() error {
	close(c.done)
	close(c.acks)
	close(c.pongs)
	c.wg.Wait()

	return c.conn.NetConn().Close()
}

func (c *KucoinStreamClient) CreateMultiplexTunnel(tunnelId string) error {
	c.tunnelId = tunnelId
	mId := getMsgId()
	err := c.SendMessage(map[string]interface{}{
		"id":          mId,
		"type":        "openTunnel",
		"newTunnelId": tunnelId,
		"response":    true,
	})

	if err != nil {
		return err
	}

	return c.waitForAck(mId)
}

func (c *KucoinStreamClient) waitForAck(msgId string) error {
	for {
		select {
		case id := <-c.acks:
			if id != msgId {
				return errors.Errorf("Invalid ack id (%s), expect (%s)", id, msgId)
			}
			if config.DebugMode {
				logger.Printf("received ack message: %s\n", id)
			}
			return nil
		case <-time.After(kucoinDefaultTimeout):
			return errors.Errorf("triggered wait ack message timeout in %v", kucoinDefaultTimeout)
		}
	}
}

func (c *KucoinStreamClient) read() {
	defer func() {
		close(c.pongs)
		c.wg.Done()
	}()

	for {
		select {
		case <-c.done:
			return
		default:
			m := &WebSocketDownstreamMessage{}
			if err := c.conn.ReadJSON(m); err != nil {
				logger.Fatalf("err while reading message from web conn: %s", err.Error())
				return
			}

			if config.DebugMode {
				logger.Println("Received message: ", string(m.RawData))
			}

			switch m.Type {
			case WelcomeMessage:
			case PongMessage:
				c.pongs <- m.Id
			case AckMessage:
				c.acks <- m.Id
			case ErrorMessage:
				logger.Printf("Error message: %s", string(m.RawData))

			case Message, Notice, Command:
				outCh, ok := c.clients[m.Topic]
				if !ok {
					logger.Printf("Received message for not subscribed topic: %s, msg %#v", m.Topic, string(m.RawData))
					continue
				}
				outCh <- m.RawData
			}
		}
	}
}

func (c *KucoinStreamClient) SendMessage(msg interface{}) error {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()

	if config.DebugMode {
		logger.Println("Send message:", helpers.ToJsonString(msg))
	}

	return c.conn.WriteJSON(msg)
}

func (c *KucoinStreamClient) keepHeartbeat() {
	c.enableHeartbeat = true
	if c.pintInterval == 0 {
		panic("Heartbeat interval is not set")
	}

	pt := time.NewTicker(c.pintInterval)
	defer c.wg.Done()
	defer pt.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-pt.C:
			if config.DebugMode {
				logger.Println("sending ping message")
			}
			err := c.SendMessage(WebSocketMessage{Id: getMsgId(), Type: PingMessage})
			if err != nil {
				logger.Printf("err while writing ping message to the conn: %s", err.Error())
			}

			select {
			case <-time.After(c.pingTimeout):
				logger.Println("failed to receive pong message. connection will be closed")
				c.Close()
				return
			case <-c.pongs:
				if config.DebugMode {
					logger.Println(`received pong message`)
				}
			}
		}
	}
}

func getMsgId() string {
	return helpers.IntToString(time.Now().UnixNano())
}
