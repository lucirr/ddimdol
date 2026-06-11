package hub

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Client represents a single connected WebSocket client.
type Client struct {
	conn     *websocket.Conn
	send     chan []byte
	tenantID string // empty string means central-operator (sees all tenants)
	logger   *zap.Logger
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

// Broadcast sends an event to all connected clients regardless of tenant.
// Use only for system-wide events (e.g. platform notifications).
func (h *Hub) Broadcast(eventType string, payload any) {
	msg := marshal(h.logger, eventType, payload)
	if msg == nil {
		return
	}
	select {
	case h.broadcast <- msg:
	default:
		h.logger.Warn("ws broadcast channel full, dropping message")
	}
}

// BroadcastTenant sends an event only to clients whose tenantID matches.
// Clients registered with an empty tenantID (central-operators) receive
// events for every tenant.
func (h *Hub) BroadcastTenant(tenantID, eventType string, payload any) {
	msg := marshal(h.logger, eventType, payload)
	if msg == nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients {
		if client.tenantID != "" && client.tenantID != tenantID {
			continue
		}
		select {
		case client.send <- msg:
		default:
			h.logger.Warn("ws send buffer full, dropping message for client")
		}
	}
}

// ServeClient registers the connection with the given tenantID, then blocks
// reading until the client disconnects. Pass an empty tenantID for
// central-operators who should receive events for all tenants.
func (h *Hub) ServeClient(conn *websocket.Conn, tenantID string) {
	client := &Client{
		conn:     conn,
		send:     make(chan []byte, 64),
		tenantID: tenantID,
		logger:   h.logger,
	}
	h.register <- client

	go client.writePump()
	client.readPump(h)
}

func marshal(logger *zap.Logger, eventType string, payload any) []byte {
	msg, err := json.Marshal(map[string]any{
		"type":    eventType,
		"payload": payload,
	})
	if err != nil {
		logger.Error("ws marshal", zap.Error(err))
		return nil
	}
	return msg
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
