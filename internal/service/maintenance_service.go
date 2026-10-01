package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

// MaintenanceService: maintenance oleh vendor/bengkel luar —
// DRAFT → SUBMITTED → SCHEDULED → IN_PROGRESS → COMPLETED (CANCELLED sebelum
// IN_PROGRESS). Lihat docs/RANCANGAN_MAINTENANCE_VENDOR.md.
//
// Kolom "startDate"/"endDate" adalah jendela blokir booking yang dijaga di
// sini (CheckMaintenanceConflict): rencana (SUBMITTED), jadwal vendor
// (SCHEDULED), atau terbuka sejak serah terima (IN_PROGRESS, endDate NULL).
// Kendaraan berstatus MAINTENANCE hanya selama IN_PROGRESS (syncResourceStatus).
type MaintenanceService struct {
	q     repository.ExtendedQuerier
	notif *NotificationService
}

func NewMaintenanceService(db *sql.DB, notif *NotificationService) *MaintenanceService {
	return &MaintenanceService{q: repository.New(db), notif: notif}
}

var (
	maintenanceCategories = map[string]string{
		"ROUTINE": "Servis berkala",
		"REPAIR":  "Perbaikan",
		"PARTS":   "Ganti part",
		"BODY":    "Body / cat",
		"OTHER":   "Lainnya",
	}
	pickupMethods = map[string]string{"DROP_OFF": "Diantar ke vendor", "PICKUP": "Dijemput vendor"}
	costBearers   = map[string]string{"COMPANY": "Perusahaan", "VENDOR": "Vendor", "UNDECIDED": "Belum ditentukan"}
	fuelLevels    = map[string]bool{"E": true, "1/4": true, "1/2": true, "3/4": true, "F": true}
	// Checklist kelengkapan berita acara (§6.3). Urutan = urutan cetak.
	handoverChecklistItems = []struct{ Key, Label string }{
		{"stnk", "STNK"},
		{"mainKey", "Kunci utama"},
		{"spareKey", "Kunci cadangan"},
		{"spareTire", "Ban serep"},
		{"jack", "Dongkrak & kunci roda"},
		{"warningTriangle", "Segitiga pengaman"},
		{"firstAid", "Kotak P3K"},
	}
	maintenanceEditable = []string{repository.MaintDraft, repository.MaintSubmitted, repository.MaintScheduled}
)

// ─── Request ─────────────────────────────────────────────────────────────────

type MaintenancePlanRequest struct {
	VehicleID     int32      `json:"vehicleId"     validate:"required"`
	VendorID      *int32     `json:"vendorId"`
	Category      string     `json:"category"      validate:"required"`
	Description   string     `json:"description"   validate:"required"`
	Complaint     string     `json:"complaint"`
	Location      string     `json:"location"`
	PlannedDate   *time.Time `json:"plannedDate"`
	EstimatedDays *int32     `json:"estimatedDays"`
	PickupMethod  string     `json:"pickupMethod"`
	EstimatedCost *float64   `json:"estimatedCost"`
	CostBearer    string     `json:"costBearer"`
	Odometer      *int32     `json:"odometer"`
	// Create saja: langsung ajukan (buat nomor surat) setelah disimpan.
	Submit bool `json:"submit"`
}

type ScheduleMaintenanceRequest struct {
	ScheduledDate time.Time `json:"scheduledDate" validate:"required"`
	EstimatedDays *int32    `json:"estimatedDays"`
	Note          string    `json:"note"`
}

type HandoverMaintenanceRequest struct {
	HandoverAt   *time.Time      `json:"handoverAt"`
	Odometer     *int32          `json:"odometer"`
	FuelLevel    string          `json:"fuelLevel"`
	ReceiverName string          `json:"receiverName" validate:"required"`
	Checklist    map[string]bool `json:"checklist"`
	Note         string          `json:"note"`
}

type ReturnMaintenanceRequest struct {
	ReturnedAt    *time.Time      `json:"returnedAt"`
	Odometer      *int32          `json:"odometer"`
	FuelLevel     string          `json:"fuelLevel"`
	HandlerName   string          `json:"handlerName"`
	Checklist     map[string]bool `json:"checklist"`
	WorkDone      string          `json:"workDone" validate:"required"`
	PartsReplaced string          `json:"partsReplaced"`
	Note          string          `json:"note"`
	ActualCost    *float64        `json:"actualCost"`
	CostBearer    string          `json:"costBearer"`
}

type CancelMaintenanceRequest struct {
	Reason string `json:"reason"`
}

type MaintenanceCostRequest struct {
	EstimatedCost *float64 `json:"estimatedCost"`
	ActualCost    *float64 `json:"actualCost"`
	CostBearer    string   `json:"costBearer"`
}

// MaintenanceActionResponse: data + peringatan opsional (mis. jadwal bentrok
// dengan booking yang sudah disetujui).
type MaintenanceActionResponse struct {
	Data    map[string]any `json:"data"`
	Warning string         `json:"warning,omitempty"`
}

// ─── Serialisasi ─────────────────────────────────────────────────────────────

func nullNumeric(f *float64) sql.NullString {
	if f == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: strconv.FormatFloat(*f, 'f', 2, 64), Valid: true}
}

func numOrNil(s sql.NullString) any {
	if !s.Valid {
		return nil
	}
	f, err := strconv.ParseFloat(s.String, 64)
	if err != nil {
		return nil
	}
	return f
}

func nullStrOf(s string) sql.NullString {
	s = strings.TrimSpace(s)
	return sql.NullString{String: s, Valid: s != ""}
}

