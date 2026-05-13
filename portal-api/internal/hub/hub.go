package hub

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Client represents a single connected WebSocket client.
type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	logger *zap.Logger
}

// Hub maintains the set of active clients and broadcasts messages to them.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	logger     *zap.Logger
}

// New creates a Hub ready to be started with Run.
func New(logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		logger:     logger,
	}
}

// Run processes register/unregister/broadcast events. Call in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Debug("ws client connected", zap.Int("total", len(h.clients)))
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- msg:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast serialises a typed event envelope and sends it to all connected clients.
func (h *Hub) Broadcast(eventType string, payload any) {
	msg, err := json.Marshal(map[string]any{
		"type":    eventType,
		"payload": payload,
	})
	if err != nil {
		h.logger.Error("ws broadcast marshal", zap.Error(err))
		return
	}
	select {
	case h.broadcast <- msg:
	default:
		h.logger.Warn("ws broadcast channel full, dropping message")
	}
}

// ServeClient registers the connection, then blocks reading until the client disconnects.
func (h *Hub) ServeClient(conn *websocket.Conn) {
	client := &Client{conn: conn, send: make(chan []byte, 64), logger: h.logger}
	h.register <- client

	go client.writePump()
	client.readPump(h)
}

func (c *Client) writePump() {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			c.logger.Debug("ws write error", zap.Error(err))
			return
		}
	}
}

func (c *Client) readPump(h *Hub) {
	defer func() {
		h.unregister <- c
		c.conn.Close()
	}()
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
