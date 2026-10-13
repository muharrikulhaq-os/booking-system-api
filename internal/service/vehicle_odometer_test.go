package service

import (
	"context"
	"strings"
	"testing"

	"booking-system-api/internal/repository"
)

// odoQuerier: kendaraan tunggal + batas bawah odometer untuk CorrectOdometer.
type odoQuerier struct {
	*MockQuerier
	vehicle repository.GetVehicleByIDRow
	floor   int32
	written *int32
	audits  []string
}

func (m *odoQuerier) GetVehicleByID(ctx context.Context, id int32) (repository.GetVehicleByIDRow, error) {
	return m.vehicle, nil
}

func (m *odoQuerier) VehicleOdometerFloor(ctx context.Context, vehicleID int32) (int32, error) {
	return m.floor, nil
}

func (m *odoQuerier) SetVehicleOdometerExact(ctx context.Context, vehicleID, odometer int32) error {
	m.written = &odometer
	m.vehicle.CurrentOdometer = odometer
	return nil
}

func (m *odoQuerier) CreateAuditLog(ctx context.Context, arg repository.CreateAuditLogParams) (repository.AuditLog, error) {
	m.audits = append(m.audits, arg.Description.String)
	return repository.AuditLog{}, nil
}

func (m *odoQuerier) PromoteDueMaintenance(ctx context.Context) (int64, error) { return 0, nil }

func (m *odoQuerier) GetVehicleIDsWithActiveSpd(ctx context.Context) ([]int32, error) {
	return nil, nil
}

func newOdoFixture(current, floor int32) (*odoQuerier, *VehicleService) {
	m := &odoQuerier{
		MockQuerier: &MockQuerier{},
		vehicle: repository.GetVehicleByIDRow{
			ID: 3, ResourceName: "Avanza", PlateNumber: "B 1 AA", CurrentOdometer: current,
		},
		floor: floor,
	}
	return m, &VehicleService{q: m}
}

func TestCorrectOdometer_LowerButAboveFloor(t *testing.T) {
	// Salah input 125.000 → seharusnya 12.500; bacaan sah terakhir 12.400.
	m, svc := newOdoFixture(125000, 12400)
	if _, err := svc.CorrectOdometer(context.Background(), 3,
		CorrectOdometerRequest{Odometer: 12500, Reason: "  salah ketik nol  "}, AuditActor{UserID: 1}); err != nil {
		t.Fatalf("CorrectOdometer: %v", err)
	}
	if m.written == nil || *m.written != 12500 {
		t.Fatalf("odometer tersimpan = %v, mau 12500", m.written)
	}
	if len(m.audits) != 1 || !strings.Contains(m.audits[0], "125000 → 12500 km") ||
		!strings.Contains(m.audits[0], "Alasan: salah ketik nol") {
		t.Errorf("audit = %v", m.audits)
	}
}

func TestCorrectOdometer_Raise(t *testing.T) {
	m, svc := newOdoFixture(12000, 12000)
	if _, err := svc.CorrectOdometer(context.Background(), 3,
		CorrectOdometerRequest{Odometer: 15000, Reason: "odometer dicek fisik"}, AuditActor{}); err != nil {
		t.Fatalf("CorrectOdometer: %v", err)
	}
	if m.written == nil || *m.written != 15000 {
		t.Fatalf("odometer tersimpan = %v, mau 15000", m.written)
	}
}

func TestCorrectOdometer_Rejected(t *testing.T) {
	cases := []struct {
		name    string
		req     CorrectOdometerRequest
		wantMsg string
	}{
		{"di bawah bacaan sah", CorrectOdometerRequest{Odometer: 12399, Reason: "koreksi"}, "tidak boleh kurang dari 12400 km"},
		{"sama dengan sekarang", CorrectOdometerRequest{Odometer: 13000, Reason: "koreksi"}, "sama dengan catatan"},
		{"alasan kosong", CorrectOdometerRequest{Odometer: 14000, Reason: "   "}, "alasan koreksi wajib"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, svc := newOdoFixture(13000, 12400)
			_, err := svc.CorrectOdometer(context.Background(), 3, tc.req, AuditActor{})
			if err == nil || !strings.Contains(err.Error(), tc.wantMsg) {
				t.Fatalf("err = %v, mau memuat %q", err, tc.wantMsg)
			}
			if m.written != nil || len(m.audits) != 0 {
				t.Error("tidak boleh ada yang tersimpan saat ditolak")
			}
		})
	}
}

func (m *odoQuerier) ListVehicleOwnership(ctx context.Context, ids []int32) (map[int32]repository.VehicleOwnership, error) {
	return nil, nil
}