func jsonChecklist(b []byte) map[string]bool {
	out := map[string]bool{}
	if len(b) > 0 {
		_ = json.Unmarshal(b, &out)
	}
	return out
}

func jsonStrings(b []byte) []string {
	out := []string{}
	if len(b) > 0 {
		_ = json.Unmarshal(b, &out)
	}
	return out
}

func serializeMaintenance(m repository.MaintenanceItem) map[string]any {
	vendorName := m.VendorName
	if !vendorName.Valid {
		vendorName = m.VendorLegacy
	}
	var vendor any
	if m.VendorID.Valid {
		vendor = map[string]any{
			"id": m.VendorID.Int32, "name": m.VendorName.String, "address": nullStr(m.VendorAddress),
			"picName": nullStr(m.VendorPic), "phone": nullStr(m.VendorPhone),
		}
	}
	var handover, ret any
	if m.HandoverAt.Valid {
		handover = map[string]any{
			"at": m.HandoverAt.Time, "odometer": nullInt32(m.HandoverOdo), "fuelLevel": nullStr(m.HandoverFuel),
			"receiverName": nullStr(m.HandoverBy), "checklist": jsonChecklist(m.HandoverList), "note": nullStr(m.HandoverNote),
		}
	}
	if m.ReturnedAt.Valid {
		ret = map[string]any{
			"at": m.ReturnedAt.Time, "odometer": nullInt32(m.ReturnOdo), "fuelLevel": nullStr(m.ReturnFuel),
			"handlerName": nullStr(m.ReturnBy), "checklist": jsonChecklist(m.ReturnList),
			"workDone": nullStr(m.WorkDone), "partsReplaced": nullStr(m.PartsReplaced), "note": nullStr(m.ReturnNote),
		}
	}
	category := m.Category.String
	if category == "" {
		category = "OTHER"
	}
	return map[string]any{
		"id":        m.ID,
		"requestNo": nullStr(m.RequestNo),
		"status":    m.Status,
		"vehicle": map[string]any{
			"id": m.VehicleID, "name": m.VehicleName, "plateNumber": m.PlateNumber, "photoUrl": nullStr(m.VehiclePhotoURL),
			"brand": m.Brand, "model": m.Model, "year": m.Year, "currentOdometer": m.CurrentOdometer,
			"ownership": m.Ownership, "ownerVendorName": nullStr(m.OwnerVendorName), "rentalContractNo": nullStr(m.RentalContract),
		},
		// Kompatibilitas daftar lama (web/mobile sebelum rancangan vendor).
		"vehicleId":     m.VehicleID,
		"vehicleName":   m.VehicleName,
		"plateNumber":   m.PlateNumber,
		"vendor":        vendor,
		"vendorName":    nullStr(vendorName),
		"category":      category,
		"categoryLabel": maintenanceCategories[category],
		"description":   m.Description,
		"complaint":     nullStr(m.Complaint),
		"location":      nullStr(m.Location),
		"plannedDate":   nullTime(m.PlannedDate),
		"estimatedDays": nullInt32(m.EstimatedDays),
		"scheduledDate": nullTime(m.ScheduledDate),
		"scheduleNote":  nullStr(m.ScheduleNote),
		"pickupMethod":  nullStr(m.PickupMethod),
		"estimatedCost": numOrNil(m.EstimatedCost),
		"actualCost":    numOrNil(m.TotalCost),
		"totalCost":     numOrNil(m.TotalCost),
		"costBearer":    nullStr(m.CostBearer),
		"odometer":      nullInt32(m.Odometer),
		"submittedAt":   nullTime(m.SubmittedAt),
		"handover":      handover,
		"return":        ret,
		"completedAt":   nullTime(m.CompletedAt),
		"cancelledAt":   nullTime(m.CancelledAt),
		"cancelReason":  nullStr(m.CancelReason),
		"proofPhotos":   jsonStrings(m.ProofPhotos),
		"sourceIssueId": nullInt32(m.SourceIssueID),
		"createdBy":     m.RecordedByName,
		"createdAt":     m.CreatedAt,
		"updatedAt":     m.UpdatedAt,
		// Blokir booking saat ini (nil = tidak memblokir).
		"blockStart": blockStart(m),
		"blockEnd":   blockEnd(m),
	}
}

func blockStart(m repository.MaintenanceItem) any {
	switch m.Status {
	case repository.MaintSubmitted, repository.MaintScheduled, repository.MaintInProgress:
		return m.StartDate
	}
	return nil
}

func blockEnd(m repository.MaintenanceItem) any {
	switch m.Status {
	case repository.MaintSubmitted, repository.MaintScheduled:
		return nullTime(m.EndDate)
	}
	return nil
}

// ─── Baca ────────────────────────────────────────────────────────────────────

type MaintenanceListFilter struct {
	VehicleID *int32
	VendorID  *int32
	Status    string // satu status atau dipisah koma; "ACTIVE" = semua yang belum final
	Search    string
	SortBy    string
	SortOrder string
}

