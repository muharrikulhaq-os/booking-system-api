package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"booking-system-api/internal/repository"
)

// MockQuerier is a minimal in-memory stand-in for repository.ExtendedQuerier.
// It embeds the interface so unimplemented methods panic loudly if a test
// exercises a code path it didn't expect to hit, while overridden methods
// below drive the specific scenarios under test.
type MockQuerier struct {
	repository.ExtendedQuerier

	updatedResourceStat repository.ResourceStatus

	booking            repository.GetBookingByIDRow
	createdOvertime    *repository.CreateDriverOvertimeParams
	overtimeForBooking *repository.DriverOvertime
}

func (m *MockQuerier) UpdateResourceStatus(ctx context.Context, arg repository.UpdateResourceStatusParams) (repository.Resource, error) {
	m.updatedResourceStat = arg.Status
	return repository.Resource{ID: arg.ID, Status: arg.Status}, nil
}

func (m *MockQuerier) GetBookingByID(ctx context.Context, id int32) (repository.GetBookingByIDRow, error) {
	return m.booking, nil
}

func (m *MockQuerier) CompleteBooking(ctx context.Context, id int32) (repository.Booking, error) {
	return repository.Booking{ID: id, Status: repository.BookingStatusCOMPLETED}, nil
}

func (m *MockQuerier) CreateDriverOvertime(ctx context.Context, arg repository.CreateDriverOvertimeParams) (repository.DriverOvertime, error) {
	m.createdOvertime = &arg
	return repository.DriverOvertime{ID: 1, BookingId: arg.BookingId, DriverId: arg.DriverId, OvertimeMinutes: arg.OvertimeMinutes}, nil
}

func (m *MockQuerier) GetOvertimeByBooking(ctx context.Context, bookingID int32) (repository.DriverOvertime, error) {
	if m.overtimeForBooking != nil {
		return *m.overtimeForBooking, nil
	}
	return repository.DriverOvertime{}, sql.ErrNoRows
}

func (m *MockQuerier) CountActiveBookingsByDriver(ctx context.Context, driverID, excludeBookingID int32) (int64, error) {
	return 0, nil
}

func (m *MockQuerier) ReleaseDriver(ctx context.Context, driverID int32) error {
	return nil
}

func (m *MockQuerier) CreateAuditLog(ctx context.Context, arg repository.CreateAuditLogParams) (repository.AuditLog, error) {
	return repository.AuditLog{}, nil
}

// GetBookingMerges: none of these fixtures involve a merged booking, so
// Complete()'s merge-sync (syncMergedBookingStatus) is always a no-op here.
func (m *MockQuerier) GetBookingMerges(ctx context.Context, bookingID int32) ([]repository.BookingMergeInfoRow, error) {
	return nil, nil
}

// ─── Overtime (Non-SPD) on booking Complete ────────────────────────────────

func TestBookingComplete_NonSPDPastEndDate_RecordsOvertime(t *testing.T) {
	scheduledEnd := time.Now().Add(-90 * time.Minute) // booking should've ended 1h30m ago
	m := &MockQuerier{
		booking: repository.GetBookingByIDRow{
			ID: 42, Status: repository.BookingStatusONGOING,
			ResourceType:     repository.ResourceTypeVEHICLE,
			BookingType:      repository.BookingTypeNONSPD,
			AssignedDriverId: sql.NullInt32{Int32: 7, Valid: true},
			EndDate:          scheduledEnd,
		},
	}
	svc := &BookingService{q: m}

	if _, err := svc.Complete(context.Background(), 42, AuditActor{UserID: 1}, "ADMIN"); err != nil {
		t.Fatalf("Complete() returned error: %v", err)
	}

	if m.createdOvertime == nil {
		t.Fatal("expected an overtime record for a NON_SPD vehicle booking completed past its endDate")
	}
	if m.createdOvertime.DriverId != 7 {
		t.Errorf("expected overtime attributed to driver 7, got %d", m.createdOvertime.DriverId)
	}
	if m.createdOvertime.OvertimeMinutes < 89 || m.createdOvertime.OvertimeMinutes > 91 {
		t.Errorf("expected ~90 overtime minutes, got %d", m.createdOvertime.OvertimeMinutes)
	}
}

func TestBookingComplete_SPD_NoOvertimeRecorded(t *testing.T) {
	scheduledEnd := time.Now().Add(-90 * time.Minute)
	m := &MockQuerier{
		booking: repository.GetBookingByIDRow{
			ID: 43, Status: repository.BookingStatusONGOING,
			ResourceType:     repository.ResourceTypeVEHICLE,
			BookingType:      repository.BookingTypeSPD,
			AssignedDriverId: sql.NullInt32{Int32: 7, Valid: true},
			EndDate:          scheduledEnd,
		},
	}
	svc := &BookingService{q: m}

	if _, err := svc.Complete(context.Background(), 43, AuditActor{UserID: 1}, "ADMIN"); err != nil {
		t.Fatalf("Complete() returned error: %v", err)
	}

	if m.createdOvertime != nil {
		t.Fatal("SPD bookings must not accrue overtime, per business rule")
	}
}

func TestBookingComplete_NonSPDOnTime_NoOvertimeRecorded(t *testing.T) {
	scheduledEnd := time.Now().Add(30 * time.Minute) // completed before the scheduled end
	m := &MockQuerier{
		booking: repository.GetBookingByIDRow{
			ID: 44, Status: repository.BookingStatusONGOING,
			ResourceType:     repository.ResourceTypeVEHICLE,
			BookingType:      repository.BookingTypeNONSPD,
			AssignedDriverId: sql.NullInt32{Int32: 7, Valid: true},
			EndDate:          scheduledEnd,
		},
	}
	svc := &BookingService{q: m}

	if _, err := svc.Complete(context.Background(), 44, AuditActor{UserID: 1}, "ADMIN"); err != nil {
		t.Fatalf("Complete() returned error: %v", err)
	}

	if m.createdOvertime != nil {
		t.Fatal("did not expect overtime when the booking completed on/before schedule")
	}
}
