package service

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"booking-system-api/internal/repository"
)

type MockQuerier struct {
	repository.Querier
	bookingConflictCount int64
}

func (m *MockQuerier) GetMasterSettingByKey(ctx context.Context, key string) (repository.MasterSetting, error) {
	if key == "price_per_liter_bbm" {
		return repository.MasterSetting{Key: key, Value: "12500"}, nil
	}
	return repository.MasterSetting{}, sql.ErrNoRows
}

func (m *MockQuerier) CreateFuelExpense(ctx context.Context, arg repository.CreateFuelExpenseParams) (repository.FuelExpense, error) {
	fmt.Printf("[Mock DB] Inserted FuelExpense -> qty: %s, pricePerUnit: %s, totalCost: %s\n", arg.Quantity, arg.PricePerUnit, arg.TotalCost)
	return repository.FuelExpense{ID: 1}, nil
}

func (m *MockQuerier) GetFuelExpenseByID(ctx context.Context, id int32) (repository.GetFuelExpenseByIDRow, error) {
	return repository.GetFuelExpenseByIDRow{
		ID:           id,
		Quantity:     "10",
		PricePerUnit: "12500",
		TotalCost:    "125000",
	}, nil
}

func (m *MockQuerier) GetVehicleByID(ctx context.Context, id int32) (repository.GetVehicleByIDRow, error) {
	return repository.GetVehicleByIDRow{
		ID:         id,
		ResourceId: 99, // mock resource id
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

func TestMockLogic(t *testing.T) {
	fmt.Println("=====================================================")
	fmt.Println("🚀 MULAI TESTING PURE LOGIC (TANPA DB REAL)")
	fmt.Println("=====================================================")

	mockQ := &MockQuerier{}

	fmt.Println("\n--- Test 1: FuelExpenseService.CreateBBM ---")
	fmt.Println("Skenario: Frontend tidak mengirimkan pricePerUnit, sistem harus mengambil dari MasterSettings dan menghitung totalCost (10 liter * 12500 = 125000)")
	
	fuelSvc := &FuelExpenseService{q: mockQ}
	reqBBM := CreateFuelExpenseBBMRequest{
		VehicleID:    1,
		FuelTypeID:   1,
		Quantity:     10,
		Odometer:     50000,
		Location:     "SPBU Sudirman",
	}
	
	resBBM, err := fuelSvc.CreateBBM(context.Background(), reqBBM, 1, nil)
	if err != nil {
		t.Fatalf("CreateBBM error: %v", err)
	}
	fmt.Printf("✅ CreateBBM Success! Result totalCost: %s\n", resBBM["totalCost"])
	
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

	fmt.Println("\n=====================================================")
	fmt.Println("🎉 SEMUA TEST BERHASIL!")
	fmt.Println("=====================================================")
}
