package service

import (
	"context"
	"database/sql"
	"log"

	"booking-system-api/internal/repository"
	ws "booking-system-api/internal/websocket"
)

type NotificationService struct {
	q   repository.Querier
	hub *ws.Hub
}

func NewNotificationService(db *sql.DB, hub *ws.Hub) *NotificationService {
	return &NotificationService{q: repository.New(db), hub: hub}
}

func (s *NotificationService) createAndSend(ctx context.Context, userID int32, title, message, notifType string, entityID *int32, data map[string]any) {
	// Save to DB
	var rEntityID sql.NullInt32
	if entityID != nil {
		rEntityID = sql.NullInt32{Int32: *entityID, Valid: true}
	}
	notif, err := s.q.CreateNotification(ctx, repository.CreateNotificationParams{
		UserID:          userID,
		Title:           title,
		Body:            message,
		Type:            notifType,
		RelatedEntityID: rEntityID,
	})
	if err != nil {
		log.Printf("Failed to save notification to DB for user %d: %v", userID, err)
		return
	}

	// Append ID to payload
	data["notificationId"] = notif.ID
	data["isRead"] = notif.IsRead
	data["createdAt"] = notif.CreatedAt

	payload := map[string]any{
		"type":    notifType,
		"title":   title,
		"message": message,
		"data":    data,
	}

	s.hub.SendToUser(int(userID), payload)
}

func (s *NotificationService) NotifyDriver(driverUserID int32, message string, data map[string]any) {
	log.Printf("Notifying driver (userID=%d): %s", driverUserID, message)
	var entityID *int32
	if bID, ok := data["bookingId"].(int32); ok {
		entityID = &bID
	}
	s.createAndSend(context.Background(), driverUserID, "New Assignment", message, "NEW_BOOKING", entityID, data)
}

func (s *NotificationService) NotifyUser(userID int32, message string, data map[string]any) {
	log.Printf("Notifying user (userID=%d): %s", userID, message)
	var entityID *int32
	if bID, ok := data["bookingId"].(int32); ok {
		entityID = &bID
	}
	s.createAndSend(context.Background(), userID, "Booking Update", message, "BOOKING_APPROVED", entityID, data)
}

func (s *NotificationService) GetMyNotifications(ctx context.Context, userID int32, page, limit int) (any, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	rows, err := s.q.ListNotificationsByUserID(ctx, repository.ListNotificationsByUserIDParams{
		UserID: userID,
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}

	total, err := s.q.CountNotificationsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"data": rows,
		"pagination": map[string]any{
			"total":      total,
			"page":       page,
			"limit":      limit,
			"totalPages": (int(total) + limit - 1) / limit,
		},
	}, nil
}

func (s *NotificationService) MarkAsRead(ctx context.Context, notifID int32, userID int32) error {
	return s.q.MarkNotificationAsRead(ctx, repository.MarkNotificationAsReadParams{
		ID:     notifID,
		UserID: userID,
	})
}

func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID int32) error {
	return s.q.MarkAllNotificationsAsRead(ctx, userID)
}
