package realtime

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/camden-git/mediasysbackend/models"
	"github.com/gorilla/websocket"
)

// apiErrorResponse mirrors handlers.APIErrorResponse so the websocket upgrade
// path (which cannot import handlers without an import cycle) returns the
// same standardized error shape as the rest of the API.
type apiErrorResponse struct {
	Errors []apiErrorDetail `json:"errors"`
}

type apiErrorDetail struct {
	Code   string `json:"code"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// writeUpgradeError writes a websocket handshake failure using the standard
// API error response shape.
func writeUpgradeError(w http.ResponseWriter, status int, reason error) {
	detail := http.StatusText(status)
	if reason != nil {
		detail = reason.Error()
	}
	resp := apiErrorResponse{
		Errors: []apiErrorDetail{
			{
				Code:   "WebSocketUpgradeError",
				Status: strconv.Itoa(status),
				Detail: detail,
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

// Event represents a message sent to websocket clients. AlbumID scopes delivery
// to users who may view that album's content.
type Event struct {
	Type      string                 `json:"type"`
	AlbumID   uint                   `json:"album_id,omitempty"`
	Path      string                 `json:"path,omitempty"`
	Task      string                 `json:"task,omitempty"`
	Status    string                 `json:"status,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

const (
	// permissions required to see an album's events: the global one matches the
	// admin album images listing, the album-scoped one its per-album counterpart.
	globalViewPermission = "album.list"
	albumViewPermission  = "album.view.content"

	maxReadSize = 4096
	pongWait    = 60 * time.Second
	pingPeriod  = 50 * time.Second
	writeWait   = 10 * time.Second
)

type Client struct {
	conn *websocket.Conn
	send chan []byte
	user *models.User
}

// canView reports whether the client's user may receive the event.
func (c *Client) canView(ev Event) bool {
	if c.user == nil {
		return false
	}
	if c.user.HasGlobalPermission(globalViewPermission) {
		return true
	}
	return ev.AlbumID != 0 && c.user.HasAlbumPermission(ev.AlbumID, albumViewPermission)
}

type message struct {
	event Event
	data  []byte
}

// Hub is a simple pubsub for websocket clients; events are only delivered to
// clients whose user may view the event's album.
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan message
	mu         sync.RWMutex
	done       chan struct{}
	stopOnce   sync.Once
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan message, 256),
		done:       make(chan struct{}),
	}
}

// Stop ends Run and disconnects every client. It is safe to call more than once.
func (h *Hub) Stop() {
	h.stopOnce.Do(func() { close(h.done) })
}

func (h *Hub) Run() {
	for {
		select {
		case <-h.done:
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return
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
		case msg := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				if !client.canView(msg.event) {
					continue
				}
				select {
				case client.send <- msg.data:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
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
	case h.broadcast <- message{event: event, data: encoded}:
	default:
		log.Printf("realtime: dropping event, broadcast channel full")
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
	Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {
		writeUpgradeError(w, status, reason)
	},
}

// ServeWS upgrades the connection and registers a client for the given
// (already authenticated) user.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request, user *models.User) {
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
	if user != nil {
		user.HasGlobalPermission("") // compute effective permissions now, before the hub goroutine reads them
	}
	client := &Client{conn: conn, send: make(chan []byte, 256), user: user}
	select {
	case h.register <- client:
	case <-h.done:
		_ = conn.Close()
		return
	}

	// writer
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		defer client.conn.Close()
		for {
			select {
			case msg, ok := <-client.send:
				_ = client.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if !ok {
					_ = client.conn.WriteMessage(websocket.CloseMessage, nil)
					return
				}
				if err := client.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			case <-ticker.C:
				_ = client.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}()

	// reader: clients send nothing, so this only services pongs and detects dead connections
	conn.SetReadLimit(maxReadSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
	select {
	case h.unregister <- client:
	case <-h.done:
	}
}