func (s *MaintenanceService) List(ctx context.Context, page, limit int, f MaintenanceListFilter) ([]map[string]any, int64, error) {
	p := repository.ListMaintenanceRecordsParams{
		Search: f.Search, SortBy: f.SortBy, SortOrder: f.SortOrder,
		Limit: int32(limit), Offset: int32((page - 1) * limit),
	}
	if f.VehicleID != nil {
		p.VehicleID = sql.NullInt32{Int32: *f.VehicleID, Valid: true}
	}
	if f.VendorID != nil {
		p.VendorID = sql.NullInt32{Int32: *f.VendorID, Valid: true}
	}
	for _, st := range strings.Split(f.Status, ",") {
		switch st = strings.ToUpper(strings.TrimSpace(st)); st {
		case "":
		case "ACTIVE":
			p.Statuses = append(p.Statuses, repository.MaintDraft, repository.MaintSubmitted,
				repository.MaintScheduled, repository.MaintInProgress)
		default:
			p.Statuses = append(p.Statuses, st)
		}
	}
	rows, total, err := s.q.ListMaintenanceRecords(ctx, p)
	if err != nil {
		return nil, 0, err
	}
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeMaintenance(r)
	}
	return out, total, nil
}

func (s *MaintenanceService) GetByID(ctx context.Context, id int32) (map[string]any, error) {
	m, err := s.q.GetMaintenanceRecord(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	out := serializeMaintenance(m)
	docs, _ := s.q.ListMaintenanceDocuments(ctx, id)
	out["documents"] = docs
	return out, nil
}

// ─── Pengajuan ───────────────────────────────────────────────────────────────

// buildPlan memvalidasi isian pengajuan & menentukan vendor (kendaraan sewa →
// selalu vendor pemiliknya).
func (s *MaintenanceService) buildPlan(ctx context.Context, req MaintenancePlanRequest) (repository.MaintenancePlan, repository.GetVehicleByIDRow, error) {
	var p repository.MaintenancePlan
	vehicle, err := s.q.GetVehicleByID(ctx, req.VehicleID)
	if err != nil {
		return p, vehicle, util.NewError(404, "kendaraan tidak ditemukan", util.ErrNotFound)
	}
	cat := strings.ToUpper(strings.TrimSpace(req.Category))
	if _, ok := maintenanceCategories[cat]; !ok {
		return p, vehicle, util.NewError(400, "kategori maintenance tidak dikenal", util.ErrBadRequest)
	}
	if strings.TrimSpace(req.Description) == "" {
		return p, vehicle, util.NewError(400, "uraian pekerjaan wajib diisi", util.ErrBadRequest)
	}
	pickup := strings.ToUpper(strings.TrimSpace(req.PickupMethod))
	if _, ok := pickupMethods[pickup]; pickup != "" && !ok {
		return p, vehicle, util.NewError(400, "cara serah tidak dikenal", util.ErrBadRequest)
	}
	bearer := strings.ToUpper(strings.TrimSpace(req.CostBearer))
	if _, ok := costBearers[bearer]; bearer != "" && !ok {
		return p, vehicle, util.NewError(400, "penanggung biaya tidak dikenal", util.ErrBadRequest)
	}
	if req.EstimatedCost != nil && *req.EstimatedCost < 0 {
		return p, vehicle, util.NewError(400, "estimasi biaya tidak boleh negatif", util.ErrBadRequest)
	}
	days := int32(1)
	if req.EstimatedDays != nil {
		if *req.EstimatedDays < 1 || *req.EstimatedDays > 365 {
			return p, vehicle, util.NewError(400, "estimasi lama pengerjaan 1–365 hari", util.ErrBadRequest)
		}
		days = *req.EstimatedDays
	}

	vendorID := sql.NullInt32{}
	if req.VendorID != nil && *req.VendorID > 0 {
		vendorID = sql.NullInt32{Int32: *req.VendorID, Valid: true}
	}
	if own, oerr := s.q.GetVehicleOwnership(ctx, vehicle.ID); oerr == nil && own.Ownership == "VENDOR" {
		vendorID = own.OwnerVendorID
	}
	if vendorID.Valid {
		v, verr := s.q.GetVendor(ctx, vendorID.Int32)
		if verr != nil {
			return p, vehicle, util.NewError(400, "vendor tidak ditemukan", util.ErrBadRequest)
		}
		if !v.IsActive {
			return p, vehicle, util.NewError(400, "vendor "+v.Name+" sedang nonaktif", util.ErrBadRequest)
		}
	}

	p = repository.MaintenancePlan{
		VehicleID: vehicle.ID, VendorID: vendorID, Category: cat,
		Description: strings.TrimSpace(req.Description), Complaint: nullStrOf(req.Complaint),
		Location: nullStrOf(req.Location), EstimatedDays: sql.NullInt32{Int32: days, Valid: true},
		PickupMethod: nullStrOf(pickup), EstimatedCost: nullNumeric(req.EstimatedCost),
		CostBearer: nullStrOf(bearer), Odometer: nullInt32PtrOf(req.Odometer),
	}
	if req.PlannedDate != nil {
		p.PlannedDate = sql.NullTime{Time: *req.PlannedDate, Valid: true}
	}
	return p, vehicle, nil
}

func nullInt32PtrOf(v *int32) sql.NullInt32 {
	if v == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: *v, Valid: true}
}

// planWindow: jendela blokir [start, start + hari).
func planWindow(start time.Time, days sql.NullInt32) (time.Time, sql.NullTime) {
	d := int32(1)
	if days.Valid && days.Int32 > 0 {
		d = days.Int32
	}
	return start, sql.NullTime{Time: start.Add(time.Duration(d) * 24 * time.Hour), Valid: true}
}

// windowFor menentukan jendela sesuai status (DRAFT tidak memblokir tapi tetap
// diisi supaya kolom NOT NULL terpenuhi).
func windowFor(status string, p repository.MaintenancePlan, scheduled sql.NullTime) (time.Time, sql.NullTime) {
	start := time.Now()
	if status == repository.MaintScheduled && scheduled.Valid {
		start = scheduled.Time
	} else if p.PlannedDate.Valid {
		start = p.PlannedDate.Time
	}
	return planWindow(start, p.EstimatedDays)
}

