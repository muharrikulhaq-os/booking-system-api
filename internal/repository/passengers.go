package repository

import (
	"context"
	"database/sql"
	"time"
)

// Karyawan yang ikut booking — docs/RANCANGAN_PARKIR_PENUMPANG.md §3.
// Booking tetap satu (atas nama pemesan); tabel ini hanya daftar yang ikut.

type BookingPassenger struct {
	BookingID   int32  `json:"bookingId"`
	UserID      int32  `json:"userId"`
	Name        string `json:"name"`
	EmployeeID  string `json:"employeeId"`
	Department  string `json:"department"`
	PhoneNumber string `json:"phoneNumber"`
}

// ListBookingPassengers: penumpang untuk beberapa booking sekaligus (urut nama).
func (q *Queries) ListBookingPassengers(ctx context.Context, bookingIDs []int32) ([]BookingPassenger, error) {
	if len(bookingIDs) == 0 {
		return nil, nil
	}
	rows, err := q.db.QueryContext(ctx, `
		SELECT bp."bookingId", u.id, u.name, u."employeeId", dep.name, `+userPhoneExpr+`
		FROM booking_passengers bp
		JOIN users u ON u.id = bp."userId"
		JOIN departments dep ON dep.id = u."departmentId"`+userPhoneJoins+`
		WHERE bp."bookingId" = ANY($1::int[])
		ORDER BY bp."bookingId", u.name`, intArrayLiteral(bookingIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BookingPassenger
	for rows.Next() {
		var p BookingPassenger
		if err := rows.Scan(&p.BookingID, &p.UserID, &p.Name, &p.EmployeeID, &p.Department, &p.PhoneNumber); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ReplaceBookingPassengers mengganti daftar penumpang booking dan
// mengembalikan user yang BARU ditambahkan (untuk notifikasi).
func (q *Queries) ReplaceBookingPassengers(ctx context.Context, bookingID int32, userIDs []int32, addedByID int32) ([]int32, error) {
	if _, err := q.db.ExecContext(ctx, `
		DELETE FROM booking_passengers
		WHERE "bookingId" = $1 AND NOT ("userId" = ANY($2::int[]))`,
		bookingID, intArrayLiteral(userIDs)); err != nil {
		return nil, err
	}
	if len(userIDs) == 0 {
		return nil, nil
	}
	rows, err := q.db.QueryContext(ctx, `
		INSERT INTO booking_passengers ("bookingId", "userId", "addedById")
		SELECT $1, u, $3 FROM unnest($2::int[]) AS u
		ON CONFLICT ("bookingId", "userId") DO NOTHING
		RETURNING "userId"`, bookingID, intArrayLiteral(userIDs), addedByID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var added []int32
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		added = append(added, id)
	}
	return added, rows.Err()
}

func (q *Queries) IsBookingPassenger(ctx context.Context, bookingID, userID int32) (bool, error) {
	var ok bool
	err := q.db.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM booking_passengers WHERE "bookingId" = $1 AND "userId" = $2)`,
		bookingID, userID).Scan(&ok)
	return ok, err
}

// ListPassengerUserIDs: penumpang satu booking (untuk notifikasi).
func (q *Queries) ListPassengerUserIDs(ctx context.Context, bookingID int32) ([]int32, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT "userId" FROM booking_passengers WHERE "bookingId" = $1`, bookingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int32
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// PassengerTrip: booking yang diikuti seorang karyawan (bukan pemesannya).
type PassengerTrip struct {
	ID           int32          `json:"id"`
	Status       string         `json:"status"`
	Purpose      string         `json:"purpose"`
	StartDate    time.Time      `json:"startDate"`
	EndDate      time.Time      `json:"endDate"`
	ResourceName string         `json:"resourceName"`
	PlateNumber  sql.NullString `json:"plateNumber"`
	OwnerName    string         `json:"ownerName"`
	DriverName   sql.NullString `json:"driverName"`
}

// ListPassengerTrips: perjalanan yang akan datang / sedang berjalan yang
// diikuti user ini (PENDING/APPROVED/ONGOING/OVERDUE), urut waktu mulai.
func (q *Queries) ListPassengerTrips(ctx context.Context, userID int32, limit int32) ([]PassengerTrip, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT b.id, b.status::text, b.purpose, b."startDate", b."endDate", r.name, v."plateNumber",
		       ou.name, du.name
		FROM booking_passengers bp
		JOIN bookings b ON b.id = bp."bookingId"
		JOIN resources r ON r.id = b."resourceId"
		JOIN users ou ON ou.id = b."userId"
		LEFT JOIN vehicles v ON v.id = b."assignedVehicleId"
		LEFT JOIN drivers d ON d.id = b."assignedDriverId"
		LEFT JOIN users du ON du.id = d."userId"
		WHERE bp."userId" = $1 AND b."userId" <> $1
		  AND b.status IN ('PENDING', 'APPROVED', 'ONGOING', 'OVERDUE')
		ORDER BY b."startDate", b.id
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PassengerTrip
	for rows.Next() {
		var t PassengerTrip
		if err := rows.Scan(&t.ID, &t.Status, &t.Purpose, &t.StartDate, &t.EndDate, &t.ResourceName,
			&t.PlateNumber, &t.OwnerName, &t.DriverName); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// UserLookup: hasil pencarian karyawan untuk dipilih sebagai penumpang
// (tanpa nomor telepon — daftar pencarian tidak membocorkan nomor).
type UserLookup struct {
	ID         int32  `json:"id"`
	Name       string `json:"name"`
	EmployeeID string `json:"employeeId"`
	Department string `json:"department"`
}

// LookupActiveUsers: karyawan & admin aktif (bukan supir/penjaga ruangan),
// cocok nama / ID karyawan. ids (opsional) membatasi ke id tertentu.
func (q *Queries) LookupActiveUsers(ctx context.Context, search string, ids []int32, limit int32) ([]UserLookup, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT u.id, u.name, u."employeeId", dep.name
		FROM users u
		JOIN roles r ON r.id = u."roleId"
		JOIN departments dep ON dep.id = u."departmentId"
		WHERE u."isActive" = TRUE AND r.name IN ('EMPLOYEE', 'ADMIN')
		  AND ($1 = '' OR u.name ILIKE '%' || $1 || '%' OR u."employeeId" ILIKE '%' || $1 || '%')
		  AND (cardinality($2::int[]) = 0 OR u.id = ANY($2::int[]))
		ORDER BY u.name
		LIMIT $3`, search, intArrayLiteral(ids), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UserLookup
	for rows.Next() {
		var u UserLookup
		if err := rows.Scan(&u.ID, &u.Name, &u.EmployeeID, &u.Department); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (q *Queries) SetBookingPassengerCount(ctx context.Context, bookingID, count int32) error {
	_, err := q.db.ExecContext(ctx, `UPDATE bookings SET "passengerCount" = $2, "updatedAt" = NOW() WHERE id = $1`, bookingID, count)
	return err
}
