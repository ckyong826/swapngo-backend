package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// writeWait is how long a single write may block before being abandoned.
	writeWait = 10 * time.Second
	// pongWait is how long we wait for a pong before considering the conn dead.
	pongWait = 60 * time.Second
	// pingPeriod must be less than pongWait so a ping is always in flight.
	pingPeriod = (pongWait * 9) / 10
	// sendBufferSize bounds the per-connection outbound queue.
	sendBufferSize = 16
)

// Event wraps any payload in the standard envelope every socket message uses:
//
//	{ "type": "<EVENT_TYPE>", "data": <payload>, "timestamp": <unix-millis> }
//
// This lets the mobile client switch on a single discriminator (`type`) to
// route price ticks vs. transaction/KYC notifications on one socket.
func Event(eventType string, data any) map[string]any {
	return map[string]any{
		"type":      eventType,
		"data":      data,
		"timestamp": time.Now().UnixMilli(),
	}
}

// Client is a single websocket connection owned by one user. Each client has a
// dedicated writer goroutine, so all writes to the underlying *websocket.Conn
// happen from exactly one goroutine (gorilla forbids concurrent writes).
type Client struct {
	hub    *Hub
	userID string
	conn   *websocket.Conn
	send   chan []byte
}

// Hub tracks live connections, keyed by user ID. A user may have multiple
// concurrent connections (e.g. phone + tablet, or a reconnect race); messages
// fan out to all of them.
type Hub struct {
	clients    map[string]map[*Client]struct{}
	clientsMux sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[*Client]struct{})}
}

// Register wraps a freshly-upgraded connection in a Client, tracks it under the
// user ID, and starts its read/write pumps. The connection is cleaned up
// automatically when it closes or stops responding to pings.
func (h *Hub) Register(userID string, conn *websocket.Conn) {
	c := &Client{
		hub:    h,
		userID: userID,
		conn:   conn,
		send:   make(chan []byte, sendBufferSize),
	}

	h.clientsMux.Lock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*Client]struct{})
	}
	h.clients[userID][c] = struct{}{}
	h.clientsMux.Unlock()

	go c.writePump()
	go c.readPump()
}

// remove unregisters a client and closes its send channel exactly once. Holding
// the write lock here guarantees it cannot run while SendToUser/BroadcastAll are
// iterating (they hold the read lock), so we never write to a closed channel.
func (h *Hub) remove(c *Client) {
	h.clientsMux.Lock()
	defer h.clientsMux.Unlock()
	conns, ok := h.clients[c.userID]
	if !ok {
		return
	}
	if _, ok := conns[c]; ok {
		delete(conns, c)
		close(c.send)
		if len(conns) == 0 {
			delete(h.clients, c.userID)
		}
	}
}

// SendToUser pushes data to every active connection for a user. A slow consumer
// is skipped rather than blocking the caller (the bizs that emit notifications).
func (h *Hub) SendToUser(userID string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	h.clientsMux.RLock()
	defer h.clientsMux.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.send <- payload:
		default: // buffer full / slow client — drop this message for that conn
		}
	}
}

// BroadcastAll pushes data to every connected client (used by the price feed).
func (h *Hub) BroadcastAll(data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	h.clientsMux.RLock()
	defer h.clientsMux.RUnlock()
	for _, conns := range h.clients {
		for c := range conns {
			select {
			case c.send <- payload:
			default:
			}
		}
	}
}

// readPump drains inbound frames (we don't expect any) so gorilla can process
// control frames (pong/close) and surface a closed connection as a read error.
func (c *Client) readPump() {
	defer func() {
		c.hub.remove(c)
		c.conn.Close()
	}()
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

// writePump is the only goroutine that writes to the connection. It flushes
// queued messages and sends periodic pings to keep the connection alive and
// detect dead peers.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel — connection is being torn down.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
