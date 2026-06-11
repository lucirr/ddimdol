package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/didimdol/portal-api/internal/hub"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // DEV: allow all origins; restrict in production
	},
}

// WsHandler upgrades HTTP connections to WebSocket and delegates to the Hub.
type WsHandler struct {
	hub    *hub.Hub
	logger *zap.Logger
}

// NewWsHandler creates a WsHandler backed by the given Hub.
func NewWsHandler(h *hub.Hub, logger *zap.Logger) *WsHandler {
	return &WsHandler{hub: h, logger: logger}
}

// HandleEdgeEvents upgrades the request to a WebSocket connection and streams
// edge events to the client until it disconnects.
// The client's tenant_id (from the JWT set by Auth middleware) is bound to the
// connection so BroadcastTenant can filter events by tenant.
// central-operators have an empty tenant_id claim and receive all events.
func (h *WsHandler) HandleEdgeEvents(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("ws upgrade failed", zap.Error(err))
		return
	}

	tenantID, _ := c.Get("tenant_id")
	tenantIDStr, _ := tenantID.(string)

	h.hub.ServeClient(conn, tenantIDStr)
}
