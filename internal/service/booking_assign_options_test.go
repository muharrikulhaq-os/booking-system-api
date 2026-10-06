package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"booking-system-api/internal/repository"
)

// assignOptionsQuerier: armada kecil dengan konflik yang bisa diatur per id.
type assignOptionsQuerier struct {
	*MockQuerier
	vehicles          []repository.ListVehiclesRow
	drivers           []repository.ListDriversRow
	busyVehicles      map[int32]bool // CheckVehicleConflict
	maintVehicles     map[int32]bool // CheckMaintenanceConflict
	spdVehicles       map[int32]bool
	busyDrivers       map[int32][]int32 // ListDriverConflictBookingIDs
	spdDrivers        map[int32]bool
	mergePartnerOfB   int32
	lastDriverListArg repository.ListDriversParams
}

func (m *assignOptionsQuerier) ListVehicles(ctx context.Context, arg repository.ListVehiclesParams) ([]repository.ListVehiclesRow, error) {
	return m.vehicles, nil
}
func (m *assignOptionsQuerier) ListDrivers(ctx context.Context, arg repository.ListDriversParams) ([]repository.ListDriversRow, error) {
	m.lastDriverListArg = arg
	return m.drivers, nil
}
func (m *assignOptionsQuerier) CheckVehicleConflict(ctx context.Context, arg repository.CheckVehicleConflictParams) (int64, error) {
	if m.busyVehicles[arg.AssignedVehicleId.Int32] {
		return 1, nil
	}
	return 0, nil
}
func (m *assignOptionsQuerier) CheckMaintenanceConflict(ctx context.Context, arg repository.CheckMaintenanceConflictParams) (int64, error) {
	if m.maintVehicles[arg.VehicleID] {
		return 1, nil
	}
	return 0, nil
}
func (m *assignOptionsQuerier) CheckVehicleSpdConflict(ctx context.Context, arg repository.CheckVehicleSpdConflictParams) (int64, error) {
	if m.spdVehicles[arg.VehicleID] {
		return 1, nil
	}
	return 0, nil
}
func (m *assignOptionsQuerier) ListDriverConflictBookingIDs(ctx context.Context, driverID int32, start, end time.Time, excludeID int32) ([]int32, error) {
	return m.busyDrivers[driverID], nil
}
func (m *assignOptionsQuerier) GetBookingMerges(ctx context.Context, bookingID int32) ([]repository.BookingMergeInfoRow, error) {
	if m.mergePartnerOfB == 0 {
		return nil, nil
	}
	return []repository.BookingMergeInfoRow{{OtherBookingID: m.mergePartnerOfB}}, nil
}
func (m *assignOptionsQuerier) CheckDriverSpdConflict(ctx context.Context, arg repository.CheckDriverSpdConflictParams) (int64, error) {
	if m.spdDrivers[arg.DriverID] {
		return 1, nil
	}
	return 0, nil
}