func (s *MaintenanceService) Create(ctx context.Context, req MaintenancePlanRequest, actor AuditActor) (MaintenanceActionResponse, error) {
	return s.create(ctx, req, sql.NullInt32{}, actor)
}

func (s *MaintenanceService) create(ctx context.Context, req MaintenancePlanRequest, sourceIssue sql.NullInt32, actor AuditActor) (MaintenanceActionResponse, error) {
	p, vehicle, err := s.buildPlan(ctx, req)
	if err != nil {
		return MaintenanceActionResponse{}, err
	}
	if openID, oerr := s.q.GetOpenMaintenanceID(ctx, vehicle.ID, 0); oerr == nil {
		return MaintenanceActionResponse{}, util.NewError(409,
			fmt.Sprintf("kendaraan ini masih punya maintenance yang belum selesai (#%d)", openID), util.ErrConflict)
	}
	p.StartDate, p.EndDate = windowFor(repository.MaintDraft, p, sql.NullTime{})
	id, err := s.q.InsertMaintenance(ctx, p, sourceIssue, actor.UserID)
	if err != nil {
		return MaintenanceActionResponse{}, err
	}
	logAudit(ctx, s.q, actor, "CREATE", "Maintenance", id, "Membuat draf maintenance kendaraan "+vehicle.PlateNumber)
	if req.Submit {
		return s.Submit(ctx, id, actor)
	}
	data, _ := s.GetByID(ctx, id)
	return MaintenanceActionResponse{Data: data}, nil
}

func (s *MaintenanceService) Update(ctx context.Context, id int32, req MaintenancePlanRequest, actor AuditActor) (MaintenanceActionResponse, error) {
	existing, err := s.q.GetMaintenanceRecord(ctx, id)
	if err != nil {
		return MaintenanceActionResponse{}, util.ErrNotFound
	}
	if !contains(maintenanceEditable, existing.Status) {
		return MaintenanceActionResponse{}, util.NewError(409,
			"pengajuan yang sudah dikerjakan/selesai/dibatalkan tidak bisa diubah (biaya & dokumen tetap bisa)", util.ErrConflict)
	}
	if existing.Status != repository.MaintDraft && req.VehicleID != existing.VehicleID {
		return MaintenanceActionResponse{}, util.NewError(409,
			"kendaraan tidak bisa diganti setelah surat diajukan - batalkan lalu buat pengajuan baru", util.ErrConflict)
	}
	p, vehicle, err := s.buildPlan(ctx, req)
	if err != nil {
		return MaintenanceActionResponse{}, err
	}
	if existing.Status != repository.MaintDraft {
		if !p.VendorID.Valid {
			return MaintenanceActionResponse{}, util.NewError(400, "vendor tujuan wajib diisi", util.ErrBadRequest)
		}
		if !p.PlannedDate.Valid {
			return MaintenanceActionResponse{}, util.NewError(400, "rencana tanggal wajib diisi", util.ErrBadRequest)
		}
	}
	if req.VehicleID != existing.VehicleID {
		if openID, oerr := s.q.GetOpenMaintenanceID(ctx, vehicle.ID, id); oerr == nil {
			return MaintenanceActionResponse{}, util.NewError(409,
				fmt.Sprintf("kendaraan ini masih punya maintenance yang belum selesai (#%d)", openID), util.ErrConflict)
		}
	}
	p.StartDate, p.EndDate = windowFor(existing.Status, p, existing.ScheduledDate)
	ok, err := s.q.UpdateMaintenancePlan(ctx, id, p, maintenanceEditable)
	if err != nil {
		return MaintenanceActionResponse{}, err
	}
	if !ok {
		return MaintenanceActionResponse{}, errStatusChanged
	}
	logAudit(ctx, s.q, actor, "UPDATE", "Maintenance", id, "Mengubah pengajuan maintenance kendaraan "+vehicle.PlateNumber)
	data, _ := s.GetByID(ctx, id)
	resp := MaintenanceActionResponse{Data: data}
	if existing.Status != repository.MaintDraft {
		resp.Warning = s.bookingConflictWarning(ctx, vehicle.ResourceId, p.StartDate, p.EndDate)
	}
	return resp, nil
}

var errStatusChanged = util.NewError(409, "status maintenance baru saja berubah - muat ulang lalu coba lagi", util.ErrConflict)

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// bookingConflictWarning: booking yang sudah disetujui/berjalan di kendaraan
// ini pada jendela maintenance — admin perlu mengalihkannya (MT-03).
func (s *MaintenanceService) bookingConflictWarning(ctx context.Context, resourceID int32, start time.Time, end sql.NullTime) string {
	e := start.AddDate(10, 0, 0)
	if end.Valid {
		e = end.Time
	}
	n, err := s.q.CountActiveResourceOverlap(ctx, resourceID, start, e, 0)
	if err != nil || n == 0 {
		return ""
	}
	return fmt.Sprintf("Ada %d booking disetujui/berjalan yang bentrok dengan jadwal maintenance ini - alihkan ke kendaraan lain.", n)
}

