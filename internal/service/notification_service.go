package service

import (
	"context"
	"database/sql"
	"log"
	"strconv"

	"booking-system-api/internal/config"
	"booking-system-api/internal/repository"
	ws "booking-system-api/internal/websocket"
)

type NotificationService struct {
	q   repository.ExtendedQuerier
	hub *ws.Hub
	fcm *FCMSender // nil bila kredensial Firebase tidak dikonfigurasi
}

// cloneData shallow-copies a payload so fan-out to multiple recipients doesn't
// share (and mutate) the same map inside createAndSend.
func cloneData(d map[string]any) map[string]any {
	n := make(map[string]any, len(d)+3)
	for k, v := range d {
		n[k] = v
	}
	return n
}

func NewNotificationService(db *sql.DB, hub *ws.Hub) *NotificationService {
	return &NotificationService{
		q:   repository.New(db),
		hub: hub,
		fcm: NewFCMSender(config.C.FirebaseCredFile),
	}
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

	// Push FCM (bila dikonfigurasi) — async agar tidak memblokir request.
	if s.fcm != nil {
		push := map[string]string{"type": notifType}
		if entityID != nil {
			push["bookingId"] = strconv.FormatInt(int64(*entityID), 10)
		}
		go s.sendPush(userID, title, message, push)
	}
}

// sendPush mengirim FCM ke semua device token milik user; token yang sudah
// mati (unregistered) dibersihkan dari DB.
func (s *NotificationService) sendPush(userID int32, title, body string, data map[string]string) {
	ctx := context.Background()
	tokens, err := s.q.ListDeviceTokensByUser(ctx, userID)
	if err != nil {
		log.Printf("FCM: gagal ambil token user %d: %v", userID, err)
		return
	}
	for _, t := range tokens {
		unregistered, err := s.fcm.Send(ctx, t, title, body, data)
		if err != nil {
			log.Printf("FCM: kirim ke user %d gagal: %v", userID, err)
			if unregistered {
				_ = s.q.DeleteDeviceToken(ctx, t)
			}
		}
	}
}

// SaveDeviceToken menyimpan/menyegarkan FCM token milik user login.
func (s *NotificationService) SaveDeviceToken(ctx context.Context, userID int32, token, platform string) error {
	if platform == "" {
		platform = "android"
	}
	return s.q.UpsertDeviceToken(ctx, userID, token, platform)
}

// RemoveDeviceToken menghapus FCM token (dipanggil saat logout).
func (s *NotificationService) RemoveDeviceToken(ctx context.Context, token string) error {
	return s.q.DeleteDeviceToken(ctx, token)
}

// Notify sends a fully-typed notification to a single user (persist + push).
// entityID is derived from data["bookingId"] when present.
func (s *NotificationService) Notify(userID int32, notifType, title, message string, data map[string]any) {
	if data == nil {
		data = map[string]any{}
	}
	var entityID *int32
	if bID, ok := data["bookingId"].(int32); ok {
		entityID = &bID
	}
	s.createAndSend(context.Background(), userID, title, message, notifType, entityID, data)
}

// NotifyAdmins fans a notification out to every active ADMIN user.
func (s *NotificationService) NotifyAdmins(notifType, title, message string, data map[string]any) {
	s.notifyRole("ADMIN", notifType, title, message, data)
}

// NotifyRoomKeepers fans a notification out to every active ROOM_KEEPER user.
func (s *NotificationService) NotifyRoomKeepers(notifType, title, message string, data map[string]any) {
	s.notifyRole("ROOM_KEEPER", notifType, title, message, data)
}

func (s *NotificationService) notifyRole(role, notifType, title, message string, data map[string]any) {
	ids, err := s.q.ListActiveUserIDsByRole(context.Background(), role)
	if err != nil {
		log.Printf("notifyRole(%s): %v", role, err)
		return
	}
	for _, id := range ids {
		s.Notify(id, notifType, title, message, cloneData(data))
	}
}

// UnreadCount returns the number of unread notifications for a user.
func (s *NotificationService) UnreadCount(ctx context.Context, userID int32) (int64, error) {
	return s.q.CountUnreadNotifications(ctx, userID)
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

// GetMyNotifications returns (rows, total, error) - dipisah (bukan map
// gabungan) supaya handler bisa lewat util.Paginated() dan hasilkan
// amplop {success,message,data,pagination} yang SAMA seperti semua list
// endpoint lain (bukan pagination bersarang di dalam data).
func (s *NotificationService) GetMyNotifications(ctx context.Context, userID int32, page, limit int) ([]map[string]any, int64, error) {
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
		return nil, 0, err
	}

	total, err := s.q.CountNotificationsByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	// rows datang langsung dari sqlc (repository.Notification) - kalau di-
	// serialize apa adanya, RelatedEntityID (sql.NullInt32) jadi objek Go
	// mentah ({"Int32":.., "Valid":..}) di JSON, bukan angka/null. Ratakan
	// manual, konsisten dengan pola nullInt32() di booking_service.go.
	data := make([]map[string]any, len(rows))
	for i, n := range rows {
		data[i] = map[string]any{
			"id":              n.ID,
			"title":           n.Title,
			"body":            n.Body,
			"type":            n.Type,
			"relatedEntityId": nullInt32(n.RelatedEntityID),
			"isRead":          n.IsRead,
			"createdAt":       n.CreatedAt,
		}
	}

	return data, total, nil
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
