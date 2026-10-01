package repository

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// Vendor = bengkel rekanan (WORKSHOP), pemilik kendaraan sewa (OWNER), atau
// keduanya (BOTH). Lihat docs/RANCANGAN_MAINTENANCE_VENDOR.md §3.1.
type Vendor struct {
	ID        int32          `json:"id"`
	Name      string         `json:"name"`
	Type      string         `json:"type"`
	Address   sql.NullString `json:"address"`
	PicName   sql.NullString `json:"picName"`
	Phone     sql.NullString `json:"phone"`
	Email     sql.NullString `json:"email"`
	Note      sql.NullString `json:"note"`
	IsActive  bool           `json:"isActive"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	// Jumlah pemakaian (kendaraan sewa + maintenance) — hanya diisi ListVendors.
	VehicleCount     int64 `json:"vehicleCount"`
	MaintenanceCount int64 `json:"maintenanceCount"`
}

const vendorColumns = `v.id, v.name, v.type, v.address, v."picName", v.phone, v.email, v.note,
	v."isActive", v."createdAt", v."updatedAt"`

func scanVendor(row interface{ Scan(...any) error }, v *Vendor, extra ...any) error {
	dst := []any{&v.ID, &v.Name, &v.Type, &v.Address, &v.PicName, &v.Phone, &v.Email, &v.Note,
		&v.IsActive, &v.CreatedAt, &v.UpdatedAt}
	return row.Scan(append(dst, extra...)...)
}

type ListVendorsParams struct {
	Search   string
	Type     string // OWNER / WORKSHOP: juga mencakup BOTH
	IsActive sql.NullBool
}

func (q *Queries) ListVendors(ctx context.Context, p ListVendorsParams) ([]Vendor, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT `+vendorColumns+`,
		       (SELECT COUNT(*) FROM vehicles x WHERE x."ownerVendorId" = v.id),
		       (SELECT COUNT(*) FROM maintenance_records m WHERE m."vendorId" = v.id)
		FROM vendors v
		WHERE ($1 = '' OR v.name ILIKE '%' || $1 || '%' OR v."picName" ILIKE '%' || $1 || '%')
		  AND ($2 = '' OR v.type = $2 OR v.type = 'BOTH')
		  AND ($3::boolean IS NULL OR v."isActive" = $3::boolean)
		ORDER BY v."isActive" DESC, LOWER(v.name)`,
		strings.TrimSpace(p.Search), p.Type, p.IsActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Vendor{}
	for rows.Next() {
		var v Vendor
		if err := scanVendor(rows, &v, &v.VehicleCount, &v.MaintenanceCount); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (q *Queries) GetVendor(ctx context.Context, id int32) (Vendor, error) {
	var v Vendor
	err := scanVendor(q.db.QueryRowContext(ctx, `SELECT `+vendorColumns+` FROM vendors v WHERE v.id = $1`, id), &v)
	return v, err
}

type VendorParams struct {
	Name, Type                           string
	Address, PicName, Phone, Email, Note sql.NullString
}

func (q *Queries) CreateVendor(ctx context.Context, p VendorParams) (Vendor, error) {
	var v Vendor
	err := scanVendor(q.db.QueryRowContext(ctx, `
		INSERT INTO vendors AS v (name, type, address, "picName", phone, email, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING `+vendorColumns,
		p.Name, p.Type, p.Address, p.PicName, p.Phone, p.Email, p.Note), &v)
	return v, err
}

func (q *Queries) UpdateVendor(ctx context.Context, id int32, p VendorParams) (Vendor, error) {
	var v Vendor
	err := scanVendor(q.db.QueryRowContext(ctx, `
		UPDATE vendors AS v SET name = $2, type = $3, address = $4, "picName" = $5, phone = $6,
		       email = $7, note = $8, "updatedAt" = NOW()
		WHERE v.id = $1
		RETURNING `+vendorColumns,
		id, p.Name, p.Type, p.Address, p.PicName, p.Phone, p.Email, p.Note), &v)
	return v, err
}

func (q *Queries) SetVendorActive(ctx context.Context, id int32, active bool) error {
	_, err := q.db.ExecContext(ctx, `UPDATE vendors SET "isActive" = $2, "updatedAt" = NOW() WHERE id = $1`, id, active)
	return err
}

func (q *Queries) DeleteVendor(ctx context.Context, id int32) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM vendors WHERE id = $1`, id)
	return err
}

// ─── Kepemilikan kendaraan ──────────────────────────────────────────────────

type VehicleOwnership struct {
	VehicleID        int32          `json:"vehicleId"`
	Ownership        string         `json:"ownership"`
	OwnerVendorID    sql.NullInt32  `json:"ownerVendorId"`
	OwnerVendorName  sql.NullString `json:"ownerVendorName"`
	RentalContractNo sql.NullString `json:"rentalContractNo"`
}

// ListVehicleOwnership: kepemilikan untuk sekumpulan kendaraan (nil = semua).
func (q *Queries) ListVehicleOwnership(ctx context.Context, vehicleIDs []int32) (map[int32]VehicleOwnership, error) {
	ids := make([]int64, len(vehicleIDs))
	for i, id := range vehicleIDs {
		ids[i] = int64(id)
	}
	rows, err := q.db.QueryContext(ctx, `
		SELECT x.id, x.ownership, x."ownerVendorId", vd.name, x."rentalContractNo"
		FROM vehicles x LEFT JOIN vendors vd ON vd.id = x."ownerVendorId"
		WHERE x.id = ANY($1::bigint[])`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int32]VehicleOwnership{}
	for rows.Next() {
		var o VehicleOwnership
		if err := rows.Scan(&o.VehicleID, &o.Ownership, &o.OwnerVendorID, &o.OwnerVendorName, &o.RentalContractNo); err != nil {
			return nil, err
		}
		out[o.VehicleID] = o
	}
	return out, rows.Err()
}

func (q *Queries) GetVehicleOwnership(ctx context.Context, vehicleID int32) (VehicleOwnership, error) {
	m, err := q.ListVehicleOwnership(ctx, []int32{vehicleID})
	if err != nil {
		return VehicleOwnership{}, err
	}
	o, ok := m[vehicleID]
	if !ok {
		return VehicleOwnership{}, sql.ErrNoRows
	}
	return o, nil
}

func (q *Queries) SetVehicleOwnership(ctx context.Context, vehicleID int32, ownership string, vendorID sql.NullInt32, contractNo sql.NullString) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE vehicles SET ownership = $2, "ownerVendorId" = $3, "rentalContractNo" = $4 WHERE id = $1`,
		vehicleID, ownership, vendorID, contractNo)
	return err
}