// Submit: DRAFT → SUBMITTED, membuat nomor surat.
func (s *MaintenanceService) Submit(ctx context.Context, id int32, actor AuditActor) (MaintenanceActionResponse, error) {
	m, err := s.q.GetMaintenanceRecord(ctx, id)
	if err != nil {
		return MaintenanceActionResponse{}, util.ErrNotFound
	}
	if m.Status != repository.MaintDraft {
		return MaintenanceActionResponse{}, util.NewError(409, "hanya draf yang bisa diajukan", util.ErrConflict)
	}
	if !m.VendorID.Valid {
		return MaintenanceActionResponse{}, util.NewError(400, "pilih vendor tujuan sebelum mengajukan", util.ErrBadRequest)
	}
	if !m.PlannedDate.Valid {
		return MaintenanceActionResponse{}, util.NewError(400, "isi rencana tanggal sebelum mengajukan", util.ErrBadRequest)
	}
	settings, _ := s.q.GetDocumentSettings(ctx)
	now := time.Now()
	year := now.In(util.WIB).Year()
	seq, err := s.q.NextDocumentNumber(ctx, fmt.Sprintf("MNT-%d", year))
	if err != nil {
		return MaintenanceActionResponse{}, err
	}
	requestNo := formatLetterNo(seq, settings.LetterCode, now)
	start, end := planWindow(m.PlannedDate.Time, m.EstimatedDays)
	ok, err := s.q.TransitionMaintenance(ctx, id, []string{repository.MaintDraft}, repository.MaintSubmitted,
		`"requestNo" = $3, "submittedAt" = NOW(), "startDate" = $4, "endDate" = $5`, requestNo, start, end)
	if err != nil {
		return MaintenanceActionResponse{}, err
	}
	if !ok {
		return MaintenanceActionResponse{}, errStatusChanged
	}
	logAudit(ctx, s.q, actor, "SUBMIT", "Maintenance", id, "Mengajukan maintenance "+requestNo+" ("+m.PlateNumber+")")
	data, _ := s.GetByID(ctx, id)
	vehicle, _ := s.q.GetVehicleByID(ctx, m.VehicleID)
	return MaintenanceActionResponse{Data: data,
		Warning: s.bookingConflictWarning(ctx, vehicle.ResourceId, start, end)}, nil
}

var romanMonths = []string{"", "I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII"}

// formatLetterNo: {urut 3 digit}/{kode}/{bulan romawi}/{tahun} menurut WIB.
func formatLetterNo(seq int, code string, t time.Time) string {
	code = strings.TrimSpace(code)
	if code == "" {
		code = "KCE-MNT"
	}
	w := t.In(util.WIB)
	return fmt.Sprintf("%03d/%s/%s/%d", seq, code, romanMonths[int(w.Month())], w.Year())
}

// Schedule: SUBMITTED/SCHEDULED → SCHEDULED (konfirmasi / ubah jadwal vendor).
func (s *MaintenanceService) Schedule(ctx context.Context, id int32, req ScheduleMaintenanceRequest, actor AuditActor) (MaintenanceActionResponse, error) {
	m, err := s.q.GetMaintenanceRecord(ctx, id)
	if err != nil {
		return MaintenanceActionResponse{}, util.ErrNotFound
	}
	if m.Status != repository.MaintSubmitted && m.Status != repository.MaintScheduled {
		return MaintenanceActionResponse{}, util.NewError(409, "hanya pengajuan yang sudah diajukan yang bisa dijadwalkan", util.ErrConflict)
	}
	days := m.EstimatedDays
	if req.EstimatedDays != nil {
		if *req.EstimatedDays < 1 || *req.EstimatedDays > 365 {
			return MaintenanceActionResponse{}, util.NewError(400, "estimasi lama pengerjaan 1–365 hari", util.ErrBadRequest)
		}
		days = sql.NullInt32{Int32: *req.EstimatedDays, Valid: true}
	}
	start, end := planWindow(req.ScheduledDate, days)
	ok, err := s.q.TransitionMaintenance(ctx, id, []string{repository.MaintSubmitted, repository.MaintScheduled},
		repository.MaintScheduled, `"scheduledDate" = $3, "estimatedDays" = $4, "scheduleNote" = $5, "startDate" = $6, "endDate" = $7`,
		req.ScheduledDate, days, nullStrOf(req.Note), start, end)
	if err != nil {
		return MaintenanceActionResponse{}, err
	}
	if !ok {
		return MaintenanceActionResponse{}, errStatusChanged
	}
	logAudit(ctx, s.q, actor, "SCHEDULE", "Maintenance", id,
		"Jadwal maintenance dari vendor: "+req.ScheduledDate.In(util.WIB).Format("02-01-2006 15:04")+" WIB")
	data, _ := s.GetByID(ctx, id)
	vehicle, _ := s.q.GetVehicleByID(ctx, m.VehicleID)
	return MaintenanceActionResponse{Data: data, Warning: s.bookingConflictWarning(ctx, vehicle.ResourceId, start, end)}, nil
}

func validateChecklist(c map[string]bool) ([]byte, error) {
	known := map[string]bool{}
	for _, it := range handoverChecklistItems {
		known[it.Key] = true
	}
	clean := map[string]bool{}
	for k, v := range c {
		if !known[k] {
			return nil, util.NewError(400, "item checklist tidak dikenal: "+k, util.ErrBadRequest)
		}
		clean[k] = v
	}
	return json.Marshal(clean)
}

func validFuel(level string) (sql.NullString, error) {
	level = strings.ToUpper(strings.TrimSpace(level))
	if level == "" {
		return sql.NullString{}, nil
	}
	if !fuelLevels[level] {
		return sql.NullString{}, util.NewError(400, "level BBM harus E, 1/4, 1/2, 3/4, atau F", util.ErrBadRequest)
	}
	return sql.NullString{String: level, Valid: true}, nil
}

