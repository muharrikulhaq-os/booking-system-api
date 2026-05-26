package service

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
	BookingID      *int32  `json:"bookingId"`
	Liter          float64 `json:"liter"          validate:"required,gt=0"`
	PricePerLiter  float64 `json:"pricePerLiter"  validate:"required,gt=0"`
	OdometerBefore int32   `json:"odometerBefore" validate:"required"`
	OdometerAfter  int32   `json:"odometerAfter"  validate:"required"`
	Note           string  `json:"note"`
}

type CreateFuelExpenseListrikRequest struct {
	VehicleID     int32   `json:"vehicleId"     validate:"required"`
	BookingID     *int32  `json:"bookingId"`
	Kwh           float64 `json:"kwh"           validate:"required,gt=0"`
	PricePerKwh   float64 `json:"pricePerKwh"   validate:"required,gt=0"`
	BatteryBefore float64 `json:"batteryBefore" validate:"min=0,max=100"`
	BatteryAfter  float64 `json:"batteryAfter"  validate:"min=0,max=100"`
	Note          string  `json:"note"`
}

func numericFromFloat(f float64) sql.NullString {
	return sql.NullString{String: fmt.Sprintf("%g", f), Valid: true}
}

func numericStr(f float64) string {
	return fmt.Sprintf("%g", f)
}

func nullNumeric(f *float64) sql.NullString {
	if f == nil {
		return sql.NullString{}
	}
	return numericFromFloat(*f)
}

func serializeFuelExpenseRow(fe repository.ListFuelExpensesRow) map[string]any {
	return map[string]any{
		"id":          fe.ID,
		"driverId":    fe.DriverId,
		"driverName":  fe.DriverName,
		"vehicleId":   fe.VehicleId,
		"plateNumber": fe.PlateNumber,
		"vehicleName": fe.VehicleName,
		"bookingId":   nullInt32(fe.BookingId),
		"fuelType":    string(fe.FuelType),
		"totalAmount": fe.TotalAmount,
		"note":        nullStr(fe.Note),
		"createdAt":   fe.CreatedAt,
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
		params.FuelType = repository.NullFuelType{FuelType: repository.FuelType(*fuelType), Valid: true}
	}

	rows, err := s.q.ListFuelExpenses(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, _ := s.q.CountFuelExpenses(ctx, repository.CountFuelExpensesParams{
		DriverID: params.DriverID, VehicleID: params.VehicleID, FuelType: params.FuelType,
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
	return serializeFuelExpenseRow(repository.ListFuelExpensesRow{
		ID: fe.ID, DriverId: fe.DriverId, DriverName: fe.DriverName,
		VehicleId: fe.VehicleId, PlateNumber: fe.PlateNumber, VehicleName: fe.VehicleName,
		BookingId: fe.BookingId, FuelType: fe.FuelType, TotalAmount: fe.TotalAmount,
		Note: fe.Note, CreatedAt: fe.CreatedAt,
	}), nil
}

func (s *FuelExpenseService) CreateBBM(ctx context.Context, req CreateFuelExpenseBBMRequest, driverID int32) (map[string]any, error) {
	if req.OdometerAfter <= req.OdometerBefore {
		return nil, util.NewError(400, "odometerAfter must be greater than odometerBefore", util.ErrBadRequest)
	}
	total := req.Liter * req.PricePerLiter
	var bookingID sql.NullInt32
	if req.BookingID != nil {
		bookingID = sql.NullInt32{Int32: *req.BookingID, Valid: true}
	}

	fe, err := s.q.CreateFuelExpenseBBM(ctx, repository.CreateFuelExpenseBBMParams{
		DriverId:       driverID,
		VehicleId:      req.VehicleID,
		BookingId:      bookingID,
		Liter:          numericFromFloat(req.Liter),
		PricePerLiter:  numericFromFloat(req.PricePerLiter),
		OdometerBefore: sql.NullInt32{Int32: req.OdometerBefore, Valid: true},
		OdometerAfter:  sql.NullInt32{Int32: req.OdometerAfter, Valid: true},
		TotalAmount:    numericStr(total),
		Note:           sql.NullString{String: req.Note, Valid: req.Note != ""},
	})
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, fe.ID)
}

func (s *FuelExpenseService) CreateListrik(ctx context.Context, req CreateFuelExpenseListrikRequest, driverID int32) (map[string]any, error) {
	total := req.Kwh * req.PricePerKwh
	var bookingID sql.NullInt32
	if req.BookingID != nil {
		bookingID = sql.NullInt32{Int32: *req.BookingID, Valid: true}
	}

	fe, err := s.q.CreateFuelExpenseListrik(ctx, repository.CreateFuelExpenseListrikParams{
		DriverId:      driverID,
		VehicleId:     req.VehicleID,
		BookingId:     bookingID,
		Kwh:           numericFromFloat(req.Kwh),
		PricePerKwh:   numericFromFloat(req.PricePerKwh),
		BatteryBefore: numericFromFloat(req.BatteryBefore),
		BatteryAfter:  numericFromFloat(req.BatteryAfter),
		TotalAmount:   numericStr(total),
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
