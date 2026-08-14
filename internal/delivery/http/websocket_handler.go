package http

import (
	"booking-system-api/internal/middleware"
	ws "booking-system-api/internal/websocket"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

type WebsocketHandler struct {
	hub *ws.Hub
}

func NewWebsocketHandler(hub *ws.Hub) *WebsocketHandler {
	return &WebsocketHandler{hub: hub}
}

func (h *WebsocketHandler) Register(r fiber.Router) {
	// Middleware to check if it's a websocket request
	r.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// We apply Auth middleware. This assumes the client sends Authorization header
	// or token in query param.
	// If the frontend sends token via query parameter ?token=xxx, the Auth middleware
	// should support parsing from query param.
	r.Get("/ws", middleware.Auth(), websocket.New(func(c *websocket.Conn) {
		// middleware.Auth() sets LocalUserID ("userID") & LocalUserRole ("userRole") in Locals
		userID, _ := c.Locals(middleware.LocalUserID).(int)
		userRole, _ := c.Locals(middleware.LocalUserRole).(string)

		client := &ws.Client{
			Conn:     c,
			UserID:   userID,
			UserRole: userRole,
			Send:     make(chan []byte, 256),
		}
		
		h.hub.Register <- client

		// Allow collection of memory referenced by the caller by doing all work in
		// new goroutines.
		go client.WritePump()
		client.ReadPump(h.hub)
	}))
}
