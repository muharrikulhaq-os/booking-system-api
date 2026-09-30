package service

import (
	"context"
	"database/sql"
	"strings"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

// VehicleOwnershipInput: kepemilikan kendaraan (docs/RANCANGAN_MAINTENANCE_VENDOR.md
// §3.2). Kosong saat update = tidak diubah.
type VehicleOwnershipInput struct {
	Ownership        string `json:"ownership"`
	OwnerVendorID    *int32 `json:"ownerVendorId"`
	RentalContractNo string `json:"rentalContractNo"`
}

// resolveOwnership memvalidasi input; ok=false bila tidak ada perubahan diminta.
func (s *VehicleService) resolveOwnership(ctx context.Context, in VehicleOwnershipInput) (string, sql.NullInt32, sql.NullString, bool, error) {
	own := strings.ToUpper(strings.TrimSpace(in.Ownership))
	switch own {
	case "":
		return "", sql.NullInt32{}, sql.NullString{}, false, nil
	case "COMPANY":
		return own, sql.NullInt32{}, sql.NullString{}, true, nil
	case "VENDOR":
		if in.OwnerVendorID == nil || *in.OwnerVendorID <= 0 {
			return "", sql.NullInt32{}, sql.NullString{}, false,
				util.NewError(400, "kendaraan sewa wajib memilih vendor pemiliknya", util.ErrBadRequest)
		}
		v, err := s.q.GetVendor(ctx, *in.OwnerVendorID)
		if err != nil {
			return "", sql.NullInt32{}, sql.NullString{}, false, util.NewError(400, "vendor tidak ditemukan", util.ErrBadRequest)
		}
		if !v.IsActive {
			return "", sql.NullInt32{}, sql.NullString{}, false, util.NewError(400, "vendor "+v.Name+" sedang nonaktif", util.ErrBadRequest)
		}
		if v.Type == "WORKSHOP" {
			return "", sql.NullInt32{}, sql.NullString{}, false,
				util.NewError(400, v.Name+" terdaftar sebagai bengkel saja - ubah jenis vendor menjadi Pemilik/keduanya", util.ErrBadRequest)
		}
		return own, sql.NullInt32{Int32: v.ID, Valid: true}, nullStrOf(in.RentalContractNo), true, nil
	}
	return "", sql.NullInt32{}, sql.NullString{}, false, util.NewError(400, "kepemilikan harus COMPANY atau VENDOR", util.ErrBadRequest)
}

// attachOwnership menambahkan ownership/ownerVendor/rentalContractNo ke hasil serialisasi.
func (s *VehicleService) attachOwnership(ctx context.Context, items ...map[string]any) {
	ids := make([]int32, 0, len(items))
	for _, it := range items {
		if id, ok := it["id"].(int32); ok {
			ids = append(ids, id)
		}
	}
	owns, _ := s.q.ListVehicleOwnership(ctx, ids)
	for _, it := range items {
		id, _ := it["id"].(int32)
		o, ok := owns[id]
		if !ok {
			o = repository.VehicleOwnership{Ownership: "COMPANY"}
		}
		it["ownership"] = o.Ownership
		it["rentalContractNo"] = nullStr(o.RentalContractNo)
		if o.OwnerVendorID.Valid {
			it["ownerVendor"] = map[string]any{"id": o.OwnerVendorID.Int32, "name": o.OwnerVendorName.String}
		} else {
			it["ownerVendor"] = nil
		}
	}
}
