package service

import (
	"context"
	"database/sql"
	"strconv"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

// AssignOptions: daftar supir & kendaraan untuk dropdown "Tugaskan" /
// "Alihkan & Setujui", masing-masing dengan pasangan tetapnya (supir ↔
// kendaraan tetap) dan ketersediaan untuk JADWAL BOOKING INI.
//
// Alasan tidak tersedia memakai pengecekan yang SAMA dengan AssignVehicle,
// supaya pilihan yang tampil "tersedia" tidak ditolak saat disimpan.
// Kapasitas kurang hanya peringatan (backend tidak menolaknya).
func (s *BookingService) AssignOptions(ctx context.Context, id int32) (map[string]any, error) {
	b, err := s.q.GetBookingByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if b.ResourceType != repository.ResourceTypeVEHICLE {
		return nil, util.NewError(400, "pilihan supir & kendaraan hanya untuk booking kendaraan", util.ErrBadRequest)
	}
	if b.Status != repository.BookingStatusPENDING && b.Status != repository.BookingStatusAPPROVED {
		return nil, util.NewError(409, "booking ini sudah tidak bisa ditugaskan ulang", util.ErrConflict)
	}

	vehicles, err := s.q.ListVehicles(ctx, repository.ListVehiclesParams{Limit: 1000})
	if err != nil {
		return nil, err
	}
	drivers, err := s.q.ListDrivers(ctx, repository.ListDriversParams{
		Limit: 1000, IsActive: sql.NullBool{Bool: true, Valid: true},
	})
	if err != nil {
		return nil, err
	}

	// Kendaraan tetap per supir (dari kolom fixedDriverId kendaraan).
	fixedVehicleOf := map[int32]map[string]any{}
	for _, v := range vehicles {
		if v.FixedDriverId.Valid {
			fixedVehicleOf[v.FixedDriverId.Int32] = map[string]any{
				"id": v.ID, "name": v.ResourceName, "plateNumber": v.PlateNumber,
			}
		}
	}

	vehicleOut := make([]map[string]any, 0, len(vehicles))
	for _, v := range vehicles {
		reason := s.vehicleUnavailableReason(ctx, b, v)
		var warning any
		if int32(v.Capacity) < b.PassengerCount {
			warning = "Kapasitas " + strconv.Itoa(int(v.Capacity)) + " kursi, penumpang " + strconv.Itoa(int(b.PassengerCount))
		}
		vehicleOut = append(vehicleOut, map[string]any{
			"id":          v.ID,
			"name":        v.ResourceName,
			"plateNumber": v.PlateNumber,
			"capacity":    v.Capacity,
			"status":      v.ResourceStatus,
			"fixedDriver": fixedDriverField(v.FixedDriverId, v.FixedDriverName),
			"available":   reason == "",
			"reason":      nilIfEmpty(reason),
			"warning":     warning,
			"isCurrent":   b.AssignedVehicleId.Valid && b.AssignedVehicleId.Int32 == v.ID,
		})
	}

	driverOut := make([]map[string]any, 0, len(drivers))
	for _, d := range drivers {
		if !d.UserIsActive { // sama dengan syarat AssignVehicle
			continue
		}
		reason := s.driverUnavailableReason(ctx, b, d.ID)
		var fixed any
		if fv, ok := fixedVehicleOf[d.ID]; ok {
			fixed = fv
		}
		driverOut = append(driverOut, map[string]any{
			"id":           d.ID,
			"name":         d.UserName,
			"employeeId":   d.EmployeeId,
			"phoneNumber":  d.PhoneNumber,
			"fixedVehicle": fixed,
			"available":    reason == "",
			"reason":       nilIfEmpty(reason),
			"isCurrent":    b.AssignedDriverId.Valid && b.AssignedDriverId.Int32 == d.ID,
		})
	}

	return map[string]any{"drivers": driverOut, "vehicles": vehicleOut}, nil
}

// vehicleUnavailableReason: "" bila kendaraan boleh ditugaskan ke booking b.
func (s *BookingService) vehicleUnavailableReason(ctx context.Context, b repository.GetBookingByIDRow, v repository.ListVehiclesRow) string {
	if v.ResourceStatus == repository.ResourceStatusINACTIVE {
		return "Kendaraan nonaktif"
	}
	if mc, err := s.q.CheckMaintenanceConflict(ctx, repository.CheckMaintenanceConflictParams{
		VehicleID: v.ID, CheckStart: b.StartDate, CheckEnd: b.EndDate,
	}); err == nil && mc > 0 {
		return "Jadwal maintenance di tanggal ini"
	}
	if c, err := s.q.CheckVehicleConflict(ctx, repository.CheckVehicleConflictParams{
		AssignedVehicleId: sql.NullInt32{Int32: v.ID, Valid: true},
		CheckStart:        b.StartDate,
		CheckEnd:          b.EndDate,
		ExcludeID:         b.ID,
	}); err == nil && c > 0 {
		return "Dipakai booking lain di jadwal ini"
	}
	if c, err := s.vehicleSpdConflict(ctx, v.ID, b.BookingType, b.StartDate, b.EndDate, b.ID); err == nil && c {
		return "Bertugas SPD di tanggal ini"
	}
	return ""
}

// driverUnavailableReason: "" bila supir boleh ditugaskan ke booking b.
func (s *BookingService) driverUnavailableReason(ctx context.Context, b repository.GetBookingByIDRow, driverID int32) string {
	if c, err := s.hasDriverConflict(ctx, driverID, b); err == nil && c {
		return "Bertugas di booking lain pada jadwal ini"
	}
	if c, err := s.driverSpdConflict(ctx, driverID, b.BookingType, b.StartDate, b.EndDate, b.ID); err == nil && c {
		return "Bertugas SPD di tanggal ini"
	}
	return ""
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
