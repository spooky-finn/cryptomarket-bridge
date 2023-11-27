# Binance.dex socket wrapper

Implementation of websocket client base on gorilla/websocket library and Binance.dex data via websocket stream.

## Overview

This client provides following easy to implement functionality

- Support for emitting and receiving text and binary data
- Data compression
- Concurrency control

## Install

```bash
go get github.com/oresdev/binance.dex-socket/binance
```

## Description

Create instance of websocket by passing url of Binance.dex end-point

```go
config := binance.Config{
	URL:             "wss://testnet-dex.binance.org/api/ws",
	OnMessage:       _onMessage,
	OnClose:         _onClose,
	OnConnect:       _onConnect,
	PingPeriod:      15 * time.Second,
	SessionPeriod:   25 * time.Minute,
	ReconnectPeriod: 60 * time.Second,
	IsAutoReconnect: true,
}

client := binance.Connect(config)
```

**Important Note** : url to websocket server must be specified with either **ws** or **wss**.

Set pipes that connect concurrent goroutines

```go
var (
    connected = make(chan bool, 1)
    block     = make(chan bool, 1)
)
```

Receive bytes from Binance.dex end-point

```go
func _onMessage(msg []byte) {
    //
}
```

Set and close new connection

```go
func _onConnect() {
    connected <- true
}

func _onClose(code int, text string) error {
    return nil
}
```

## Usage

Registering all listeners and set subscribe message

```go
import (
	"github.com/oresdev/binance.dex-socket/binance"
)

var (
	connected = make(chan bool, 1)
	block     = make(chan bool, 1)
)

func _onMessage(msg []byte) {
	log.Println("OnMessage: ", string(msg))

	binance.UnmarshalJSON(msg)
}

func _onConnect() {
	log.Println("OnConnect")
	connected <- true
}

func _onClose(code int, text string) error {
	log.Println("OnClose", code, text)
	return nil
}

func main() {
	conf := binance.Conf{
		URL:             "wss://testnet-dex.binance.org/api/ws",
		OnMessage:       _onMessage,
		OnClose:         _onClose,
		OnConnect:       _onConnect,
		PingPeriod:      15 * time.Second,
		ReconnectPeriod: 60 * time.Second,
		SessionPeriod:   1500 * time.Second,
		IsAutoReconnect: true,
	}

	client := binance.Connect(conf)

	go client.Start()

	<-connected

	m, err := json.Marshal(binance.Subscribe{
		Method:  "subscribe",
		Topic:   "transfers",
		Address: "tbnb1qtuf578qs9wfl0wh3vs0r5nszf80gvxd28hkrc",
	})
	if err != nil {
		log.Printf("write error: %v", err)
		return
	}

	client.Subscribe(m)

	<-block
}
```

- Please checkout [**example/binance.dex-socket**](https://github.com/oresdev/binance.dex-socket/tree/master/example) directory for detailed code..
