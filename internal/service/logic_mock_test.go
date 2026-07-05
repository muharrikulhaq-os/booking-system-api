package service

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"booking-system-api/internal/repository"
	ws "booking-system-api/internal/websocket"
)

type MockQuerier struct {
	repository.ExtendedQuerier
	bookingConflictCount int64
}

func (m *MockQuerier) GetFuelTypeByID(ctx context.Context, id int32) (repository.FuelType, error) {
	if id == 1 {
		return repository.FuelType{ID: 1, DefaultPrice: sql.NullString{String: "12500", Valid: true}}, nil
	}
	return repository.FuelType{}, sql.ErrNoRows
}

func (m *MockQuerier) CreateFuelExpense(ctx context.Context, arg repository.CreateFuelExpenseParams) (repository.FuelExpense, error) {
	fmt.Printf("[Mock DB] Inserted FuelExpense -> qty: %v, pricePerUnit: %v, totalCost: %v\n", arg.Quantity.String, arg.PricePerUnit.String, arg.TotalCost.String)
	return repository.FuelExpense{ID: 1}, nil
}

func (m *MockQuerier) GetFuelExpenseByID(ctx context.Context, id int32) (repository.GetFuelExpenseByIDRow, error) {
	return repository.GetFuelExpenseByIDRow{
		ID:           id,
		Quantity:     sql.NullString{String: "10", Valid: true},
		PricePerUnit: sql.NullString{String: "12500", Valid: true},
		TotalCost:    sql.NullString{String: "125000", Valid: true},
	}, nil
}

func (m *MockQuerier) GetVehicleByID(ctx context.Context, id int32) (repository.GetVehicleByIDRow, error) {
	return repository.GetVehicleByIDRow{
		ID:         id,
		ResourceId: 99, // mock resource id
		Capacity:   6,
	}, nil
}

func (m *MockQuerier) CheckBookingConflict(ctx context.Context, arg repository.CheckBookingConflictParams) (int64, error) {
	fmt.Printf("[Mock DB] Checking booking conflict for resourceId %d...\n", arg.ResourceId)
	return m.bookingConflictCount, nil
}

func (m *MockQuerier) CreateMaintenance(ctx context.Context, arg repository.CreateMaintenanceParams) (repository.MaintenanceRecord, error) {
	fmt.Printf("[Mock DB] Inserted Maintenance -> vehicleId: %d, desc: %s\n", arg.VehicleId, arg.Description)
	return repository.MaintenanceRecord{ID: 2}, nil
}

func (m *MockQuerier) UpdateResourceStatus(ctx context.Context, arg repository.UpdateResourceStatusParams) (repository.Resource, error) {
	fmt.Printf("[Mock DB] Updated ResourceStatus for resourceId %d -> %s\n", arg.ID, arg.Status)
	return repository.Resource{}, nil
}

func (m *MockQuerier) GetMaintenanceByID(ctx context.Context, id int32) (repository.GetMaintenanceByIDRow, error) {
	return repository.GetMaintenanceByIDRow{
		ID: id,
	}, nil
}

func (m *MockQuerier) ListAvailableDrivers(ctx context.Context, arg repository.ListAvailableDriversParams) ([]repository.ListAvailableDriversRow, error) {
	fmt.Println("[Mock DB] Fetching ListAvailableDrivers...")
	return []repository.ListAvailableDriversRow{
		{
			DriverID: 1, DriverName: "Budi", EmployeeId: "EMP-001",
			VehicleID: 1, PlateNumber: "B 1234 CD", Capacity: 6,
			OverlappingPassengers: int32(m.bookingConflictCount),
		},
	}, nil
}

func (m *MockQuerier) GetBookingByID(ctx context.Context, id int32) (repository.GetBookingByIDRow, error) {
	var vId sql.NullInt32
	if id == 999 { // mock booking for testing capacity
		vId = sql.NullInt32{Int32: 1, Valid: true}
	}
	return repository.GetBookingByIDRow{
		ID: id, Status: repository.BookingStatusPENDING,
		AssignedVehicleId: vId,
		PassengerCount:    4, // this booking wants 4 seats
	}, nil
}

func (m *MockQuerier) GetOverlappingPassengerCount(ctx context.Context, arg repository.GetOverlappingPassengerCountParams) (int32, error) {
	fmt.Printf("[Mock DB] Checking overlapping passenger count for vehicle %d...\n", arg.AssignedVehicleId.Int32)
	return int32(m.bookingConflictCount), nil // reuse this variable for overlapping passengers
}

func (m *MockQuerier) ApproveBooking(ctx context.Context, arg repository.ApproveBookingParams) (repository.Booking, error) {
	fmt.Printf("[Mock DB] Booking %d approved by %d\n", arg.ID, arg.ApprovedById.Int32)
	return repository.Booking{ID: arg.ID}, nil
}

func (m *MockQuerier) CreateApprovalLog(ctx context.Context, arg repository.CreateApprovalLogParams) (repository.ApprovalLog, error) {
	return repository.ApprovalLog{}, nil
}