// Handover: SUBMITTED/SCHEDULED → IN_PROGRESS (kendaraan diserahkan ke vendor).
func (s *MaintenanceService) Handover(ctx context.Context, id int32, req HandoverMaintenanceRequest, actor AuditActor) (MaintenanceActionResponse, error) {
	m, err := s.q.GetMaintenanceRecord(ctx, id)
	if err != nil {
		return MaintenanceActionResponse{}, util.ErrNotFound
	}
	if m.Status != repository.MaintSubmitted && m.Status != repository.MaintScheduled {
		return MaintenanceActionResponse{}, util.NewError(409, "serah terima hanya untuk pengajuan yang sudah diajukan/dijadwalkan", util.ErrConflict)
	}
	vehicle, err := s.q.GetVehicleByID(ctx, m.VehicleID)
	if err != nil {
		return MaintenanceActionResponse{}, util.ErrNotFound
	}
	facts, _ := s.q.GetResourceStatusFacts(ctx, vehicle.ResourceId)
	if facts.HasActiveTrip {
		return MaintenanceActionResponse{}, util.NewError(409,
			"kendaraan sedang dipakai perjalanan - selesaikan booking-nya dulu sebelum diserahkan ke vendor", util.ErrConflict)
	}
	at := time.Now()
	if req.HandoverAt != nil {
		at = *req.HandoverAt
	}
	if at.After(time.Now().Add(10 * time.Minute)) {
		return MaintenanceActionResponse{}, util.NewError(400, "waktu serah terima tidak boleh di masa depan", util.ErrBadRequest)
	}
	if req.Odometer != nil && *req.Odometer < vehicle.CurrentOdometer {
		return MaintenanceActionResponse{}, util.NewError(400,
			fmt.Sprintf("odometer tidak boleh kurang dari catatan kendaraan saat ini (%d km)", vehicle.CurrentOdometer), util.ErrBadRequest)
	}
	fuel, err := validFuel(req.FuelLevel)
	if err != nil {
		return MaintenanceActionResponse{}, err
	}
	list, err := validateChecklist(req.Checklist)
	if err != nil {
		return MaintenanceActionResponse{}, err
	}
	ok, err := s.q.TransitionMaintenance(ctx, id, []string{repository.MaintSubmitted, repository.MaintScheduled},
		repository.MaintInProgress,
		`"handoverAt" = $3, "handoverOdometer" = $4, "handoverFuelLevel" = $5, "handoverReceiverName" = $6,
		 "handoverChecklist" = $7::jsonb, "handoverNote" = $8, "startDate" = $3, "endDate" = NULL,
		 odometer = COALESCE($4, odometer)`,
		at, nullInt32PtrOf(req.Odometer), fuel, strings.TrimSpace(req.ReceiverName), string(list), nullStrOf(req.Note))
	if err != nil {
		return MaintenanceActionResponse{}, err
	}
	if !ok {
		return MaintenanceActionResponse{}, errStatusChanged
	}
	if req.Odometer != nil {
		_, _ = s.q.UpdateVehicleOdometer(ctx, repository.UpdateVehicleOdometerParams{ID: vehicle.ID, CurrentOdometer: *req.Odometer})
	}
	// Kendaraan kini di vendor → MAINTENANCE (hilang dari pilihan booking).
	syncResourceStatus(ctx, s.q, vehicle.ResourceId, false)
	logAudit(ctx, s.q, actor, "HANDOVER", "Maintenance", id,
		"Kendaraan "+vehicle.PlateNumber+" diserahkan ke vendor (diterima "+strings.TrimSpace(req.ReceiverName)+")")
	s.notifyVehicleHolder(ctx, vehicle.ID, "Kendaraan "+vehicle.PlateNumber+" diserahkan ke vendor untuk maintenance", id)
	data, _ := s.GetByID(ctx, id)
	days := m.EstimatedDays
	_, end := planWindow(at, days)
	return MaintenanceActionResponse{Data: data, Warning: s.bookingConflictWarning(ctx, vehicle.ResourceId, at, end)}, nil
}

