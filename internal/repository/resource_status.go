package repository

import (
	"context"
	"time"
)

// activeMaintenanceSQL: maintenance kendaraan yang SEDANG berlangsung — belum
// selesai, sudah dimulai, dan belum lewat tanggal selesainya (endDate NULL =
// terbuka). Maintenance terjadwal di masa depan belum mengunci kendaraan;
// tanggalnya diblokir lewat CheckMaintenanceConflict.
const activeMaintenanceSQL = `
	SELECT 1 FROM vehicles v
	JOIN maintenance_records m ON m."vehicleId" = v.id
	WHERE v."resourceId" = r.id
	  AND m.status <> 'completed'
	  AND m."startDate" <= NOW()
	  AND (m."endDate" IS NULL OR m."endDate" > NOW())`

const activeTripSQL = `
	SELECT 1 FROM bookings b
	WHERE b."resourceId" = r.id AND b.status IN ('ONGOING', 'OVERDUE')`

// ResourceStatusFacts adalah bahan menentukan status resource yang benar.
type ResourceStatusFacts struct {
	Status            ResourceStatus
	HasActiveTrip     bool
	HasActiveMaintain bool
}

func (q *Queries) GetResourceStatusFacts(ctx context.Context, resourceID int32) (ResourceStatusFacts, error) {
	var f ResourceStatusFacts
	err := q.db.QueryRowContext(ctx, `
		SELECT r.status,
		       EXISTS (`+activeTripSQL+`),
		       EXISTS (`+activeMaintenanceSQL+`)
		FROM resources r WHERE r.id = $1`, resourceID).Scan(&f.Status, &f.HasActiveTrip, &f.HasActiveMaintain)
	return f, err
}

// PromoteDueMaintenance mengubah kendaraan AVAILABLE yang maintenance-nya
// SUDAH tiba waktunya menjadi MAINTENANCE (tidak ada penjadwal; dipanggil
// saat daftar booking/kendaraan dibuka). Kendaraan yang sedang dipakai trip
// dibiarkan IN_USE — akan jadi MAINTENANCE saat trip selesai.
func (q *Queries) PromoteDueMaintenance(ctx context.Context) (int64, error) {
	res, err := q.db.ExecContext(ctx, `
		UPDATE resources r SET status = 'MAINTENANCE', "updatedAt" = NOW()
		WHERE r.status = 'AVAILABLE'
		  AND EXISTS (`+activeMaintenanceSQL+`)
		  AND NOT EXISTS (`+activeTripSQL+`)`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// CountActiveResourceOverlap menghitung booking APPROVED/ONGOING/OVERDUE lain
// di resource yang sama yang jamnya bertumpuk dengan [start, end). Dipakai
// untuk ruangan: satu ruangan tidak bisa dipakai dua rapat sekaligus.
func (q *Queries) CountActiveResourceOverlap(ctx context.Context, resourceID int32, start, end time.Time, excludeID int32) (int64, error) {
	var n int64
	err := q.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM bookings
		WHERE "resourceId" = $1
		  AND status IN ('APPROVED', 'ONGOING', 'OVERDUE')
		  AND "startDate" < $3 AND "endDate" > $2
		  AND id != $4`, resourceID, start, end, excludeID).Scan(&n)
	return n, err
}