func TestAssignOptions_PairsAndAvailability(t *testing.T) {
	start := time.Now().Add(24 * time.Hour)
	q := &assignOptionsQuerier{
		MockQuerier: &MockQuerier{booking: repository.GetBookingByIDRow{
			ID: 50, Status: repository.BookingStatusAPPROVED,
			ResourceType: repository.ResourceTypeVEHICLE, BookingType: repository.BookingTypeNONSPD,
			StartDate: start, EndDate: start.Add(4 * time.Hour), PassengerCount: 6,
			AssignedDriverId: sql.NullInt32{Int32: 2, Valid: true},
		}},
		vehicles: []repository.ListVehiclesRow{
			{ID: 1, ResourceName: "Avanza", PlateNumber: "B 1", Capacity: 7, ResourceStatus: repository.ResourceStatusAVAILABLE,
				FixedDriverId: sql.NullInt32{Int32: 1, Valid: true}, FixedDriverName: sql.NullString{String: "Budi", Valid: true}},
			{ID: 2, ResourceName: "HR-V", PlateNumber: "B 2", Capacity: 5, ResourceStatus: repository.ResourceStatusINUSE},
			{ID: 3, ResourceName: "Innova", PlateNumber: "B 3", Capacity: 7, ResourceStatus: repository.ResourceStatusAVAILABLE},
			{ID: 4, ResourceName: "Xpander", PlateNumber: "B 4", Capacity: 7, ResourceStatus: repository.ResourceStatusINACTIVE},
			{ID: 5, ResourceName: "Gran Max", PlateNumber: "B 5", Capacity: 7, ResourceStatus: repository.ResourceStatusMAINTENANCE},
		},
		drivers: []repository.ListDriversRow{
			{ID: 1, UserName: "Budi", EmployeeId: "D1", IsActive: true, UserIsActive: true},
			{ID: 2, UserName: "Agus", EmployeeId: "D2", IsActive: true, UserIsActive: true},
			{ID: 3, UserName: "Rahmat", EmployeeId: "D3", IsActive: true, UserIsActive: true},
			{ID: 4, UserName: "Akun mati", EmployeeId: "D4", IsActive: true, UserIsActive: false},
			{ID: 5, UserName: "Joko", EmployeeId: "D5", IsActive: true, UserIsActive: true},
		},
		busyVehicles:  map[int32]bool{3: true},
		maintVehicles: map[int32]bool{5: true},
		busyDrivers:   map[int32][]int32{3: {77}, 2: {51}},
		spdDrivers:    map[int32]bool{5: true},
		// Booking 51 adalah pasangan merge booking ini → bukan bentrok untuk Agus.
		mergePartnerOfB: 51,
	}
	out, err := (&BookingService{q: q}).AssignOptions(context.Background(), 50)
	if err != nil {
		t.Fatalf("AssignOptions: %v", err)
	}
	if !q.lastDriverListArg.IsActive.Valid || !q.lastDriverListArg.IsActive.Bool {
		t.Error("daftar supir harus difilter isActive=true")
	}

	vs := map[int32]map[string]any{}
	for _, v := range out["vehicles"].([]map[string]any) {
		vs[v["id"].(int32)] = v
	}
	wantV := map[int32]any{1: nil, 2: nil, 3: "Dipakai booking lain di jadwal ini", 4: "Kendaraan nonaktif", 5: "Jadwal maintenance di tanggal ini"}
	for id, want := range wantV {
		if got := vs[id]["reason"]; got != want {
			t.Errorf("kendaraan %d reason = %v, want %v", id, got, want)
		}
		if got := vs[id]["available"].(bool); got != (want == nil) {
			t.Errorf("kendaraan %d available = %v", id, got)
		}
	}
	// IN_USE saat ini tapi bebas di jadwal booking → tetap tersedia; kapasitas kurang hanya peringatan.
	if vs[2]["warning"] != "Kapasitas 5 kursi, penumpang 6" {
		t.Errorf("warning kapasitas = %v", vs[2]["warning"])
	}
	if vs[1]["fixedDriver"] == nil {
		t.Error("Avanza harus membawa supir tetap Budi")
	}

	ds := map[int32]map[string]any{}
	for _, d := range out["drivers"].([]map[string]any) {
		ds[d["id"].(int32)] = d
	}
	if _, ok := ds[4]; ok {
		t.Error("supir dengan akun nonaktif tidak boleh muncul")
	}
	wantD := map[int32]any{1: nil, 2: nil, 3: "Bertugas di booking lain pada jadwal ini", 5: "Bertugas SPD di tanggal ini"}
	for id, want := range wantD {
		if got := ds[id]["reason"]; got != want {
			t.Errorf("supir %d reason = %v, want %v", id, got, want)
		}
	}
	if fv, _ := ds[1]["fixedVehicle"].(map[string]any); fv == nil || fv["plateNumber"] != "B 1" {
		t.Errorf("Budi harus membawa kendaraan tetap B 1, got %v", ds[1]["fixedVehicle"])
	}
	if ds[2]["isCurrent"] != true || ds[1]["isCurrent"] != false {
		t.Error("isCurrent harus menandai supir yang sedang ditugaskan")
	}
}

func TestAssignOptions_RejectsRoomAndFinishedBookings(t *testing.T) {
	svc := func(b repository.GetBookingByIDRow) *BookingService {
		return &BookingService{q: &assignOptionsQuerier{MockQuerier: &MockQuerier{booking: b}}}
	}
	if _, err := svc(repository.GetBookingByIDRow{ResourceType: repository.ResourceTypeROOM, Status: repository.BookingStatusPENDING}).
		AssignOptions(context.Background(), 1); err == nil {
		t.Error("booking ruangan harus ditolak")
	}
	if _, err := svc(repository.GetBookingByIDRow{ResourceType: repository.ResourceTypeVEHICLE, Status: repository.BookingStatusONGOING}).
		AssignOptions(context.Background(), 1); err == nil {
		t.Error("booking ONGOING harus ditolak")
	}
}
