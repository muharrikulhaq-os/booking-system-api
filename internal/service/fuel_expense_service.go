package service

import (
	"context"
	"database/sql"
	"fmt"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type FuelExpenseService struct {
	q repository.ExtendedQuerier
}

func NewFuelExpenseService(db *sql.DB) *FuelExpenseService {
	return &FuelExpenseService{q: repository.New(db)}
}

type CreateFuelExpenseRequest struct {
	VehicleID      int32   `json:"vehicleId"      validate:"required"`
	FuelTypeID     int32   `json:"fuelTypeId"`
	BookingID      *int32  `json:"bookingId"`
	FuelGrade      string  `json:"fuelGrade"`
	Liter          float64 `json:"liter"`
	PricePerLiter  float64 `json:"pricePerLiter"`
	Kwh            float64 `json:"kwh"`
	PricePerKwh    float64 `json:"pricePerKwh"`
	OdometerBefore int32   `json:"odometerBefore"`
	OdometerAfter  int32   `json:"odometerAfter"`
	ProofPhotoUrl  string  `json:"proofPhotoUrl"`
	Note           string  `json:"note"`
}

func numericFromFloat(f float64) sql.NullString {
	return sql.NullString{String: fmt.Sprintf("%g", f), Valid: true}
}

func numericStr(f float64) string {
	return fmt.Sprintf("%g", f)
}

func serializeFuelExpenseRow(fe repository.ListFuelExpensesRow) map[string]any {
	var kwh, pricePerKwh float64
	if fe.FuelCategoryName == repository.FuelCategoryLISTRIK {
		if fe.Quantity.Valid {
			kwh = util.ParseStringToFloat64(fe.Quantity.String)
		}
		if fe.PricePerUnit.Valid {
			pricePerKwh = util.ParseStringToFloat64(fe.PricePerUnit.String)
		}
	}
	
	var liter, pricePerLiter float64
	if fe.FuelCategoryName != repository.FuelCategoryLISTRIK {
		if fe.Quantity.Valid {
			liter = util.ParseStringToFloat64(fe.Quantity.String)
		}
		if fe.PricePerUnit.Valid {
			pricePerLiter = util.ParseStringToFloat64(fe.PricePerUnit.String)
		}
	}

	totalCost := 0.0
	if fe.TotalCost.Valid {
		totalCost = util.ParseStringToFloat64(fe.TotalCost.String)
	}

	return map[string]any{
		"id":             fe.ID,
		"vehicleId":      fe.VehicleId,
		"vehicle": map[string]any{
			"id":          fe.VehicleId,
			"name":        fe.VehicleName,
			"plateNumber": fe.PlateNumber,
		},
		"fuelTypeId":     fe.FuelTypeId,
		"fuelType":       fe.FuelCategoryName,
		"fuelGrade":      fe.FuelGrade.String,
		"bookingId":      fe.BookingId.Int32,
		"driverId":       fe.DriverId.Int32,
		"driverName":     fe.DriverName.String,
		"recordedById":   fe.RecordedById,
		"odometerBefore": fe.OdometerBefore.Int32,
		"odometerAfter":  fe.OdometerAfter.Int32,
		"distanceKm":     fe.DistanceKm.Int32,
		"liter":          liter,
		"pricePerLiter":  pricePerLiter,
		"kwh":            kwh,
		"pricePerKwh":    pricePerKwh,
		"totalCost":      totalCost,
		"proofPhotoUrl":  fe.ProofPhotoUrl.String,
		"note":           fe.Note.String,
		"createdAt":      fe.CreatedAt,
	}
}

func (s *FuelExpenseService) List(ctx context.Context, page, limit int, driverID, vehicleID *int32, fuelType *string, bookingID *int32, actorID int32, role string) ([]map[string]any, int64, error) {
	if role == "DRIVER" {
		driver, err := s.q.GetDriverByUserID(ctx, actorID)
		if err == nil {
			driverID = &driver.ID
		}
	}

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
	if bookingID != nil {
		params.BookingID = sql.NullInt32{Int32: *bookingID, Valid: true}
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

func (s *FuelExpenseService) Create(ctx context.Context, req CreateFuelExpenseRequest, recordedByID int32, driverID *int32) (map[string]any, error) {
	var totalCost float64
	var quantity float64
	var pricePerUnit float64

	if req.Liter > 0 {
		quantity = req.Liter
		pricePerUnit = req.PricePerLiter
	} else if req.Kwh > 0 {
		quantity = req.Kwh
		pricePerUnit = req.PricePerKwh
	}

	// Jika frontend tidak mengirim harga (0), ambil dari default_price di master fuel_types
	if pricePerUnit <= 0 {
		fuelType, err := s.q.GetFuelTypeByID(ctx, req.FuelTypeID)
		if err == nil && fuelType.DefaultPrice.Valid {
			pricePerUnit = util.ParseStringToFloat64(fuelType.DefaultPrice.String)
		}
	}

	totalCost = quantity * pricePerUnit
	
	distanceKm := int32(0)
	if req.OdometerAfter > req.OdometerBefore {
		distanceKm = req.OdometerAfter - req.OdometerBefore
	}

	var bookingID sql.NullInt32
	if req.BookingID != nil {
		bookingID = sql.NullInt32{Int32: *req.BookingID, Valid: true}
	}
	// driverId mereferensi drivers.id (BUKAN users.id). Resolve dari user yang login.
	// Jika user bukan driver (mis. admin), driverId dibiarkan NULL.
	var dID sql.NullInt32
	if driver, derr := s.q.GetDriverByUserID(ctx, recordedByID); derr == nil {
		dID = sql.NullInt32{Int32: driver.ID, Valid: true}
	}

	fe, err := s.q.CreateFuelExpense(ctx, repository.CreateFuelExpenseParams{
		VehicleId:      req.VehicleID,
		FuelTypeId:     req.FuelTypeID, // In real app, we might infer this from FuelGrade or pass from FE
		BookingId:      bookingID,
		DriverId:       dID,
		RecordedById:   recordedByID,
		FuelGrade:      sql.NullString{String: req.FuelGrade, Valid: req.FuelGrade != ""},
		ProofPhotoUrl:  sql.NullString{String: req.ProofPhotoUrl, Valid: req.ProofPhotoUrl != ""},
		OdometerBefore: sql.NullInt32{Int32: req.OdometerBefore, Valid: true},
		OdometerAfter:  sql.NullInt32{Int32: req.OdometerAfter, Valid: true},
		DistanceKm:     sql.NullInt32{Int32: distanceKm, Valid: true},
		Quantity:       numericFromFloat(quantity),
		PricePerUnit:   numericFromFloat(pricePerUnit),
		TotalCost:      numericFromFloat(totalCost),
		Location:       "Uploaded via API",
		Note:           sql.NullString{String: req.Note, Valid: req.Note != ""},
	})
	if err != nil {
		return nil, err
	}

	if req.OdometerAfter > 0 {
		_, _ = s.q.UpdateVehicleOdometer(ctx, repository.UpdateVehicleOdometerParams{
			ID: req.VehicleID, CurrentOdometer: req.OdometerAfter,
		})
		checkAndTriggerAutoMaintenance(ctx, s.q, req.VehicleID, recordedByID)
	}

	return s.GetByID(ctx, fe.ID)
}

func (s *FuelExpenseService) Delete(ctx context.Context, id int32) error {
	if _, err := s.q.GetFuelExpenseByID(ctx, id); err != nil {
		return util.ErrNotFound
	}
	return s.q.DeleteFuelExpense(ctx, id)
}