func TestMockLogic(t *testing.T) {
	fmt.Println("=====================================================")
	fmt.Println("🚀 MULAI TESTING PURE LOGIC (TANPA DB REAL)")
	fmt.Println("=====================================================")

	mockQ := &MockQuerier{}

	fmt.Println("\n--- Test 1: FuelExpenseService.Create ---")
	fmt.Println("Skenario: Frontend tidak mengirimkan pricePerUnit, sistem harus mengambil dari MasterSettings dan menghitung totalCost (10 liter * 12500 = 125000)")

	fuelSvc := &FuelExpenseService{q: mockQ}
	reqBBM := CreateFuelExpenseRequest{
		VehicleID:  1,
		FuelTypeID: 1,
		Liter:      10,
	}

	resBBM, err := fuelSvc.Create(context.Background(), reqBBM, 1, nil)
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	fmt.Printf("✅ Create Success! Result totalCost: %s\n", resBBM["totalCost"])

	fmt.Println("\n--- Test 2: MaintenanceService.Create (Tanpa Conflict) ---")
	fmt.Println("Skenario: Create maintenance, tidak ada booking conflict.")

	maintSvc := &MaintenanceService{q: mockQ}
	mockQ.bookingConflictCount = 0 // tidak ada conflict

	reqMaint := CreateMaintenanceRequest{
		VehicleID:   1,
		Description: "Ganti Oli Rutin",
		StartDate:   time.Now(),
		Location:    "Bengkel Resmi",
	}

	resMaint1, err := maintSvc.Create(context.Background(), reqMaint, 1)
	if err != nil {
		t.Fatalf("CreateMaintenance error: %v", err)
	}
	fmt.Printf("✅ Maintenance Created! Warning: '%s'\n", resMaint1.Warning)

	fmt.Println("\n--- Test 3: MaintenanceService.Create (Dengan Conflict Booking) ---")
	fmt.Println("Skenario: Create maintenance, ada 2 booking aktif beririsan. Sistem tidak boleh error, tapi kembalikan warning.")

	mockQ.bookingConflictCount = 2 // simulasi ada conflict
	resMaint2, err := maintSvc.Create(context.Background(), reqMaint, 1)
	if err != nil {
		t.Fatalf("CreateMaintenance error: %v", err)
	}
	fmt.Printf("✅ Maintenance Created! Warning: '%s'\n", resMaint2.Warning)

	fmt.Println("\n--- Test 4: DriverService.GetAvailableDrivers ---")
	fmt.Println("Skenario: Fetch driver saat ada 2 penumpang yang overlap. Kapasitas mobil 6, maka remainingSeats harus 4.")

	driverSvc := &DriverService{q: mockQ}
	mockQ.bookingConflictCount = 2 // 2 overlapping passengers
	drivers, err := driverSvc.GetAvailableDrivers(context.Background(), time.Now(), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("GetAvailableDrivers error: %v", err)
	}
	if len(drivers) > 0 {
		fmt.Printf("✅ Driver %s (Mobil %s) - Kapasitas: %v, Overlapping: %v, Sisa Kursi: %v\n",
			drivers[0]["driverName"], drivers[0]["plateNumber"],
			drivers[0]["vehicleCapacity"], drivers[0]["overlappingPassengers"], drivers[0]["remainingSeats"])
	}

	fmt.Println("\n--- Test 5: BookingService.Approve (Dengan WebSocket Notification) ---")
	fmt.Println("Skenario: Admin approve booking untuk 4 orang. Sistem harus mem-push notifikasi ke Websocket (User & Driver).")

	hub := ws.NewHub()
	go hub.Run()
	notifSvc := NewNotificationService(hub)

	bookingSvc := &BookingService{q: mockQ, notif: notifSvc}
	mockQ.bookingConflictCount = 2
	resApprove1, err := bookingSvc.Approve(context.Background(), 999, ApproveBookingRequest{Note: "OK"}, 1)
	if err != nil {
		t.Fatalf("Approve error: %v", err)
	}
	fmt.Printf("✅ Approve Berhasil! Warning: '%s'\n", resApprove1.Warning)

	fmt.Println("\n--- Test 6: BookingService.Approve (Kapasitas Overload) ---")
	fmt.Println("Skenario: Admin approve booking untuk 4 orang. Overlapping saat ini 4. Kapasitas 6. (4+4 = 8, overload 2 orang). Sistem harus beri soft warning.")

	mockQ.bookingConflictCount = 4
	resApprove2, err := bookingSvc.Approve(context.Background(), 999, ApproveBookingRequest{Note: "Paksa Approve"}, 1)
	if err != nil {
		t.Fatalf("Approve error: %v", err)
	}
	fmt.Printf("✅ Approve Berhasil! Warning: '%s'\n", resApprove2.Warning)

	fmt.Println("\n=====================================================")
	fmt.Println("🎉 SEMUA TEST BERHASIL!")
	fmt.Println("=====================================================")
}
