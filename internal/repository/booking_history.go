package repository

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"
)

// Riwayat booking (admin) + data per bagian untuk ekspor Excel.
// Hand-written: query dinamis (filter gabungan), tidak lewat sqlc.

// BookingHistoryFilter: [From, To) = rentang jadwal mulai booking.
type BookingHistoryFilter struct {
	From, To     time.Time
	ResourceType string   // "" | VEHICLE | ROOM
	BookingType  string   // "" | SPD | NON_SPD (khusus kendaraan)
	Statuses     []string // kosong = semua (sudah divalidasi pemanggil)
	Search       string
	// Urutan (kolom tabel riwayat); kosong = jadwal terbaru dulu.
	SortBy, SortOrder string
}

// BookingHistorySortColumns: kunci sort dari web → ekspresi SQL (whitelist,
// lihat BuildOrderBy). "handler" = supir (kendaraan) / penjaga (ruangan).
var BookingHistorySortColumns = map[string]string{
	"id":           "b.id",
	"startDate":    `b."startDate"`,
	"resourceName": "r.name",
	"bookingType":  `b."bookingType"::text`,
	"requester":    "u.name",
	"handler":      "COALESCE(NULLIF(du.name, ''), rkn.names, '')",
	"status":       "b.status::text",
	"createdAt":    `b."createdAt"`,
}

const bookingHistoryFrom = `
	FROM bookings b
	JOIN users u ON u.id = b."userId"
	LEFT JOIN departments dept ON dept.id = u."departmentId"
	JOIN resources r ON r.id = b."resourceId"`

