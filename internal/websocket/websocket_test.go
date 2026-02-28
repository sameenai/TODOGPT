package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ws "github.com/gorilla/websocket"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()
	if hub == nil {
		t.Fatal("expected non-nil hub")
	}
	if hub.clients == nil {
		t.Error("expected initialized clients map")
	}
	if hub.broadcast == nil {
		t.Error("expected initialized broadcast channel")
	}
}

func TestHubRegisterAndUnregister(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client := &Client{
		hub:  hub,
		send: make(chan []byte, 256),
	}

	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	hub.mu.RLock()
	if len(hub.clients) != 1 {
		t.Errorf("expected 1 client, got %d", len(hub.clients))
	}
	hub.mu.RUnlock()

	hub.unregister <- client
	time.Sleep(50 * time.Millisecond)

	hub.mu.RLock()
	if len(hub.clients) != 0 {
		t.Errorf("expected 0 clients after unregister, got %d", len(hub.clients))
	}
	hub.mu.RUnlock()
}

func TestHubBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	client1 := &Client{hub: hub, send: make(chan []byte, 256)}
	client2 := &Client{hub: hub, send: make(chan []byte, 256)}

	hub.register <- client1
	hub.register <- client2
	time.Sleep(50 * time.Millisecond)

	msg := []byte(`{"type":"test"}`)
	hub.Broadcast(msg)
	time.Sleep(50 * time.Millisecond)

	select {
	case received := <-client1.send:
		if string(received) != string(msg) {
			t.Errorf("client1: expected %q, got %q", msg, received)
		}
	default:
		t.Error("client1 did not receive broadcast")
	}

	select {
	case received := <-client2.send:
		if string(received) != string(msg) {
			t.Errorf("client2: expected %q, got %q", msg, received)
		}
	default:
		t.Error("client2 did not receive broadcast")
	}
}

func TestWebSocketUpgrade(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, w, r)
	}))
	defer server.Close()

	// Convert http URL to ws URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/"

	// Connect WebSocket client
	conn, resp, err := ws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial error: %v", err)
	}
	defer conn.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		t.Errorf("expected 101 Switching Protocols, got %d", resp.StatusCode)
	}

	// Wait for registration
	time.Sleep(100 * time.Millisecond)

	hub.mu.RLock()
	clientCount := len(hub.clients)
	hub.mu.RUnlock()

	if clientCount != 1 {
		t.Errorf("expected 1 connected client, got %d", clientCount)
	}
}

func TestWebSocketReceivesBroadcast(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/"
	conn, _, err := ws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket dial error: %v", err)
	}
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)

	// Send a broadcast
	testMsg := []byte(`{"type":"test_broadcast","payload":"hello"}`)
	hub.Broadcast(testMsg)

	// Read the message
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}

	if string(msg) != string(testMsg) {
		t.Errorf("expected %q, got %q", testMsg, msg)
	}
}

func TestWebSocketMultipleClients(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/"

	var conns []*ws.Conn
	for i := 0; i < 3; i++ {
		conn, _, err := ws.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("client %d dial error: %v", i, err)
		}
		conns = append(conns, conn)
	}
	defer func() {
		for _, c := range conns {
			c.Close()
		}
	}()

	time.Sleep(100 * time.Millisecond)

	hub.mu.RLock()
	if len(hub.clients) != 3 {
		t.Errorf("expected 3 clients, got %d", len(hub.clients))
	}
	hub.mu.RUnlock()

	// Broadcast should reach all
	testMsg := []byte(`{"type":"multi_test"}`)
	hub.Broadcast(testMsg)

	for i, conn := range conns {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("client %d read error: %v", i, err)
		}
		if string(msg) != string(testMsg) {
			t.Errorf("client %d: expected %q, got %q", i, testMsg, msg)
		}
	}
}

func TestWebSocketDisconnect(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ServeWs(hub, w, r)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/"
	conn, _, err := ws.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	hub.mu.RLock()
	if len(hub.clients) != 1 {
		t.Errorf("expected 1 client before disconnect, got %d", len(hub.clients))
	}
	hub.mu.RUnlock()

	conn.Close()
	time.Sleep(200 * time.Millisecond)

	hub.mu.RLock()
	if len(hub.clients) != 0 {
		t.Errorf("expected 0 clients after disconnect, got %d", len(hub.clients))
	}
	hub.mu.RUnlock()
}

func TestUpgraderCheckOrigin(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "http://evil.com")
	if !upgrader.CheckOrigin(req) {
		t.Error("expected CheckOrigin to allow all origins for localhost dev")
	}
}

func TestBroadcastMethod(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Broadcast with no clients should not panic
	hub.Broadcast([]byte("test"))
	time.Sleep(50 * time.Millisecond)
}
