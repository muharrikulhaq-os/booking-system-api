package service

import (
	"context"
	"database/sql"
	"time"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type GuestBookingService struct {
	q *repository.Queries
}

func NewGuestBookingService(db *sql.DB) *GuestBookingService {
	return &GuestBookingService{q: repository.New(db)}
}

type CreateGuestBookingRequest struct {
	GuestName      string    `json:"guestName"      validate:"required"`
	GuestEmail     string    `json:"guestEmail"     validate:"required,email"`
	GuestPhone     string    `json:"guestPhone"     validate:"required"`
	DepartmentName string    `json:"departmentName" validate:"required"`
	ResourceID     int32     `json:"resourceId"     validate:"required"`
	StartDate      time.Time `json:"startDate"      validate:"required"`
	EndDate        time.Time `json:"endDate"        validate:"required"`
	Purpose        string    `json:"purpose"        validate:"required"`
}

type RejectGuestBookingRequest struct {
	Note string `json:"note" validate:"required"`
}

func serializeGuestRow(g repository.ListGuestBookingsRow) map[string]any {
	return map[string]any{
		"id":             g.ID,
		"guestName":      g.GuestName,
		"guestEmail":     g.GuestEmail,
		"guestPhone":     g.GuestPhone,
		"departmentName": g.DepartmentName,
		"resource":       map[string]any{"id": g.ResourceId, "name": g.ResourceName, "type": string(g.ResourceType)},
		"startDate":      g.StartDate,
		"endDate":        g.EndDate,
		"purpose":        g.Purpose,
		"status":         string(g.Status),
		"accessToken":    g.AccessToken,
		"approvedBy":     nullStr(g.ApproverName),
		"approvedAt":     nullTime(g.ApprovedAt),
		"rejectionNote":  nullStr(g.RejectionNote),
		"returnedAt":     nullTime(g.ReturnedAt),
		"createdAt":      g.CreatedAt,
	}
}

func serializeGuestByID(g repository.GetGuestBookingByIDRow) map[string]any {
	return map[string]any{
		"id":             g.ID,
		"guestName":      g.GuestName,
		"guestEmail":     g.GuestEmail,
		"guestPhone":     g.GuestPhone,
		"departmentName": g.DepartmentName,
		"resource":       map[string]any{"id": g.ResourceId, "name": g.ResourceName, "type": string(g.ResourceType)},
		"startDate":      g.StartDate,
		"endDate":        g.EndDate,
		"purpose":        g.Purpose,
		"status":         string(g.Status),
		"accessToken":    g.AccessToken,
		"approvedAt":     nullTime(g.ApprovedAt),
		"rejectionNote":  nullStr(g.RejectionNote),
		"returnedAt":     nullTime(g.ReturnedAt),
		"createdAt":      g.CreatedAt,
	}
}

func serializeGuestByToken(g repository.GetGuestBookingByTokenRow) map[string]any {
	return map[string]any{
		"id":             g.ID,
		"guestName":      g.GuestName,
		"guestEmail":     g.GuestEmail,
		"guestPhone":     g.GuestPhone,
		"departmentName": g.DepartmentName,
		"resource":       map[string]any{"id": g.ResourceId, "name": g.ResourceName, "type": string(g.ResourceType)},
		"startDate":      g.StartDate,
		"endDate":        g.EndDate,
		"purpose":        g.Purpose,
		"status":         string(g.Status),
		"approvedAt":     nullTime(g.ApprovedAt),
		"rejectionNote":  nullStr(g.RejectionNote),
		"returnedAt":     nullTime(g.ReturnedAt),
		"createdAt":      g.CreatedAt,
	}
}

func (s *GuestBookingService) List(ctx context.Context, page, limit int, status *string) ([]map[string]any, int64, error) {
	params := repository.ListGuestBookingsParams{
		Limit:  int32(limit),
		Offset: int32((page - 1) * limit),
	}
	if status != nil {
		params.Status = repository.NullBookingStatus{
			BookingStatus: repository.BookingStatus(*status), Valid: true,
		}
	}
	rows, err := s.q.ListGuestBookings(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, _ := s.q.CountGuestBookings(ctx, params.Status)
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeGuestRow(r)
	}
	return out, total, nil
}

func (s *GuestBookingService) GetByToken(ctx context.Context, token string) (map[string]any, error) {
	g, err := s.q.GetGuestBookingByToken(ctx, token)
	if err != nil {
		return nil, util.ErrNotFound
	}
	return serializeGuestByToken(g), nil
}

func (s *GuestBookingService) GetByID(ctx context.Context, id int32) (map[string]any, error) {
	g, err := s.q.GetGuestBookingByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	return serializeGuestByID(g), nil
}

func (s *GuestBookingService) Create(ctx context.Context, req CreateGuestBookingRequest) (map[string]any, error) {
	if !req.EndDate.After(req.StartDate) {
		return nil, util.ErrInvalidDateRange
	}

	token := util.GenerateToken(32)
	g, err := s.q.CreateGuestBooking(ctx, repository.CreateGuestBookingParams{
		GuestName:      req.GuestName,
		GuestEmail:     req.GuestEmail,
		GuestPhone:     req.GuestPhone,
		DepartmentName: req.DepartmentName,
		ResourceId:     req.ResourceID,
		StartDate:      req.StartDate,
		EndDate:        req.EndDate,
		Purpose:        req.Purpose,
		AccessToken:    token,
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, g.ID)
}

func (s *GuestBookingService) CompleteByToken(ctx context.Context, token string) (map[string]any, error) {
	g, err := s.q.CompleteGuestBookingByToken(ctx, token)
	if err != nil {
		return nil, util.ErrNotFound
	}
	return s.GetByID(ctx, g.ID)
}

func (s *GuestBookingService) CancelByToken(ctx context.Context, token string) (map[string]any, error) {
	_, err := s.q.GetGuestBookingByToken(ctx, token)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if _, err = s.q.CancelGuestBookingByToken(ctx, token); err != nil {
		return nil, err
	}
	g, _ := s.q.GetGuestBookingByToken(ctx, token)
	return s.GetByID(ctx, g.ID)
}

func (s *GuestBookingService) Approve(ctx context.Context, id int32, approverID int32) (map[string]any, error) {
	g, err := s.q.GetGuestBookingByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if g.Status != repository.BookingStatusPENDING {
		return nil, util.ErrBookingNotPending
	}
	if _, err = s.q.ApproveGuestBooking(ctx, repository.ApproveGuestBookingParams{
		ID: id, ApprovedById: sql.NullInt32{Int32: approverID, Valid: true},
	}); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *GuestBookingService) Reject(ctx context.Context, id int32, req RejectGuestBookingRequest, approverID int32) (map[string]any, error) {
	g, err := s.q.GetGuestBookingByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if g.Status != repository.BookingStatusPENDING {
		return nil, util.ErrBookingNotPending
	}
	if _, err = s.q.RejectGuestBooking(ctx, repository.RejectGuestBookingParams{
		ID: id, ApprovedById: sql.NullInt32{Int32: approverID, Valid: true},
		RejectionNote: sql.NullString{String: req.Note, Valid: true},
	}); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *GuestBookingService) Start(ctx context.Context, id int32) (map[string]any, error) {
	g, err := s.q.GetGuestBookingByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if g.Status != repository.BookingStatusAPPROVED {
		return nil, util.NewError(409, "only APPROVED bookings can be started", util.ErrForbidden)
	}
	if time.Now().Before(g.StartDate) {
		return nil, util.NewError(400, "cannot start before scheduled time", util.ErrBadRequest)
	}
	if _, err = s.q.StartGuestBooking(ctx, id); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}
