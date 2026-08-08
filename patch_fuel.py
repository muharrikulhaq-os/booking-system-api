import os

fuel_service_code = """package service

import (
	"context"
	"database/sql"
	"fmt"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type FuelExpenseService struct {
	q *repository.Queries
}

func NewFuelExpenseService(db *sql.DB) *FuelExpenseService {
	return &FuelExpenseService{q: repository.New(db)}
}

type CreateFuelExpenseBBMRequest struct {
	VehicleID      int32   `json:"vehicleId"      validate:"required"`
	FuelTypeID     int32   `json:"fuelTypeId"     validate:"required"`
	BookingID      *int32  `json:"bookingId"`
	Quantity       float64 `json:"quantity"       validate:"required,gt=0"`
	PricePerUnit   float64 `json:"pricePerUnit"`
	Odometer       int32   `json:"odometer"       validate:"required"`
	Location       string  `json:"location"       validate:"required"`
	StationName    string  `json:"stationName"`
	Note           string  `json:"note"`
}

type CreateFuelExpenseListrikRequest struct {
	VehicleID     int32   `json:"vehicleId"     validate:"required"`
	FuelTypeID    int32   `json:"fuelTypeId"    validate:"required"`
	BookingID     *int32  `json:"bookingId"`
	Quantity      float64 `json:"quantity"      validate:"required,gt=0"`
	PricePerUnit  float64 `json:"pricePerUnit"`
	BatteryBefore float64 `json:"batteryBefore" validate:"min=0,max=100"`
	BatteryAfter  float64 `json:"batteryAfter"  validate:"min=0,max=100"`
	Location      string  `json:"location"      validate:"required"`
	StationName   string  `json:"stationName"`
	Note          string  `json:"note"`
}

func numericFromFloat(f float64) sql.NullString {
	return sql.NullString{String: fmt.Sprintf("%g", f), Valid: true}
}

func numericStr(f float64) string {
	return fmt.Sprintf("%g", f)
}

func serializeFuelExpenseRow(fe repository.ListFuelExpensesRow) map[string]any {
	return map[string]any{
		"id":           fe.ID,
		"vehicleId":    fe.VehicleId,
		"plateNumber":  fe.PlateNumber,
		"vehicleName":  fe.VehicleName,
		"fuelTypeId":   fe.FuelTypeId,
		"fuelCategory": fe.FuelCategoryName,
		"bookingId":    fe.BookingId.Int32,
		"driverId":     fe.DriverId.Int32,
		"driverName":   fe.DriverName.String,
		"recordedById": fe.RecordedById,
		"odometer":     fe.Odometer.Int32,
		"quantity":     fe.Quantity,
		"pricePerUnit": fe.PricePerUnit,
		"totalCost":    fe.TotalCost,
		"location":     fe.Location,
		"stationName":  fe.StationName.String,
		"note":         fe.Note.String,
		"createdAt":    fe.CreatedAt,
	}
}

func (s *FuelExpenseService) List(ctx context.Context, page, limit int, driverID, vehicleID *int32, fuelType *string) ([]map[string]any, int64, error) {
	params := repository.ListFuelExpensesParams{
		Limit:  int32(limit),
		Offset: int32((page - 1) * limit),
	}
	if driverID != nil {
		params.DriverID = sql.NullInt32{Int32: *driverID, Valid: true}
	}
	if vehicleID != nil {
		params.VehicleID = sql.NullInt32{Int32: *vehicleID, Valid: true}
	}
	if fuelType != nil {
		params.FuelCategory = repository.NullFuelCategory{FuelCategory: repository.FuelCategory(*fuelType), Valid: true}
	}

	rows, err := s.q.ListFuelExpenses(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, _ := s.q.CountFuelExpenses(ctx, repository.CountFuelExpensesParams{
		DriverID: params.DriverID, VehicleID: params.VehicleID, FuelCategory: params.FuelCategory,
	})
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeFuelExpenseRow(r)
	}
	return out, total, nil
}

func (s *FuelExpenseService) GetByID(ctx context.Context, id int32) (map[string]any, error) {
	fe, err := s.q.GetFuelExpenseByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	return serializeFuelExpenseRow(repository.ListFuelExpensesRow(fe)), nil
}

func (s *FuelExpenseService) CreateBBM(ctx context.Context, req CreateFuelExpenseBBMRequest, recordedByID int32, driverID *int32) (map[string]any, error) {
	if req.PricePerUnit == 0 {
		ms, err := s.q.GetMasterSettingByKey(ctx, "price_per_liter_bbm")
		if err == nil {
			req.PricePerUnit = util.ParseStringToFloat64(ms.Value)
		}
	}
	total := req.Quantity * req.PricePerUnit
	var bookingID sql.NullInt32
	if req.BookingID != nil {
		bookingID = sql.NullInt32{Int32: *req.BookingID, Valid: true}
	}
	var dID sql.NullInt32
	if driverID != nil {
		dID = sql.NullInt32{Int32: *driverID, Valid: true}
	}

	fe, err := s.q.CreateFuelExpense(ctx, repository.CreateFuelExpenseParams{
		VehicleId:    req.VehicleID,
		FuelTypeId:   req.FuelTypeID,
		BookingId:    bookingID,
		DriverId:     dID,
		RecordedById: recordedByID,
		Odometer:     sql.NullInt32{Int32: req.Odometer, Valid: true},
		Quantity:     numericStr(req.Quantity),
		PricePerUnit: numericStr(req.PricePerUnit),
		TotalCost:    numericStr(total),
		Location:     req.Location,
		StationName:  sql.NullString{String: req.StationName, Valid: req.StationName != ""},
		Note:         sql.NullString{String: req.Note, Valid: req.Note != ""},
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, fe.ID)
}

func (s *FuelExpenseService) CreateListrik(ctx context.Context, req CreateFuelExpenseListrikRequest, recordedByID int32, driverID *int32) (map[string]any, error) {
	if req.PricePerUnit == 0 {
		ms, err := s.q.GetMasterSettingByKey(ctx, "price_per_kwh_listrik")
		if err == nil {
			req.PricePerUnit = util.ParseStringToFloat64(ms.Value)
		}
	}
	total := req.Quantity * req.PricePerUnit
	var bookingID sql.NullInt32
	if req.BookingID != nil {
		bookingID = sql.NullInt32{Int32: *req.BookingID, Valid: true}
	}
	var dID sql.NullInt32
	if driverID != nil {
		dID = sql.NullInt32{Int32: *driverID, Valid: true}
	}

	fe, err := s.q.CreateFuelExpense(ctx, repository.CreateFuelExpenseParams{
		VehicleId:     req.VehicleID,
		FuelTypeId:    req.FuelTypeID,
		BookingId:     bookingID,
		DriverId:      dID,
		RecordedById:  recordedByID,
		Quantity:      numericStr(req.Quantity),
		PricePerUnit:  numericStr(req.PricePerUnit),
		TotalCost:     numericStr(total),
		BatteryBefore: numericFromFloat(req.BatteryBefore),
		BatteryAfter:  numericFromFloat(req.BatteryAfter),
		Location:      req.Location,
		StationName:   sql.NullString{String: req.StationName, Valid: req.StationName != ""},
		Note:          sql.NullString{String: req.Note, Valid: req.Note != ""},
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, fe.ID)
}

func (s *FuelExpenseService) Delete(ctx context.Context, id int32) error {
	if _, err := s.q.GetFuelExpenseByID(ctx, id); err != nil {
		return util.ErrNotFound
	}
	return s.q.DeleteFuelExpense(ctx, id)
}
"""

with open('internal/service/fuel_expense_service.go', 'w') as f:
    f.write(fuel_service_code)
print("Updated internal/service/fuel_expense_service.go")
