package repository

import (
	"context"
	"database/sql"
	"time"
)

// GetDriverHoldVehicleID mengembalikan kendaraan yang SEHARUSNYA dipegang
// supir saat ini, diturunkan dari booking aktifnya: yang sedang berjalan
// (ONGOING/OVERDUE) lebih dulu, lalu booking APPROVED yang paling awal
// dimulai. sql.ErrNoRows bila supir tidak punya booking aktif (= kosong).
//
// Ini sumber kebenaran untuk driver_assignments — lihat syncDriverHold di
// service; tabel penugasan tidak lagi diubah langsung per alur.
func (q *Queries) GetDriverHoldVehicleID(ctx context.Context, driverID int32) (int32, error) {
	var id int32
	err := q.db.QueryRowContext(ctx, `
		SELECT "assignedVehicleId" FROM bookings
		WHERE "assignedDriverId" = $1
		  AND status IN ('ONGOING', 'OVERDUE', 'APPROVED')
		  AND "assignedVehicleId" IS NOT NULL
		ORDER BY (status = 'APPROVED'), "startDate" ASC, id ASC
		LIMIT 1`, driverID).Scan(&id)
	return id, err
}

// ListDriverConflictBookingIDs mengembalikan booking aktif lain milik supir
// yang jadwalnya bertumpuk dengan [start, end). Dipakai untuk mencegah satu
// supir disetujui di dua perjalanan bersamaan; pemanggil mengecualikan
// pasangan merge (satu trip yang sama).
func (q *Queries) ListDriverConflictBookingIDs(ctx context.Context, driverID int32, start, end time.Time, excludeID int32) ([]int32, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT id FROM bookings
		WHERE "assignedDriverId" = $1
		  AND status IN ('APPROVED', 'ONGOING', 'OVERDUE')
		  AND "startDate" < $3 AND "endDate" > $2
		  AND id != $4`, driverID, start, end, excludeID)
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

// SubstituteBookingResource mengalihkan booking ke resource lain secara utuh:
// resource, kendaraan yang ditugaskan (untuk booking kendaraan), dan penanda
// asal (originalResourceId) — sama seperti AssignVehicleAndUpdateResource.
// Mengalihkan kembali ke resource asal menghapus penanda.
func (q *Queries) SubstituteBookingResource(ctx context.Context, bookingID, resourceID int32, vehicleID sql.NullInt32) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE bookings
		SET "originalResourceId" = CASE
		        WHEN "originalResourceId" = $2 THEN NULL
		        WHEN "originalResourceId" IS NULL THEN "resourceId"
		        ELSE "originalResourceId"
		    END,
		    "resourceId" = $2,
		    "assignedVehicleId" = COALESCE($3, "assignedVehicleId"),
		    "updatedAt" = NOW()
		WHERE id = $1`, bookingID, resourceID, vehicleID)
	return err
}

// ListDriverActiveBookingIDs: booking APPROVED/ONGOING/OVERDUE yang masih
// ditugaskan ke supir ini - supir tidak boleh dinonaktifkan selama ada.
func (q *Queries) ListDriverActiveBookingIDs(ctx context.Context, driverID int32) ([]int32, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT id FROM bookings
		WHERE "assignedDriverId" = $1 AND status IN ('APPROVED', 'ONGOING', 'OVERDUE')
		ORDER BY "startDate"`, driverID)
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
