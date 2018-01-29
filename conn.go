package websocket

import (
	"log"
	"net"
	"sync"

	websocket "github.com/gorilla/websocket"
)

// Conn wraps a websocket.Conn to provide channels for reading and writing
type Conn struct {
	ID          string
	ReceiveChan chan []byte
	sendChan    chan []byte
	ClientAddr  net.Addr
	wsConn      *websocket.Conn
	closeLock   *sync.Mutex
	closed      bool
}

// NewConnHandler is called whenever a new connection to this WebSocket server is created.
// It receives the relevant Conn, allowing for easy access to its ReceiveChan and SendChan.
type NewConnHandler func(*Conn)

// Send sends a message through the connection.
// It is safe to use if the connection is closed, because
// it will return an error indicating that condition.
func (c *Conn) Send(msg []byte) error {
	c.closeLock.Lock()
	if c.closed {
		c.closeLock.Unlock()
		return ErrConnClosed
	}
	c.sendChan <- msg
	c.closeLock.Unlock()
	return nil
}

func (c *Conn) LocalAddrStr() string {
	return c.wsConn.LocalAddr().String()
}

func newConnection(key string, wsConn *websocket.Conn) *Conn {
	c := &Conn{
		ID:          key,
		ReceiveChan: make(chan []byte),
		sendChan:    make(chan []byte),
		ClientAddr:  wsConn.RemoteAddr(),
		wsConn:      wsConn,
		closeLock:   &sync.Mutex{},
		closed:      false,
	}
	// Start the sender and receiver
	go func() {
		killChan := make(chan struct{})
		go c.sender(killChan)
		c.receiver()

		// set the closed flag so Send method won't work
		c.closeLock.Lock()
		c.closed = true
		c.closeLock.Unlock()

		close(killChan)
		close(c.sendChan)
		close(c.ReceiveChan)
	}()

	return c
}

func (c *Conn) sender(killChan chan struct{}) {
	sendMessage := func(msg []byte) {
		if err := c.wsConn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
			log.Println("Unable to send message:", err)
		}
	}

	for {
		select {
		case <-killChan:
			c.wsConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			c.wsConn.Close()
			return
		case msg := <-c.sendChan:
			sendMessage(msg)
		}
	}
}

func (c *Conn) receiver() {
	readMessage := func() ([]byte, error) {
		msgType, msg, err := c.wsConn.ReadMessage()
		if err != nil {
			return msg, err
		}
		if msgType != websocket.BinaryMessage {
			return msg, errNotABinaryMessage
		}
		return msg, nil
	}

	for {
		msg, err := readMessage()
		if err != nil {
			// A read error here is permanent, so leave the loop
			log.Println("Unable to read message:", err)
			return
		}
		c.ReceiveChan <- msg
	}
}
