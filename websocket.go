package tracks

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/net/websocket"
)

type WebSocketConn struct {
	*websocket.Conn
	request *http.Request
}

func (c *WebSocketConn) Context() context.Context {
	return c.request.Context()
}

func (c *WebSocketConn) Param(name string) string {
	return c.request.PathValue(name)
}

type WebSocketHandler interface {
	Handle(conn *WebSocketConn)
}

type WebSocketHub struct {
	mu          sync.RWMutex
	connections map[string]map[*WebSocketConn]bool // room -> connections
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		connections: make(map[string]map[*WebSocketConn]bool),
	}
}

func (h *WebSocketHub) Join(room string, conn *WebSocketConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.connections[room] == nil {
		h.connections[room] = make(map[*WebSocketConn]bool)
	}
	h.connections[room][conn] = true
}

func (h *WebSocketHub) Leave(room string, conn *WebSocketConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.connections[room] != nil {
		delete(h.connections[room], conn)
		if len(h.connections[room]) == 0 {
			delete(h.connections, room)
		}
	}
}

func (h *WebSocketHub) Broadcast(room string, data any) error {
	h.mu.RLock()
	conns := h.connections[room]
	h.mu.RUnlock()

	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	for conn := range conns {
		_, _ = conn.Write(b)
	}
	return nil
}

func (r *router) WebSocket(path string, handler WebSocketHandler, mws ...MiddlewareBuilder) Router {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	wsHandler := websocket.Handler(func(ws *websocket.Conn) {
		req := ws.Request()
		conn := &WebSocketConn{Conn: ws, request: req}
		handler.Handle(conn)
	})

	// Wrap with middlewares
	h, err := r.requestMiddlewares.Wrap(r, wsHandler, mws...)
	if err != nil {
		panic(err)
	}

	r.mux.Handle("GET "+path, h)
	return r
}
