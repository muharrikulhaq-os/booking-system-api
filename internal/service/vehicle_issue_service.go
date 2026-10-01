package service

import (
	"context"
	"database/sql"
	"fmt"
	"mime/multipart"
	"strings"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

// VehicleIssueService: laporan kendala/kerusakan kendaraan dari supir (termasuk
// masalah di jalan). Admin menindaklanjuti menjadi draf maintenance atau
// mengabaikannya (docs/RANCANGAN_MAINTENANCE_VENDOR.md §4.2).
type VehicleIssueService struct {
	q     repository.ExtendedQuerier
	maint *MaintenanceService
	notif *NotificationService
}

func NewVehicleIssueService(db *sql.DB, maint *MaintenanceService, notif *NotificationService) *VehicleIssueService {
	return &VehicleIssueService{q: repository.New(db), maint: maint, notif: notif}
}

type CreateVehicleIssueRequest struct {
	VehicleID   int32  `json:"vehicleId"   form:"vehicleId"   validate:"required"`
	Description string `json:"description" form:"description" validate:"required"`
	Location    string `json:"location"    form:"location"`
	CanContinue bool   `json:"canContinue" form:"canContinue"`
}

type ResolveVehicleIssueRequest struct {
	Note string `json:"note"`
}

func serializeIssue(i repository.VehicleIssue) map[string]any {
	return map[string]any{
		"id":            i.ID,
		"vehicle":       map[string]any{"id": i.VehicleID, "name": i.VehicleName, "plateNumber": i.PlateNumber},
		"bookingId":     nullInt32(i.BookingID),
		"reportedBy":    map[string]any{"id": i.ReportedByID, "name": i.ReportedByName},
		"description":   i.Description,
		"location":      nullStr(i.Location),
		"photos":        jsonStrings(i.Photos),
		"canContinue":   i.CanContinue,
		"status":        i.Status,
		"handledBy":     nullStr(i.HandledByName),
		"handledNote":   nullStr(i.HandledNote),
		"handledAt":     nullTime(i.HandledAt),
		"maintenanceId": nullInt32(i.MaintenanceID),
		"createdAt":     i.CreatedAt,
	}
}

// MyVehicles: kendaraan yang bisa dilaporkan supir yang sedang login.
func (s *VehicleIssueService) MyVehicles(ctx context.Context, userID int32) ([]repository.DriverVehicle, error) {
	d, err := s.q.GetDriverByUserID(ctx, userID)
	if err != nil {
		return []repository.DriverVehicle{}, nil
	}
	return s.q.ListDriverVehicles(ctx, d.ID)
}

func (s *VehicleIssueService) Create(ctx context.Context, req CreateVehicleIssueRequest, photos []*multipart.FileHeader, role string, actor AuditActor) (map[string]any, error) {
	if strings.TrimSpace(req.Description) == "" {
		return nil, util.NewError(400, "uraian kendala wajib diisi", util.ErrBadRequest)
	}
	vehicle, err := s.q.GetVehicleByID(ctx, req.VehicleID)
	if err != nil {
		return nil, util.NewError(404, "kendaraan tidak ditemukan", util.ErrNotFound)
	}
	var bookingID sql.NullInt32
	if role == "DRIVER" {
		d, derr := s.q.GetDriverByUserID(ctx, actor.UserID)
		if derr != nil {
			return nil, util.ErrForbidden
		}
		list, _ := s.q.ListDriverVehicles(ctx, d.ID)
		allowed := false
		for _, dv := range list {
			if dv.VehicleID == vehicle.ID {
				allowed, bookingID = true, dv.BookingID
				break
			}
		}
		if !allowed {
			return nil, util.NewError(403, "Anda hanya bisa melaporkan kendaraan yang sedang/akan Anda bawa", util.ErrForbidden)
		}
	}
	if len(photos) > 5 {
		return nil, util.NewError(400, "maksimal 5 foto", util.ErrBadRequest)
	}
	var urls []string
	for _, fh := range photos {
		path, err := util.SaveUploadedFile(fh, "vehicle-issue")
		if err != nil {
			return nil, util.NewError(400, err.Error(), util.ErrBadRequest)
		}
		urls = append(urls, "/uploads/"+strings.TrimPrefix(path, "/"))
	}
	id, err := s.q.InsertVehicleIssue(ctx, vehicle.ID, bookingID, actor.UserID, strings.TrimSpace(req.Description),
		nullStrOf(req.Location), urls, req.CanContinue)
	if err != nil {
		return nil, err
	}
	logAudit(ctx, s.q, actor, "REPORT_ISSUE", "VehicleIssue", id, "Melaporkan kendala kendaraan "+vehicle.PlateNumber)
	if s.notif != nil {
		title := "Laporan kendala kendaraan"
		msg := fmt.Sprintf("Kendala pada %s: %s", vehicle.PlateNumber, truncate(req.Description, 80))
		if !req.CanContinue {
			title = "DARURAT: kendaraan tidak bisa jalan"
		}
		s.notif.NotifyAdmins("VEHICLE_ISSUE", title, msg, map[string]any{"issueId": id, "vehicleId": vehicle.ID})
	}
	i, _ := s.q.GetVehicleIssue(ctx, id)
	return serializeIssue(i), nil
}

func truncate(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}

func (s *VehicleIssueService) List(ctx context.Context, page, limit int, status string, vehicleID *int32, role string, userID int32) ([]map[string]any, int64, error) {
	p := repository.ListVehicleIssuesParams{Status: strings.ToUpper(status), Limit: int32(limit), Offset: int32((page - 1) * limit)}
	if vehicleID != nil {
		p.VehicleID = sql.NullInt32{Int32: *vehicleID, Valid: true}
	}
	if role != "ADMIN" {
		p.ReportedBy = sql.NullInt32{Int32: userID, Valid: true}
	}
	rows, total, err := s.q.ListVehicleIssues(ctx, p)
	if err != nil {
		return nil, 0, err
	}
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeIssue(r)
	}
	return out, total, nil
}