func (f BookingHistoryFilter) where() (string, []any) {
	conds := []string{`b."startDate" >= $1`, `b."startDate" < $2`}
	args := []any{f.From, f.To}
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "?", "$"+strconv.Itoa(len(args))))
	}
	if f.ResourceType != "" {
		add(`r.type::text = ?`, f.ResourceType)
	}
	if f.BookingType != "" {
		// SPD/Non-SPD hanya bermakna untuk kendaraan.
		add(`(r.type = 'VEHICLE' AND b."bookingType"::text = ?)`, f.BookingType)
	}
	if len(f.Statuses) > 0 {
		add(`b.status::text = ANY(?::text[])`, "{"+strings.Join(f.Statuses, ",")+"}")
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		add(`(r.name ILIKE '%' || ? || '%' OR u.name ILIKE '%' || ? || '%' OR b.id::text = ?)`, s)
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// BookingHistoryRow: satu booking lengkap (sheet "Booking" & daftar riwayat).
type BookingHistoryRow struct {
	ID               int32
	Status           string
	ResourceType     string
	BookingType      string
	ResourceName     string
	PlateNumber      string
	RoomLocation     string
	RequesterName    string
	EmployeeID       string
	Department       string
	RequesterPhone   string
	Purpose          string
	PassengerCount   int32
	PickupLocation   string
	Destination      string
	StartDate        time.Time
	EndDate          time.Time
	CreatedAt        time.Time
	ApprovedBy       string
	ApprovedAt       sql.NullTime
	DriverName       string
	OriginalResource string
	ReturnedAt       sql.NullTime
	OdometerStart    sql.NullInt32
	StartLocation    string
	StartPhotoURL    string
	OvertimeMinutes  sql.NullInt32
	TollBudget       sql.NullFloat64
	ParkingAllowance sql.NullFloat64
	RejectNote       string
	RoomKeepers      string
	StartedAt        sql.NullTime
	StartedBy        string
	CompletedAt      sql.NullTime
}

const bookingHistorySelect = `
	SELECT b.id, b.status::text, r.type::text, b."bookingType"::text, r.name,
	       COALESCE(av."plateNumber", rv."plateNumber", ''), COALESCE(rm.location, ''),
	       u.name, u."employeeId", COALESCE(dept.name, ''), COALESCE(u."phoneNumber", ''),
	       b.purpose, b."passengerCount", COALESCE(b."pickupLocation", ''), COALESCE(b.destination, ''),
	       b."startDate", b."endDate", b."createdAt",
	       COALESCE(ab.name, ''), b."approvedAt",
	       COALESCE(du.name, ''), COALESCE(orig.name, ''),
	       b."returnedAt", b."odometerStart", COALESCE(b."startLocation", ''), COALESCE(b."startPhotoUrl", ''),
	       (SELECT ot."overtimeMinutes" FROM driver_overtimes ot WHERE ot."bookingId" = b.id LIMIT 1),
	       b."tollBudget"::float8, b."parkingAllowance"::float8,
	       COALESCE((SELECT al.note FROM approval_logs al
	                 WHERE al."bookingId" = b.id AND al.action = 'REJECTED'
	                 ORDER BY al.id DESC LIMIT 1), ''),
	       COALESCE(rkn.names, ''),
	       st."createdAt", COALESCE(st.name, ''),
	       (SELECT a."createdAt" FROM audit_logs a
	        WHERE a."entityType" = 'Booking' AND a."entityId" = b.id AND a.action = 'COMPLETE'
	        ORDER BY a.id DESC LIMIT 1)` + bookingHistoryFrom + `
	LEFT JOIN vehicles rv ON rv."resourceId" = r.id
	LEFT JOIN rooms rm ON rm."resourceId" = r.id
	LEFT JOIN vehicles av ON av.id = b."assignedVehicleId"
	LEFT JOIN users ab ON ab.id = b."approvedById"
	LEFT JOIN drivers drv ON drv.id = b."assignedDriverId"
	LEFT JOIN users du ON du.id = drv."userId"
	LEFT JOIN resources orig ON orig.id = b."originalResourceId"
	LEFT JOIN LATERAL (
		SELECT a."createdAt", su.name FROM audit_logs a
		LEFT JOIN users su ON su.id = a."userId"
		WHERE a."entityType" = 'Booking' AND a."entityId" = b.id AND a.action = 'START'
		ORDER BY a.id LIMIT 1
	) st ON TRUE
	LEFT JOIN LATERAL (
		SELECT string_agg(ku.name, ', ' ORDER BY ku.name) AS names
		FROM room_keeper_rooms rkr
		JOIN room_keepers rk ON rk.id = rkr."roomKeeperId"
		JOIN users ku ON ku.id = rk."userId"
		WHERE rkr."roomId" = rm.id
	) rkn ON TRUE`

// ListBookingHistory: limit <= 0 = semua baris (ekspor).
func (q *Queries) ListBookingHistory(ctx context.Context, f BookingHistoryFilter, limit, offset int32) ([]BookingHistoryRow, error) {
	where, args := f.where()
	order := BuildOrderBy(f.SortBy, f.SortOrder, BookingHistorySortColumns, `b."startDate" DESC`)
	query := bookingHistorySelect + where + " ORDER BY " + order + ", b.id DESC"
	if limit > 0 {
		args = append(args, limit, offset)
		query += ` LIMIT $` + strconv.Itoa(len(args)-1) + ` OFFSET $` + strconv.Itoa(len(args))
	}
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []BookingHistoryRow{}
	for rows.Next() {
		var i BookingHistoryRow
		if err := rows.Scan(&i.ID, &i.Status, &i.ResourceType, &i.BookingType, &i.ResourceName,
			&i.PlateNumber, &i.RoomLocation,
			&i.RequesterName, &i.EmployeeID, &i.Department, &i.RequesterPhone,
			&i.Purpose, &i.PassengerCount, &i.PickupLocation, &i.Destination,
			&i.StartDate, &i.EndDate, &i.CreatedAt,
			&i.ApprovedBy, &i.ApprovedAt,
			&i.DriverName, &i.OriginalResource,
			&i.ReturnedAt, &i.OdometerStart, &i.StartLocation, &i.StartPhotoURL,
			&i.OvertimeMinutes, &i.TollBudget, &i.ParkingAllowance,
			&i.RejectNote, &i.RoomKeepers,
			&i.StartedAt, &i.StartedBy, &i.CompletedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (q *Queries) CountBookingHistory(ctx context.Context, f BookingHistoryFilter) (int64, error) {
	where, args := f.where()
	var n int64
	err := q.db.QueryRowContext(ctx, `SELECT COUNT(*)`+bookingHistoryFrom+where, args...).Scan(&n)
	return n, err
}

// ── Data per bagian (untuk ekspor), berdasarkan daftar booking ──────────────

// HistoryRecord: satu baris bagian ekspor - kolom sesuai urutan header di
// service. Nilai bertipe time.Time/sql.Null* diformat oleh penulis Excel.
type HistoryRecord struct {
	BookingID int32
	Values    []any
}

func (q *Queries) historyRecords(ctx context.Context, query string, ids []int32, ncol int) ([]HistoryRecord, error) {
	if len(ids) == 0 {
		return []HistoryRecord{}, nil
	}
	rows, err := q.db.QueryContext(ctx, query, intArrayLiteral(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []HistoryRecord{}
	for rows.Next() {
		var bid int32
		vals := make([]any, ncol)
		ptrs := make([]any, ncol+1)
		ptrs[0] = &bid
		for i := range vals {
			ptrs[i+1] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		items = append(items, HistoryRecord{BookingID: bid, Values: vals})
	}
	return items, rows.Err()
}

// HistoryPassengers: nama, jenis, NIP, departemen, HP, keterangan.
func (q *Queries) HistoryPassengers(ctx context.Context, ids []int32) ([]HistoryRecord, error) {
	return q.historyRecords(ctx, `
		SELECT bp."bookingId", COALESCE(pu.name, bp."guestName", ''),
		       CASE WHEN bp."userId" IS NULL THEN 'Tamu' ELSE 'Karyawan' END,
		       COALESCE(pu."employeeId", ''), COALESCE(pd.name, ''),
		       COALESCE(pu."phoneNumber", bp."guestPhone", ''), COALESCE(bp."guestNote", '')
		FROM booking_passengers bp
		LEFT JOIN users pu ON pu.id = bp."userId"
		LEFT JOIN departments pd ON pd.id = pu."departmentId"
		WHERE bp."bookingId" = ANY($1::int[])
		ORDER BY bp."bookingId", bp.id`, ids, 6)
}

// HistoryFuel: waktu, kendaraan, supir, BBM, jumlah, satuan, harga/satuan,
// total, odometer sebelum/sesudah, jarak, SPBU, sumber, alasan, lokasi,
// catatan, foto, dicatat oleh, dibatalkan, alasan batal.
func (q *Queries) HistoryFuel(ctx context.Context, ids []int32) ([]HistoryRecord, error) {
	return q.historyRecords(ctx, `
		SELECT fe."bookingId", fe."createdAt", COALESCE(v."plateNumber", ''), COALESCE(fdu.name, ''),
		       TRIM(COALESCE(ft.name, '') || ' ' || COALESCE(fe."fuelGrade", '')),
		       fe.quantity::float8, COALESCE(ft.unit::text, ''), fe."pricePerUnit"::float8, fe."totalCost"::float8,
		       fe."odometerBefore", fe."odometerAfter", fe."distanceKm",
		       COALESCE(fs.name, fe."stationName", ''), COALESCE(fe.source, ''), COALESCE(fe.reason, ''),
		       COALESCE(fe.location, ''), COALESCE(fe.note, ''), COALESCE(fe."proofPhotoUrl", ''),
		       COALESCE(ru.name, ''), fe."voidedAt", COALESCE(fe."voidReason", '')
		FROM fuel_expenses fe
		LEFT JOIN vehicles v ON v.id = fe."vehicleId"
		LEFT JOIN drivers fd ON fd.id = fe."driverId"
		LEFT JOIN users fdu ON fdu.id = fd."userId"
		LEFT JOIN fuel_types ft ON ft.id = fe."fuelTypeId"
		LEFT JOIN fuel_stations fs ON fs.id = fe."stationId"
		LEFT JOIN users ru ON ru.id = fe."recordedById"
		WHERE fe."bookingId" = ANY($1::int[])
		ORDER BY fe."bookingId", fe."createdAt"`, ids, 20)
}

// HistoryReturnReports: dikirim, oleh, odometer akhir, lokasi, catatan,
// jumlah foto.
func (q *Queries) HistoryReturnReports(ctx context.Context, ids []int32) ([]HistoryRecord, error) {
	return q.historyRecords(ctx, `
		SELECT rr."bookingId", rr."submittedAt", COALESCE(su.name, ''), rr.odometer,
		       COALESCE(rr.location, ''), COALESCE(rr.note, ''),
		       (SELECT COUNT(*) FROM attachments a
		        WHERE a."bookingId" = rr."bookingId" AND a.description = 'return_photo')
		FROM booking_return_reports rr
		LEFT JOIN users su ON su.id = rr."submittedById"
		WHERE rr."bookingId" = ANY($1::int[])
		ORDER BY rr."bookingId"`, ids, 6)
}

// HistoryTripExpenses: waktu, kategori, gerbang tol, masuk, keluar, nominal,
// cara bayar, catatan, foto, dicatat oleh, dibatalkan, alasan batal.
func (q *Queries) HistoryTripExpenses(ctx context.Context, ids []int32) ([]HistoryRecord, error) {
	return q.historyRecords(ctx, `
		SELECT te."bookingId", te."occurredAt", COALESCE(te.category, ''), COALESCE(te."tollPoint", ''),
		       COALESCE(te."entryGate", ''), COALESCE(te."exitGate", ''), te.amount::float8,
		       COALESCE(te."paymentMethod", ''), COALESCE(te.note, ''), COALESCE(te."proofPhotoUrl", ''),
		       COALESCE(ru.name, ''), te."voidedAt", COALESCE(te."voidReason", '')
		FROM trip_expenses te
		LEFT JOIN users ru ON ru.id = te."recordedById"
		WHERE te."bookingId" = ANY($1::int[])
		ORDER BY te."bookingId", te."occurredAt"`, ids, 12)
}

// HistoryRatings: untuk (supir/penjaga ruangan), nama, bintang, ulasan, waktu.
func (q *Queries) HistoryRatings(ctx context.Context, ids []int32) ([]HistoryRecord, error) {
	return q.historyRecords(ctx, `
		SELECT x."bookingId", x.target, x.name, x.rating, x.review, x."createdAt" FROM (
			SELECT dr."bookingId", 'Supir' AS target, COALESCE(du.name, '') AS name,
			       dr.rating::int AS rating, COALESCE(dr.review, '') AS review, dr."createdAt"
			FROM driver_ratings dr
			LEFT JOIN drivers d ON d.id = dr."driverId"
			LEFT JOIN users du ON du.id = d."userId"
			WHERE dr."bookingId" = ANY($1::int[])
			UNION ALL
			SELECT rr."bookingId", 'Penjaga ruangan', COALESCE(ku.name, 'Ruangan'),
			       rr.rating::int, COALESCE(rr.review, ''), rr."createdAt"
			FROM room_ratings rr
			LEFT JOIN room_keepers rk ON rk.id = rr."roomKeeperId"
			LEFT JOIN users ku ON ku.id = rk."userId"
			WHERE rr."bookingId" = ANY($1::int[])
		) x ORDER BY x."bookingId", x."createdAt"`, ids, 5)
}

// HistoryActivity: waktu, aksi, keterangan, oleh.
func (q *Queries) HistoryActivity(ctx context.Context, ids []int32) ([]HistoryRecord, error) {
	return q.historyRecords(ctx, `
		SELECT a."entityId", a."createdAt", a.action, COALESCE(a.description, ''), COALESCE(au.name, 'Sistem')
		FROM audit_logs a
		LEFT JOIN users au ON au.id = a."userId"
		WHERE a."entityType" = 'Booking' AND a."entityId" = ANY($1::int[])
		ORDER BY a."entityId", a.id`, ids, 4)
}

// HistoryAttachments: diunggah, oleh, nama file, keterangan, path. Termasuk
// foto laporan pengembalian (keterangan "return_photo").
func (q *Queries) HistoryAttachments(ctx context.Context, ids []int32) ([]HistoryRecord, error) {
	return q.historyRecords(ctx, `
		SELECT a."bookingId", a."createdAt", COALESCE(uu.name, ''), COALESCE(a."fileName", ''),
		       COALESCE(a.description, ''), a."filePath"
		FROM attachments a
		LEFT JOIN users uu ON uu.id = a."uploadedById"
		WHERE a."bookingId" = ANY($1::int[])
		ORDER BY a."bookingId", a.id`, ids, 5)
}