// Return: IN_PROGRESS → COMPLETED (kendaraan kembali dari vendor).
func (s *MaintenanceService) Return(ctx context.Context, id int32, req ReturnMaintenanceRequest, actor AuditActor) (MaintenanceActionResponse, error) {
	m, err := s.q.GetMaintenanceRecord(ctx, id)
	if err != nil {
		return MaintenanceActionResponse{}, util.ErrNotFound
	}
	if m.Status != repository.MaintInProgress {
		return MaintenanceActionResponse{}, util.NewError(409, "hanya maintenance yang sedang dikerjakan yang bisa diterima kembali", util.ErrConflict)
	}
	at := time.Now()
	if req.ReturnedAt != nil {
		at = *req.ReturnedAt
	}
	if at.After(time.Now().Add(10 * time.Minute)) {
		return MaintenanceActionResponse{}, util.NewError(400, "waktu kembali tidak boleh di masa depan", util.ErrBadRequest)
	}
	if m.HandoverAt.Valid && !at.After(m.HandoverAt.Time) {
		return MaintenanceActionResponse{}, util.NewError(400, "waktu kembali harus setelah waktu serah terima", util.ErrBadRequest)
	}
	vehicle, err := s.q.GetVehicleByID(ctx, m.VehicleID)
	if err != nil {
		return MaintenanceActionResponse{}, util.ErrNotFound
	}
	if req.Odometer != nil {
		min := vehicle.CurrentOdometer
		if m.HandoverOdo.Valid && m.HandoverOdo.Int32 > min {
			min = m.HandoverOdo.Int32
		}
		if *req.Odometer < min {
			return MaintenanceActionResponse{}, util.NewError(400,
				fmt.Sprintf("odometer kembali tidak boleh kurang dari %d km", min), util.ErrBadRequest)
		}
	}
	if req.ActualCost != nil && *req.ActualCost < 0 {
		return MaintenanceActionResponse{}, util.NewError(400, "biaya tidak boleh negatif", util.ErrBadRequest)
	}
	bearer := strings.ToUpper(strings.TrimSpace(req.CostBearer))
	if _, okB := costBearers[bearer]; bearer != "" && !okB {
		return MaintenanceActionResponse{}, util.NewError(400, "penanggung biaya tidak dikenal", util.ErrBadRequest)
	}
	fuel, err := validFuel(req.FuelLevel)
	if err != nil {
		return MaintenanceActionResponse{}, err
	}
	list, err := validateChecklist(req.Checklist)
	if err != nil {
		return MaintenanceActionResponse{}, err
	}
	ok, err := s.q.TransitionMaintenance(ctx, id, []string{repository.MaintInProgress}, repository.MaintCompleted,
		`"returnedAt" = $3, "returnOdometer" = $4, "returnFuelLevel" = $5, "returnHandlerName" = $6,
		 "returnChecklist" = $7::jsonb, "workDone" = $8, "partsReplaced" = $9, "returnNote" = $10,
		 "endDate" = $3, "completedAt" = $3,
		 "totalCost" = COALESCE($11::numeric, "totalCost"), "costBearer" = COALESCE($12, "costBearer")`,
		at, nullInt32PtrOf(req.Odometer), fuel, nullStrOf(req.HandlerName), string(list),
		strings.TrimSpace(req.WorkDone), nullStrOf(req.PartsReplaced), nullStrOf(req.Note),
		nullNumeric(req.ActualCost), nullStrOf(bearer))
	if err != nil {
		return MaintenanceActionResponse{}, err
	}
	if !ok {
		return MaintenanceActionResponse{}, errStatusChanged
	}
	if req.Odometer != nil {
		_, _ = s.q.UpdateVehicleOdometer(ctx, repository.UpdateVehicleOdometerParams{ID: vehicle.ID, CurrentOdometer: *req.Odometer})
	}
	// Kembali dari vendor → AVAILABLE (kecuali INACTIVE).
	syncResourceStatus(ctx, s.q, vehicle.ResourceId, true)
	logAudit(ctx, s.q, actor, "COMPLETE", "Maintenance", id, "Kendaraan "+vehicle.PlateNumber+" kembali dari vendor")
	s.notifyVehicleHolder(ctx, vehicle.ID, "Kendaraan "+vehicle.PlateNumber+" sudah kembali dari maintenance", id)
	data, _ := s.GetByID(ctx, id)
	return MaintenanceActionResponse{Data: data}, nil
}

// Cancel: sebelum kendaraan diserahkan (DRAFT/SUBMITTED/SCHEDULED).
func (s *MaintenanceService) Cancel(ctx context.Context, id int32, req CancelMaintenanceRequest, actor AuditActor) (MaintenanceActionResponse, error) {
	m, err := s.q.GetMaintenanceRecord(ctx, id)
	if err != nil {
		return MaintenanceActionResponse{}, util.ErrNotFound
	}
	if !contains(maintenanceEditable, m.Status) {
		return MaintenanceActionResponse{}, util.NewError(409,
			"maintenance yang sudah dikerjakan/selesai tidak bisa dibatalkan", util.ErrConflict)
	}
	if m.Status != repository.MaintDraft && strings.TrimSpace(req.Reason) == "" {
		return MaintenanceActionResponse{}, util.NewError(400, "alasan pembatalan wajib diisi", util.ErrBadRequest)
	}
	ok, err := s.q.TransitionMaintenance(ctx, id, maintenanceEditable, repository.MaintCancelled,
		`"cancelledAt" = NOW(), "cancelReason" = $3`, nullStrOf(req.Reason))
	if err != nil {
		return MaintenanceActionResponse{}, err
	}
	if !ok {
		return MaintenanceActionResponse{}, errStatusChanged
	}
	logAudit(ctx, s.q, actor, "CANCEL", "Maintenance", id, "Membatalkan maintenance "+m.PlateNumber+": "+req.Reason)
	data, _ := s.GetByID(ctx, id)
	return MaintenanceActionResponse{Data: data}, nil
}