func (s *VehicleIssueService) GetByID(ctx context.Context, id int32, role string, userID int32) (map[string]any, error) {
	i, err := s.q.GetVehicleIssue(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if role != "ADMIN" && i.ReportedByID != userID {
		return nil, util.ErrForbidden
	}
	return serializeIssue(i), nil
}

// Convert: laporan → draf maintenance (atau ditautkan ke maintenance kendaraan
// yang masih berjalan, supaya tidak ada dua proses untuk satu kendaraan).
func (s *VehicleIssueService) Convert(ctx context.Context, id int32, actor AuditActor) (map[string]any, error) {
	i, err := s.q.GetVehicleIssue(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if i.Status != "OPEN" {
		return nil, util.NewError(409, "laporan ini sudah ditindaklanjuti", util.ErrConflict)
	}
	var maintID int32
	note := "Dibuatkan pengajuan maintenance"
	if openID, oerr := s.q.GetOpenMaintenanceID(ctx, i.VehicleID, 0); oerr == nil {
		maintID = openID
		note = fmt.Sprintf("Ditautkan ke maintenance #%d yang masih berjalan", openID)
	} else {
		complaint := i.Description
		if i.Location.Valid && i.Location.String != "" {
			complaint += " (lokasi: " + i.Location.String + ")"
		}
		resp, cerr := s.maint.create(ctx, MaintenancePlanRequest{
			VehicleID: i.VehicleID, Category: "REPAIR",
			Description: "Pemeriksaan & perbaikan sesuai laporan kendala supir",
			Complaint:   complaint,
		}, sql.NullInt32{Int32: i.ID, Valid: true}, actor)
		if cerr != nil {
			return nil, cerr
		}
		maintID, _ = resp.Data["id"].(int32)
	}
	ok, err := s.q.ResolveVehicleIssue(ctx, id, "CONVERTED", actor.UserID, nullStrOf(note), sql.NullInt32{Int32: maintID, Valid: maintID != 0})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, util.NewError(409, "laporan ini sudah ditindaklanjuti", util.ErrConflict)
	}
	logAudit(ctx, s.q, actor, "CONVERT_ISSUE", "VehicleIssue", id, note)
	s.notifyReporter(i, "Laporan kendala "+i.PlateNumber+" ditindaklanjuti: "+note)
	out, _ := s.GetByID(ctx, id, "ADMIN", actor.UserID)
	return out, nil
}

func (s *VehicleIssueService) Dismiss(ctx context.Context, id int32, req ResolveVehicleIssueRequest, actor AuditActor) (map[string]any, error) {
	i, err := s.q.GetVehicleIssue(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if strings.TrimSpace(req.Note) == "" {
		return nil, util.NewError(400, "catatan wajib diisi saat mengabaikan laporan", util.ErrBadRequest)
	}
	ok, err := s.q.ResolveVehicleIssue(ctx, id, "DISMISSED", actor.UserID, nullStrOf(req.Note), sql.NullInt32{})
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, util.NewError(409, "laporan ini sudah ditindaklanjuti", util.ErrConflict)
	}
	logAudit(ctx, s.q, actor, "DISMISS_ISSUE", "VehicleIssue", id, "Laporan diabaikan: "+req.Note)
	s.notifyReporter(i, "Laporan kendala "+i.PlateNumber+" ditutup admin: "+req.Note)
	return s.GetByID(ctx, id, "ADMIN", actor.UserID)
}

func (s *VehicleIssueService) notifyReporter(i repository.VehicleIssue, msg string) {
	if s.notif != nil {
		s.notif.Notify(i.ReportedByID, "VEHICLE_ISSUE_UPDATE", "Laporan kendala", msg, map[string]any{"issueId": i.ID})
	}
}
