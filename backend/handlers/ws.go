package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Hub fans payment events out to driver dashboards by vehicle code.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*websocket.Conn]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*websocket.Conn]struct{})}
}

func (h *Hub) Subscribe(code string, c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[code] == nil {
		h.clients[code] = make(map[*websocket.Conn]struct{})
	}
	h.clients[code][c] = struct{}{}
}

func (h *Hub) Unsubscribe(code string, c *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.clients[code]; ok {
		delete(m, c)
		if len(m) == 0 {
			delete(h.clients, code)
		}
	}
	_ = c.Close()
}

func (h *Hub) Broadcast(code string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[code] {
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("ws write %s: %v", code, err)
		}
	}
}

type WSHandler struct {
	Hub *Hub
}

func (w *WSHandler) DriverSocket(c *gin.Context) {
	code := c.Param("code")
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	w.Hub.Subscribe(code, conn)
	defer w.Hub.Unsubscribe(code, conn)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
