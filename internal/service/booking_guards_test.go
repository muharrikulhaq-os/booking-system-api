package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type guardFake struct {
	repository.ExtendedQuerier
	booking  repository.GetBookingByIDRow
	resource repository.Resource
}

func (f *guardFake) GetBookingByID(ctx context.Context, id int32) (repository.GetBookingByIDRow, error) {
	return f.booking, nil
}

func (f *guardFake) GetResourceByID(ctx context.Context, id int32) (repository.Resource, error) {
	return f.resource, nil
}

func TestApprove_RejectsSelfApproval(t *testing.T) {
	s := &BookingService{q: &guardFake{booking: repository.GetBookingByIDRow{
		ID: 1, UserId: 7, Status: repository.BookingStatusPENDING,
	}}}
	_, err := s.Approve(context.Background(), 1, ApproveBookingRequest{}, AuditActor{UserID: 7})
	if !errors.Is(err, util.ErrSelfApproval) {
		t.Fatalf("expected ErrSelfApproval, got %v", err)
	}
}

func TestCreate_RejectsPastStart(t *testing.T) {
	s := &BookingService{q: &guardFake{}}
	start := time.Now().Add(-bookingStartGrace - time.Minute)
	_, err := s.Create(context.Background(), CreateBookingRequest{
		ResourceID: 1, StartDate: start, EndDate: start.Add(time.Hour),
	}, AuditActor{UserID: 1})
	if !errors.Is(err, util.ErrBadRequest) {
		t.Fatalf("expected bad request for past start, got %v", err)
	}
}

func TestCreate_RejectsInactiveResource(t *testing.T) {
	s := &BookingService{q: &guardFake{resource: repository.Resource{ID: 1, Status: repository.ResourceStatusINACTIVE}}}
	start := time.Now().Add(time.Hour)
	_, err := s.Create(context.Background(), CreateBookingRequest{
		ResourceID: 1, StartDate: start, EndDate: start.Add(time.Hour),
	}, AuditActor{UserID: 1})
	if !errors.Is(err, util.ErrConflict) {
		t.Fatalf("expected conflict for INACTIVE resource, got %v", err)
	}
}

func TestInUseError(t *testing.T) {
	fk := fmt.Errorf("wrap: %w", &pgconn.PgError{Code: "23503"})
	if err := inUseError(fk, "dipakai"); !errors.Is(err, util.ErrConflict) {
		t.Fatalf("FK violation should map to conflict, got %v", err)
	}
	other := errors.New("boom")
	if err := inUseError(other, "dipakai"); err != other {
		t.Fatalf("other errors must pass through, got %v", err)
	}
	if err := inUseError(nil, "dipakai"); err != nil {
		t.Fatalf("nil must stay nil, got %v", err)
	}
}
