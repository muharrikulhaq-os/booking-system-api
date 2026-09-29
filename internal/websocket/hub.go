package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gofiber/websocket/v2"
)

// Client represents a single websocket connection
type Client struct {
	Conn     *websocket.Conn
	UserID   int
	UserRole string
	Send     chan []byte
}

// Hub manages websocket clients and broadcasts messages
type Hub struct {
	Clients    map[*Client]bool
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan []byte
	mu         sync.RWMutex
}

// NewHub creates a new websocket hub
func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan []byte),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			h.Clients[client] = true
			h.mu.Unlock()
			log.Printf("Client registered: userID=%d role=%s", client.UserID, client.UserRole)
		case client := <-h.Unregister:
			h.mu.Lock()
			if _, ok := h.Clients[client]; ok {
				delete(h.Clients, client)
				close(client.Send)
				log.Printf("Client unregistered: userID=%d role=%s", client.UserID, client.UserRole)
			}
			h.mu.Unlock()
		case message := <-h.Broadcast:
			// Lock penuh (bukan RLock): client yang buffernya penuh dihapus
			// dari map di dalam loop ini.
			h.mu.Lock()
			for client := range h.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.Clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

// SendToAll mengirim pesan ke SEMUA client yang terhubung (mis. event
// DATA_CHANGED). Non-blocking: client yang buffernya penuh dilewati.
func (h *Hub) SendToAll(message interface{}) {
	b, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling WS message: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.Clients {
		select {
		case client.Send <- b:
		default:
			// Buffer full or closed
		}
	}
}

// SendToUser sends a message to a specific user
func (h *Hub) SendToUser(userID int, message interface{}) {
	b, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling WS message: %v", err)
		return
	}
	
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.Clients {
		if client.UserID == userID {
			select {
			case client.Send <- b:
			default:
				// Buffer full or closed
			}
		}
	}
}

// SendToRole sends a message to all users with a specific role
func (h *Hub) SendToRole(role string, message interface{}) {
	b, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling WS message: %v", err)
		return
	}
	
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.Clients {
		if client.UserRole == role {
			select {
			case client.Send <- b:
			default:
				// Buffer full or closed
			}
		}
	}
}

// ReadPump pumps messages from the websocket connection to the hub.
func (c *Client) ReadPump(hub *Hub) {
	defer func() {
		hub.Unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })
	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
	}
}

// WritePump pumps messages from the hub to the websocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(50 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Satu pesan JSON = satu frame. Klien mem-parse tiap frame sebagai
			// satu objek JSON; menggabungkan pesan antrean dengan '\n' ke satu
			// frame membuat semuanya gagal di-parse (pesan hilang).
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
