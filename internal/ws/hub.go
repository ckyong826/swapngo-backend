package ws

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	// 存储所有活跃连接，key 可以是 userID
	clients    map[string]*websocket.Conn
	clientsMux sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]*websocket.Conn),
	}
}

// Register 注册一个新连接
func (h *Hub) Register(userID string, conn *websocket.Conn) {
	h.clientsMux.Lock()
	defer h.clientsMux.Unlock()
	h.clients[userID] = conn
}

// Unregister 断开连接
func (h *Hub) Unregister(userID string) {
	h.clientsMux.Lock()
	defer h.clientsMux.Unlock()
	if conn, ok := h.clients[userID]; ok {
		conn.Close()
		delete(h.clients, userID)
	}
}

// BroadcastToUser 推送给特定用户
func (h *Hub) SendToUser(userID string, data any) {
	h.clientsMux.RLock()
	conn, ok := h.clients[userID]
	h.clientsMux.RUnlock()

	if ok {
		payload, _ := json.Marshal(data)
		conn.WriteMessage(websocket.TextMessage, payload)
	}
}