// UpdateCost: biaya bebas diisi kapan saja (keputusan pemilik produk).
func (s *MaintenanceService) UpdateCost(ctx context.Context, id int32, req MaintenanceCostRequest, actor AuditActor) (map[string]any, error) {
	m, err := s.q.GetMaintenanceRecord(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if (req.EstimatedCost != nil && *req.EstimatedCost < 0) || (req.ActualCost != nil && *req.ActualCost < 0) {
		return nil, util.NewError(400, "biaya tidak boleh negatif", util.ErrBadRequest)
	}
	bearer := strings.ToUpper(strings.TrimSpace(req.CostBearer))
	if _, ok := costBearers[bearer]; bearer != "" && !ok {
		return nil, util.NewError(400, "penanggung biaya tidak dikenal", util.ErrBadRequest)
	}
	if err := s.q.UpdateMaintenanceCost(ctx, id, nullNumeric(req.EstimatedCost), nullNumeric(req.ActualCost), nullStrOf(bearer)); err != nil {
		return nil, err
	}
	logAudit(ctx, s.q, actor, "UPDATE_COST", "Maintenance", id, "Mengubah biaya maintenance "+m.PlateNumber)
	return s.GetByID(ctx, id)
}

// Delete: hanya draf atau yang dibatalkan (riwayat pengajuan resmi tidak dihapus).
func (s *MaintenanceService) Delete(ctx context.Context, id int32, actor AuditActor) error {
	m, err := s.q.GetMaintenanceRecord(ctx, id)
	if err != nil {
		return util.ErrNotFound
	}
	ok, err := s.q.DeleteMaintenanceRecord(ctx, id, []string{repository.MaintDraft, repository.MaintCancelled})
	if err != nil {
		return err
	}
	if !ok {
		return util.NewError(409, "hanya draf atau pengajuan yang dibatalkan yang bisa dihapus - batalkan dulu", util.ErrConflict)
	}
	logAudit(ctx, s.q, actor, "DELETE", "Maintenance", id, "Menghapus maintenance kendaraan "+m.PlateNumber)
	return nil
}

// ─── Dokumen ─────────────────────────────────────────────────────────────────

var maintenanceDocKinds = map[string]bool{
	"INVOICE": true, "SIGNED_REQUEST": true, "SIGNED_HANDOVER": true, "SIGNED_RETURN": true, "PHOTO": true, "OTHER": true,
}

func (s *MaintenanceService) UploadDocuments(ctx context.Context, id int32, kind string, files []*multipart.FileHeader, actor AuditActor) ([]repository.MaintenanceDocument, error) {
	if _, err := s.q.GetMaintenanceRecord(ctx, id); err != nil {
		return nil, util.ErrNotFound
	}
	kind = strings.ToUpper(strings.TrimSpace(kind))
	if !maintenanceDocKinds[kind] {
		return nil, util.NewError(400, "jenis dokumen tidak dikenal", util.ErrBadRequest)
	}
	if len(files) == 0 {
		return nil, util.NewError(400, "pilih file yang akan diunggah", util.ErrBadRequest)
	}
	for _, fh := range files {
		path, err := util.SaveUploadedFile(fh, "maintenance")
		if err != nil {
			return nil, util.NewError(400, err.Error(), util.ErrBadRequest)
		}
		if !strings.HasPrefix(path, "/uploads/") {
			path = "/uploads/" + strings.TrimPrefix(path, "/")
		}
		if _, err := s.q.InsertMaintenanceDocument(ctx, repository.MaintenanceDocument{
			MaintenanceID: id, Kind: kind, FileURL: path, FileName: fh.Filename, UploadedByID: actor.UserID,
		}); err != nil {
			return nil, err
		}
	}
	logAudit(ctx, s.q, actor, "UPLOAD_DOCUMENT", "Maintenance", id, fmt.Sprintf("Mengunggah %d dokumen %s", len(files), kind))
	return s.q.ListMaintenanceDocuments(ctx, id)
}

func (s *MaintenanceService) DeleteDocument(ctx context.Context, id, docID int32, actor AuditActor) error {
	d, err := s.q.GetMaintenanceDocument(ctx, docID)
	if err != nil || d.MaintenanceID != id {
		return util.ErrNotFound
	}
	if err := s.q.DeleteMaintenanceDocument(ctx, docID); err != nil {
		return err
	}
	util.DeleteUploadedFile(strings.TrimPrefix(d.FileURL, "/uploads/"))
	logAudit(ctx, s.q, actor, "DELETE_DOCUMENT", "Maintenance", id, "Menghapus dokumen "+d.FileName)
	return nil
}

// notifyVehicleHolder memberi tahu supir yang sedang memegang kendaraan.
func (s *MaintenanceService) notifyVehicleHolder(ctx context.Context, vehicleID int32, msg string, maintenanceID int32) {
	if s.notif == nil {
		return
	}
	holder, err := s.q.GetVehicleCurrentAssignment(ctx, vehicleID)
	if err != nil {
		return
	}
	if drv, derr := s.q.GetDriverByID(ctx, holder.DriverId); derr == nil {
		s.notif.Notify(drv.UserId, "MAINTENANCE_UPDATE", "Maintenance kendaraan", msg,
			map[string]any{"maintenanceId": maintenanceID, "vehicleId": vehicleID})
	}
}

// MaintenanceOptions: pilihan tetap untuk form (kategori, cara serah, dll).
func MaintenanceOptions() map[string]any {
	type opt struct {
		Value string `json:"value"`
		Label string `json:"label"`
	}
	toList := func(m map[string]string, order []string) []opt {
		out := make([]opt, 0, len(order))
		for _, k := range order {
			out = append(out, opt{k, m[k]})
		}
		return out
	}
	checklist := make([]opt, len(handoverChecklistItems))
	for i, it := range handoverChecklistItems {
		checklist[i] = opt{it.Key, it.Label}
	}
	return map[string]any{
		"categories":    toList(maintenanceCategories, []string{"ROUTINE", "REPAIR", "PARTS", "BODY", "OTHER"}),
		"pickupMethods": toList(pickupMethods, []string{"DROP_OFF", "PICKUP"}),
		"costBearers":   toList(costBearers, []string{"COMPANY", "VENDOR", "UNDECIDED"}),
		"fuelLevels":    []string{"E", "1/4", "1/2", "3/4", "F"},
		"checklist":     checklist,
	}
}
