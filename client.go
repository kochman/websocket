package websocket

import (
	"errors"
	"log"
	"time"

	gws "github.com/gorilla/websocket"
)

var (
	ErrNoUrlProvided = errors.New("No URL provided")
)

type Client struct {
	url         string
	connection  *Conn
	connHandler NewConnHandler
}

func NewClient(websocketURL string, connHandler NewConnHandler) (*Client, error) {
	if len(websocketURL) <= 0 {
		return nil, ErrNoUrlProvided
	}
	c := &Client{
		url:     websocketURL,
		connHandler: connHandler,
	}
	go c.start()
	return c, nil
}

func (c *Client) Connection() *Conn {
	return c.connection
}

func (c *Client) start() {
	for {
		wsConn, _, err := gws.DefaultDialer.Dial(c.url, nil)
		if err != nil {
			log.Println("Unable to create WebSocket connection:", err)
			time.Sleep(time.Second)
			continue
		}
		// creates communication channels and kicks off handling
		c.connection = newConnection(c.url, wsConn)

		// if connHandler returns, the connection needs to be reset
		c.connHandler(c.connection)
	}
}