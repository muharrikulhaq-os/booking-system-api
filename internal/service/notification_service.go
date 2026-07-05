package service

import (
	"log"
	ws "booking-system-api/internal/websocket"
)

type NotificationService struct {
	hub *ws.Hub
}

func NewNotificationService(hub *ws.Hub) *NotificationService {
	return &NotificationService{hub: hub}
}

func (s *NotificationService) NotifyDriver(driverUserID int32, message string, data map[string]any) {
	log.Printf("Notifying driver (userID=%d): %s", driverUserID, message)
	payload := map[string]any{
		"type":    "NEW_BOOKING",
		"message": message,
		"data":    data,
	}
	s.hub.SendToUser(int(driverUserID), payload)
}

func (s *NotificationService) NotifyUser(userID int32, message string, data map[string]any) {
	log.Printf("Notifying user (userID=%d): %s", userID, message)
	payload := map[string]any{
		"type":    "BOOKING_APPROVED",
		"message": message,
		"data":    data,
	}
	s.hub.SendToUser(int(userID), payload)
}
