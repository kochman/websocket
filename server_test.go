package websocket

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Sets up a server and returns it, along with server Conn and client websocket.Conn connected to it
func setUpServer(t *testing.T, h NewConnHandler) (*Server, *Conn, *websocket.Conn) {
	s, _ := NewServer(h)
	srv := httptest.NewServer(s)
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"

	clientConn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		t.Fatalf("Cannot make websocket connection: %v", err)
	}

	// loop until the connection is recorded
	numConnections := 0
	for numConnections != 1 {
		numConnections = len(s.Connections())
	}
	serverConn := s.Connections()[0]

	return s, serverConn, clientConn
}

func TestWebSocketServerReceive(t *testing.T) {
	_, serverConn, clientConn := setUpServer(t, nil)
	err := clientConn.WriteMessage(websocket.BinaryMessage, []byte("hello"))
	if err != nil {
		t.Fatalf("Cannot write message: %v", err)
	}

	msg := <-serverConn.ReceiveChan
	if string(msg) != "hello" {
		t.Errorf("Expected \"hello\", got \"%v\"", string(msg))
	}
}

func TestWebSocketServerSend(t *testing.T) {
	_, serverConn, clientConn := setUpServer(t, nil)
	err := serverConn.Send([]byte("hello"))
	if err != nil {
		t.Error(err)
	}
	msgType, msg, err := clientConn.ReadMessage()
	if err != nil {
		t.Errorf("Unable to read message: %v", err)
	}
	if msgType != websocket.BinaryMessage {
		t.Error("Expected binary message")
	}
	if string(msg) != "hello" {
		t.Errorf("Expected \"hello\", got \"%v\"", string(msg))
	}
}

// Test that the NewConnHandler passed into NewServer is called on a new connection,
// and test that we can write to it.
func TestWebSocketServerNewConnHandler(t *testing.T) {
	hit := make(chan struct{})
	handler := func(c *Conn) {
		err := c.Send([]byte("hello"))
		if err != nil {
			t.Error(err)
		}
		close(hit)
	}

	_, _, clientConn := setUpServer(t, handler)
	select {
	case <-hit:
	case <-time.After(time.Second):
		t.Error("Handler didn't close channel")
	}

	_, msg, _ := clientConn.ReadMessage()
	if string(msg) != "hello" {
		t.Errorf("Expected \"hello\", got \"%v\"", string(msg))
	}
}

// This isn't a WebSocket handshake, so we expect an error.
func TestWebSocketServerUpgradeFailure(t *testing.T) {
	s, _ := NewServer(nil)
	srv := httptest.NewServer(s)
	resp, err := http.Get(srv.URL)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("Unexpected response code: %v", resp.StatusCode)
	}
}
