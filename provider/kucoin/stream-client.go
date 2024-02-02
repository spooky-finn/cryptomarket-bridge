package kucoin

import (
	"net/http"
	"time"

	"github.com/recws-org/recws"
)

const (
	kucoinDefaultWebsocketEndpoint = "wss://api.kucoin.com"
)

type KucoinStreamClient struct {
	conn *recws.RecConn

	syncAPI *KucoinHttpAPI
}

func NewKucoinStreamClient() *KucoinStreamClient {
	return &KucoinStreamClient{
		conn:    nil,
		syncAPI: NewKucoinHttpAPI(),
	}
}

func (c *KucoinStreamClient) Connect() error {
	conn := &recws.RecConn{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 5 * time.Second,
		Conn:             nil,
		NonVerbose:       false,
	}

	conn.Dial(kucoinDefaultWebsocketEndpoint, nil)
	c.conn = conn
	logger.Println("connected to the kucoin stream websocket")

	return nil
}

func (c *KucoinStreamClient) Close() error {
	return c.conn.NetConn().Close()
}
