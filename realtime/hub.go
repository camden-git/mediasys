package realtime

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Event represents a message sent to websocket clients
type Event struct {
	Type      string                 `json:"type"`
	Path      string                 `json:"path,omitempty"`
	Task      string                 `json:"task,omitempty"`
	Status    string                 `json:"status,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

// Hub is a simple global pubsub for websocket clients
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Broadcast(event Event) {
	encoded, err := json.Marshal(event)
	if err != nil {
		log.Printf("realtime: failed to marshal event: %v", err)
		return
	}
	select {
	case h.broadcast <- encoded:
	default:
		log.Printf("realtime: dropping event, broadcast channel full")
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeWS upgrades the connection and registers a client
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	// echo back the negotiated subprotocol (if any) so the handshake completes
	// cleanly; the auth token itself is carried as a subprotocol value rather
	// than a query parameter so it never ends up in access logs (see AuthMiddleware
	// call site in routes.go, which reads it off Sec-WebSocket-Protocol).
	var respHeader http.Header
	if protos := websocket.Subprotocols(r); len(protos) > 0 {
		respHeader = http.Header{"Sec-WebSocket-Protocol": {protos[0]}}
	}

	conn, err := upgrader.Upgrade(w, r, respHeader)
	if err != nil {
		log.Printf("realtime: websocket upgrade error: %v", err)
		return
	}
	client := &Client{conn: conn, send: make(chan []byte, 256)}
	h.register <- client

	// writer
	go func() {
		for msg := range client.send {
			if err := client.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				break
			}
		}
		client.conn.Close()
	}()

	// reader (just consume pings/close)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
	h.unregister <- client
}
