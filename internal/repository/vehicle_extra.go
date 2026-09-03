package repository

import (
	"context"
	"database/sql"
)

// Hand-written (sqlc CLI unavailable in this environment) - mirrors what
// `sqlc generate` would produce from the matching entries in
// sql/query/vehicle.sql and sql/query/driver.sql.

// ─────────────────────────────────────────────────────────────────────────
// VEHICLE FIXED DRIVER
// Permanent, admin-managed 1:1 pairing (vehicles.fixedDriverId) - separate
// from driver_assignments, which only tracks who currently holds a vehicle
// through an active booking and clears itself when that booking completes.
// ─────────────────────────────────────────────────────────────────────────

type SetVehicleFixedDriverParams struct {
	VehicleID int32         `json:"vehicle_id"`
	DriverID  sql.NullInt32 `json:"driver_id"`
}

func (q *Queries) SetVehicleFixedDriver(ctx context.Context, arg SetVehicleFixedDriverParams) (Vehicle, error) {
	row := q.db.QueryRowContext(ctx, `
		UPDATE vehicles SET "fixedDriverId" = $2 WHERE id = $1
		RETURNING id, "resourceId", "plateNumber", brand, model, year, "currentOdometer", "categoryId", capacity, "photoUrl", energy_type, "maintenanceIntervalKm", "lastMaintenanceOdometer", "fixedDriverId"`,
		arg.VehicleID, arg.DriverID,
	)
	var i Vehicle
	err := row.Scan(
		&i.ID, &i.ResourceId, &i.PlateNumber, &i.Brand, &i.Model, &i.Year,
		&i.CurrentOdometer, &i.CategoryId, &i.Capacity, &i.PhotoUrl, &i.EnergyType,
		&i.MaintenanceIntervalKm, &i.LastMaintenanceOdometer, &i.FixedDriverId,
	)
	return i, err
}

// ClearVehicleFixedDriverByDriver releases whatever vehicle is currently
// fixed to driverID (no-op if none) - called before re-assigning that
// driver elsewhere, since the UNIQUE constraint only allows one vehicle
// per driver at a time.
func (q *Queries) ClearVehicleFixedDriverByDriver(ctx context.Context, driverID int32) error {
	_, err := q.db.ExecContext(ctx,
		`UPDATE vehicles SET "fixedDriverId" = NULL WHERE "fixedDriverId" = $1`, driverID)
	return err
}

type FixedDriverVehicleRow struct {
	DriverID    int32  `json:"driver_id"`
	VehicleID   int32  `json:"vehicle_id"`
	PlateNumber string `json:"plateNumber"`
}

// ListVehiclesWithFixedDriver returns every (driverId, vehicleId, plateNumber)
// pairing currently set - batched lookup for DriverService.List so it doesn't
// run one GetVehicleByFixedDriverID query per row.
func (q *Queries) ListVehiclesWithFixedDriver(ctx context.Context) ([]FixedDriverVehicleRow, error) {
	rows, err := q.db.QueryContext(ctx,
		`SELECT "fixedDriverId", id, "plateNumber" FROM vehicles WHERE "fixedDriverId" IS NOT NULL`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []FixedDriverVehicleRow
	for rows.Next() {
		var i FixedDriverVehicleRow
		if err := rows.Scan(&i.DriverID, &i.VehicleID, &i.PlateNumber); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (q *Queries) GetVehicleByFixedDriverID(ctx context.Context, driverID int32) (Vehicle, error) {
	row := q.db.QueryRowContext(ctx, `
		SELECT id, "resourceId", "plateNumber", brand, model, year, "currentOdometer", "categoryId", capacity, "photoUrl", energy_type, "maintenanceIntervalKm", "lastMaintenanceOdometer", "fixedDriverId"
		FROM vehicles WHERE "fixedDriverId" = $1 LIMIT 1`, driverID)
	var i Vehicle
	err := row.Scan(
		&i.ID, &i.ResourceId, &i.PlateNumber, &i.Brand, &i.Model, &i.Year,
		&i.CurrentOdometer, &i.CategoryId, &i.Capacity, &i.PhotoUrl, &i.EnergyType,
		&i.MaintenanceIntervalKm, &i.LastMaintenanceOdometer, &i.FixedDriverId,
	)
	return i, err
}

// GetDriverIDsWithActiveSpd mirrors GetVehicleIDsWithActiveSpd (booking_extra.go)
// on the driver side - used to badge "Digunakan SPD" in the driver picker at
// booking creation.
func (q *Queries) GetDriverIDsWithActiveSpd(ctx context.Context) ([]int32, error) {
	query := `
		SELECT DISTINCT "assignedDriverId" FROM bookings
		WHERE "bookingType" = 'SPD'
		  AND status IN ('APPROVED', 'ONGOING')
		  AND "assignedDriverId" IS NOT NULL
		  AND date_trunc('day', "startDate" AT TIME ZONE 'Asia/Jakarta') <= date_trunc('day', NOW() AT TIME ZONE 'Asia/Jakarta')
		  AND date_trunc('day', "endDate" AT TIME ZONE 'Asia/Jakarta')   >= date_trunc('day', NOW() AT TIME ZONE 'Asia/Jakarta')`
	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int32
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
