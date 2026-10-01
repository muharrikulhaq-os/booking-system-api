package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Status maintenance (docs/RANCANGAN_MAINTENANCE_VENDOR.md §4).
const (
	MaintDraft      = "DRAFT"
	MaintSubmitted  = "SUBMITTED"
	MaintScheduled  = "SCHEDULED"
	MaintInProgress = "IN_PROGRESS"
	MaintCompleted  = "COMPLETED"
	MaintCancelled  = "CANCELLED"
)

// MaintenanceItem: satu pengajuan maintenance lengkap + nama-nama terkait.
type MaintenanceItem struct {
	ID              int32
	VehicleID       int32
	VehicleName     string
	PlateNumber     string
	VehiclePhotoURL sql.NullString
	Brand           string
	Model           string
	Year            int16
	CurrentOdometer int32
	Ownership       string
	OwnerVendorName sql.NullString
	RentalContract  sql.NullString
	VendorID        sql.NullInt32
	VendorName      sql.NullString
	VendorLegacy    sql.NullString // kolom lama "vendorName" (data sebelum master vendor)
	VendorAddress   sql.NullString
	VendorPic       sql.NullString
	VendorPhone     sql.NullString
	RequestNo       sql.NullString
	Category        sql.NullString
	Type            string
	Status          string
	Description     string
	Complaint       sql.NullString
	Location        sql.NullString
	PlannedDate     sql.NullTime
	EstimatedDays   sql.NullInt32
	ScheduledDate   sql.NullTime
	ScheduleNote    sql.NullString
	PickupMethod    sql.NullString
	EstimatedCost   sql.NullString
	TotalCost       sql.NullString // biaya aktual
	CostBearer      sql.NullString
	Odometer        sql.NullInt32
	StartDate       time.Time
	EndDate         sql.NullTime
	SubmittedAt     sql.NullTime
	HandoverAt      sql.NullTime
	HandoverOdo     sql.NullInt32
	HandoverFuel    sql.NullString
	HandoverBy      sql.NullString
	HandoverList    []byte
	HandoverNote    sql.NullString
	ReturnedAt      sql.NullTime
	ReturnOdo       sql.NullInt32
	ReturnFuel      sql.NullString
	ReturnBy        sql.NullString
	ReturnList      []byte
	WorkDone        sql.NullString
	PartsReplaced   sql.NullString
	ReturnNote      sql.NullString
	CompletedAt     sql.NullTime
	CancelledAt     sql.NullTime
	CancelReason    sql.NullString
	ProofPhotos     []byte
	SourceIssueID   sql.NullInt32
	RecordedByID    int32
	RecordedByName  string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

const maintenanceSelect = `
	SELECT m.id, m."vehicleId", r.name, v."plateNumber", v."photoUrl", v.brand, v.model, v.year,
	       v."currentOdometer", v.ownership, ov.name, v."rentalContractNo",
	       m."vendorId", vd.name, m."vendorName", vd.address, vd."picName", vd.phone,
	       m."requestNo", m.category, m.type, m.status, m.description, m.complaint, m.location,
	       m."plannedDate", m."estimatedDays", m."scheduledDate", m."scheduleNote", m."pickupMethod",
	       m."estimatedCost"::text, m."totalCost"::text, m."costBearer", m.odometer,
	       m."startDate", m."endDate", m."submittedAt",
	       m."handoverAt", m."handoverOdometer", m."handoverFuelLevel", m."handoverReceiverName",
	       m."handoverChecklist", m."handoverNote",
	       m."returnedAt", m."returnOdometer", m."returnFuelLevel", m."returnHandlerName",
	       m."returnChecklist", m."workDone", m."partsReplaced", m."returnNote",
	       m."completedAt", m."cancelledAt", m."cancelReason", m."proofPhotos", m."sourceIssueId",
	       m."recordedById", u.name, m."createdAt", m."updatedAt"
	FROM maintenance_records m
	JOIN vehicles v ON v.id = m."vehicleId"
	JOIN resources r ON r.id = v."resourceId"
	JOIN users u ON u.id = m."recordedById"
	LEFT JOIN vendors vd ON vd.id = m."vendorId"
	LEFT JOIN vendors ov ON ov.id = v."ownerVendorId"`

func scanMaintenance(row interface{ Scan(...any) error }) (MaintenanceItem, error) {
	var m MaintenanceItem
	err := row.Scan(&m.ID, &m.VehicleID, &m.VehicleName, &m.PlateNumber, &m.VehiclePhotoURL, &m.Brand, &m.Model, &m.Year,
		&m.CurrentOdometer, &m.Ownership, &m.OwnerVendorName, &m.RentalContract,
		&m.VendorID, &m.VendorName, &m.VendorLegacy, &m.VendorAddress, &m.VendorPic, &m.VendorPhone,
		&m.RequestNo, &m.Category, &m.Type, &m.Status, &m.Description, &m.Complaint, &m.Location,
		&m.PlannedDate, &m.EstimatedDays, &m.ScheduledDate, &m.ScheduleNote, &m.PickupMethod,
		&m.EstimatedCost, &m.TotalCost, &m.CostBearer, &m.Odometer,
		&m.StartDate, &m.EndDate, &m.SubmittedAt,
		&m.HandoverAt, &m.HandoverOdo, &m.HandoverFuel, &m.HandoverBy, &m.HandoverList, &m.HandoverNote,
		&m.ReturnedAt, &m.ReturnOdo, &m.ReturnFuel, &m.ReturnBy, &m.ReturnList, &m.WorkDone, &m.PartsReplaced, &m.ReturnNote,
		&m.CompletedAt, &m.CancelledAt, &m.CancelReason, &m.ProofPhotos, &m.SourceIssueID,
		&m.RecordedByID, &m.RecordedByName, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}

type ListMaintenanceRecordsParams struct {
	VehicleID sql.NullInt32
	VendorID  sql.NullInt32
	Statuses  []string // kosong = semua
	Search    string   // nomor surat, plat, nama kendaraan, vendor
	SortBy    string
	SortOrder string
	Limit     int32
	Offset    int32
}

func (p ListMaintenanceRecordsParams) where() (string, []any) {
	args := []any{p.VehicleID, p.VendorID, strings.TrimSpace(p.Search)}
	w := `WHERE ($1::int IS NULL OR m."vehicleId" = $1::int)
	  AND ($2::int IS NULL OR m."vendorId" = $2::int)
	  AND ($3 = '' OR m."requestNo" ILIKE '%' || $3 || '%' OR v."plateNumber" ILIKE '%' || $3 || '%'
	       OR r.name ILIKE '%' || $3 || '%' OR vd.name ILIKE '%' || $3 || '%' OR m."vendorName" ILIKE '%' || $3 || '%')`
	if len(p.Statuses) > 0 {
		args = append(args, p.Statuses)
		w += fmt.Sprintf(` AND m.status = ANY($%d::text[])`, len(args))
	}
	return w, args
}

var maintenanceSortColumns = map[string]string{
	"createdAt":     `m."createdAt"`,
	"startDate":     `m."startDate"`,
	"scheduledDate": `m."scheduledDate"`,
	"requestNo":     `m."requestNo"`,
	"status":        `m.status`,
	"totalCost":     `m."totalCost"`,
	"plateNumber":   `v."plateNumber"`,
}

func (q *Queries) ListMaintenanceRecords(ctx context.Context, p ListMaintenanceRecordsParams) ([]MaintenanceItem, int64, error) {
	w, args := p.where()
	from := `FROM maintenance_records m
	JOIN vehicles v ON v.id = m."vehicleId"
	JOIN resources r ON r.id = v."resourceId"
	LEFT JOIN vendors vd ON vd.id = m."vendorId" `
	var total int64
	if err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) `+from+w, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	col, ok := maintenanceSortColumns[p.SortBy]
	if !ok {
		col = `m."createdAt"`
	}
	dir := "DESC"
	if strings.EqualFold(p.SortOrder, "asc") {
		dir = "ASC"
	}
	args = append(args, p.Limit, p.Offset)
	rows, err := q.db.QueryContext(ctx, maintenanceSelect+" "+w+
		fmt.Sprintf(` ORDER BY %s %s NULLS LAST, m.id DESC LIMIT $%d OFFSET $%d`, col, dir, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []MaintenanceItem{}
	for rows.Next() {
		m, err := scanMaintenance(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, m)
	}
	return out, total, rows.Err()
}

func (q *Queries) GetMaintenanceRecord(ctx context.Context, id int32) (MaintenanceItem, error) {
	return scanMaintenance(q.db.QueryRowContext(ctx, maintenanceSelect+` WHERE m.id = $1`, id))
}

// GetOpenMaintenanceID: maintenance yang belum final (bukan COMPLETED/CANCELLED)
// milik kendaraan ini — satu kendaraan hanya boleh punya satu proses berjalan.
func (q *Queries) GetOpenMaintenanceID(ctx context.Context, vehicleID, excludeID int32) (int32, error) {
	var id int32
	err := q.db.QueryRowContext(ctx, `
		SELECT id FROM maintenance_records
		WHERE "vehicleId" = $1 AND status NOT IN ('COMPLETED', 'CANCELLED') AND id <> $2
		ORDER BY id LIMIT 1`, vehicleID, excludeID).Scan(&id)
	return id, err
}

// MaintenancePlan: isian pengajuan (dapat diubah selama DRAFT/SUBMITTED/SCHEDULED).
type MaintenancePlan struct {
	VehicleID     int32
	VendorID      sql.NullInt32
	Category      string
	Description   string
	Complaint     sql.NullString
	Location      sql.NullString
	PlannedDate   sql.NullTime
	EstimatedDays sql.NullInt32
	PickupMethod  sql.NullString
	EstimatedCost sql.NullString
	CostBearer    sql.NullString
	Odometer      sql.NullInt32
	// Jendela blokir booking (dijaga service).
	StartDate time.Time
	EndDate   sql.NullTime
}

func (q *Queries) InsertMaintenance(ctx context.Context, p MaintenancePlan, sourceIssue sql.NullInt32, recordedBy int32) (int32, error) {
	var id int32
	err := q.db.QueryRowContext(ctx, `
		INSERT INTO maintenance_records ("vehicleId", "vendorId", category, type, status, description, complaint,
		       location, "plannedDate", "estimatedDays", "pickupMethod", "estimatedCost", "costBearer", odometer,
		       "startDate", "endDate", "sourceIssueId", "recordedById", "isAutoGenerated")
		VALUES ($1, $2, $3, $3, 'DRAFT', $4, $5, $6, $7, $8, $9, $10::numeric, $11, $12, $13, $14, $15, $16, FALSE)
		RETURNING id`,
		p.VehicleID, p.VendorID, p.Category, p.Description, p.Complaint, p.Location, p.PlannedDate, p.EstimatedDays,
		p.PickupMethod, p.EstimatedCost, p.CostBearer, p.Odometer, p.StartDate, p.EndDate, sourceIssue, recordedBy).Scan(&id)
	return id, err
}

// UpdateMaintenancePlan hanya berlaku bila status masih salah satu allowed.
func (q *Queries) UpdateMaintenancePlan(ctx context.Context, id int32, p MaintenancePlan, allowed []string) (bool, error) {
	res, err := q.db.ExecContext(ctx, `
		UPDATE maintenance_records SET "vehicleId" = $2, "vendorId" = $3, category = $4, type = $4,
		       description = $5, complaint = $6, location = $7, "plannedDate" = $8, "estimatedDays" = $9,
		       "pickupMethod" = $10, "estimatedCost" = $11::numeric, "costBearer" = $12, odometer = $13,
		       "startDate" = $14, "endDate" = $15, "updatedAt" = NOW()
		WHERE id = $1 AND status = ANY($16::text[])`,
		id, p.VehicleID, p.VendorID, p.Category, p.Description, p.Complaint, p.Location, p.PlannedDate,
		p.EstimatedDays, p.PickupMethod, p.EstimatedCost, p.CostBearer, p.Odometer, p.StartDate, p.EndDate, allowed)
	return affected(res, err)
}

// TransitionMaintenance menjalankan UPDATE bersyarat status (aman dari klik
// ganda / dua admin). set = potongan SET tambahan dengan parameter mulai $3.
func (q *Queries) TransitionMaintenance(ctx context.Context, id int32, from []string, to string, set string, args ...any) (bool, error) {
	all := append([]any{id, to}, args...)
	all = append(all, from)
	sqlSet := `status = $2, "updatedAt" = NOW()`
	if set != "" {
		sqlSet += ", " + set
	}
	res, err := q.db.ExecContext(ctx, fmt.Sprintf(
		`UPDATE maintenance_records SET %s WHERE id = $1 AND status = ANY($%d::text[])`, sqlSet, len(all)), all...)
	return affected(res, err)
}

// UpdateMaintenanceCost: biaya bebas diisi kapan saja (keputusan pemilik produk).
func (q *Queries) UpdateMaintenanceCost(ctx context.Context, id int32, estimated, actual, bearer sql.NullString) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE maintenance_records SET "estimatedCost" = $2::numeric, "totalCost" = $3::numeric,
		       "costBearer" = $4, "updatedAt" = NOW() WHERE id = $1`, id, estimated, actual, bearer)
	return err
}

func (q *Queries) DeleteMaintenanceRecord(ctx context.Context, id int32, allowed []string) (bool, error) {
	res, err := q.db.ExecContext(ctx, `DELETE FROM maintenance_records WHERE id = $1 AND status = ANY($2::text[])`, id, allowed)
	return affected(res, err)
}

func affected(res sql.Result, err error) (bool, error) {
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}

// ─── Penomoran surat ─────────────────────────────────────────────────────────

// NextDocumentNumber menaikkan penghitung key secara atomik & mengembalikan nilai baru.
func (q *Queries) NextDocumentNumber(ctx context.Context, key string) (int, error) {
	var n int
	err := q.db.QueryRowContext(ctx, `
		INSERT INTO document_counters (key, value) VALUES ($1, 1)
		ON CONFLICT (key) DO UPDATE SET value = document_counters.value + 1
		RETURNING value`, key).Scan(&n)
	return n, err
}

// ─── Pengaturan dokumen ──────────────────────────────────────────────────────

type DocumentSettings struct {
	CompanyName    string         `json:"companyName"`
	CompanyAddress string         `json:"companyAddress"`
	CompanyPhone   string         `json:"companyPhone"`
	CompanyEmail   string         `json:"companyEmail"`
	LogoURL        sql.NullString `json:"logoUrl"`
	SignerName     string         `json:"signerName"`
	SignerTitle    string         `json:"signerTitle"`
	LetterCode     string         `json:"letterCode"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

func (q *Queries) GetDocumentSettings(ctx context.Context) (DocumentSettings, error) {
	var s DocumentSettings
	err := q.db.QueryRowContext(ctx, `
		SELECT "companyName", "companyAddress", "companyPhone", "companyEmail", "logoUrl",
		       "signerName", "signerTitle", "letterCode", "updatedAt"
		FROM document_settings WHERE id = 1`).Scan(&s.CompanyName, &s.CompanyAddress, &s.CompanyPhone,
		&s.CompanyEmail, &s.LogoURL, &s.SignerName, &s.SignerTitle, &s.LetterCode, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return DocumentSettings{LetterCode: "KCE-MNT"}, nil
	}
	return s, err
}

func (q *Queries) UpdateDocumentSettings(ctx context.Context, s DocumentSettings) error {
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO document_settings (id, "companyName", "companyAddress", "companyPhone", "companyEmail",
		       "signerName", "signerTitle", "letterCode", "updatedAt")
		VALUES (1, $1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (id) DO UPDATE SET "companyName" = $1, "companyAddress" = $2, "companyPhone" = $3,
		       "companyEmail" = $4, "signerName" = $5, "signerTitle" = $6, "letterCode" = $7, "updatedAt" = NOW()`,
		s.CompanyName, s.CompanyAddress, s.CompanyPhone, s.CompanyEmail, s.SignerName, s.SignerTitle, s.LetterCode)
	return err
}

func (q *Queries) SetDocumentLogo(ctx context.Context, url sql.NullString) error {
	_, err := q.db.ExecContext(ctx, `
		INSERT INTO document_settings (id, "logoUrl") VALUES (1, $1)
		ON CONFLICT (id) DO UPDATE SET "logoUrl" = $1, "updatedAt" = NOW()`, url)
	return err
}

// ─── Dokumen maintenance ─────────────────────────────────────────────────────

type MaintenanceDocument struct {
	ID            int32     `json:"id"`
	MaintenanceID int32     `json:"maintenanceId"`
	Kind          string    `json:"kind"`
	FileURL       string    `json:"fileUrl"`
	FileName      string    `json:"fileName"`
	UploadedByID  int32     `json:"uploadedById"`
	UploadedBy    string    `json:"uploadedBy"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (q *Queries) ListMaintenanceDocuments(ctx context.Context, maintenanceID int32) ([]MaintenanceDocument, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT d.id, d."maintenanceId", d.kind, d."fileUrl", d."fileName", d."uploadedById", u.name, d."createdAt"
		FROM maintenance_documents d JOIN users u ON u.id = d."uploadedById"
		WHERE d."maintenanceId" = $1 ORDER BY d."createdAt"`, maintenanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MaintenanceDocument{}
	for rows.Next() {
		var d MaintenanceDocument
		if err := rows.Scan(&d.ID, &d.MaintenanceID, &d.Kind, &d.FileURL, &d.FileName, &d.UploadedByID, &d.UploadedBy, &d.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (q *Queries) InsertMaintenanceDocument(ctx context.Context, d MaintenanceDocument) (int32, error) {
	var id int32
	err := q.db.QueryRowContext(ctx, `
		INSERT INTO maintenance_documents ("maintenanceId", kind, "fileUrl", "fileName", "uploadedById")
		VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		d.MaintenanceID, d.Kind, d.FileURL, d.FileName, d.UploadedByID).Scan(&id)
	return id, err
}

func (q *Queries) GetMaintenanceDocument(ctx context.Context, id int32) (MaintenanceDocument, error) {
	var d MaintenanceDocument
	err := q.db.QueryRowContext(ctx, `
		SELECT d.id, d."maintenanceId", d.kind, d."fileUrl", d."fileName", d."uploadedById", u.name, d."createdAt"
		FROM maintenance_documents d JOIN users u ON u.id = d."uploadedById" WHERE d.id = $1`, id).
		Scan(&d.ID, &d.MaintenanceID, &d.Kind, &d.FileURL, &d.FileName, &d.UploadedByID, &d.UploadedBy, &d.CreatedAt)
	return d, err
}

func (q *Queries) DeleteMaintenanceDocument(ctx context.Context, id int32) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM maintenance_documents WHERE id = $1`, id)
	return err
}

// ─── Laporan kerusakan ───────────────────────────────────────────────────────

type VehicleIssue struct {
	ID             int32
	VehicleID      int32
	VehicleName    string
	PlateNumber    string
	BookingID      sql.NullInt32
	ReportedByID   int32
	ReportedByName string
	Description    string
	Location       sql.NullString
	Photos         []byte
	CanContinue    bool
	Status         string
	HandledByID    sql.NullInt32
	HandledByName  sql.NullString
	HandledNote    sql.NullString
	HandledAt      sql.NullTime
	MaintenanceID  sql.NullInt32
	CreatedAt      time.Time
}

const issueSelect = `
	SELECT i.id, i."vehicleId", r.name, v."plateNumber", i."bookingId", i."reportedById", ru.name,
	       i.description, i.location, i.photos, i."canContinue", i.status, i."handledById", hu.name,
	       i."handledNote", i."handledAt", i."maintenanceId", i."createdAt"
	FROM vehicle_issue_reports i
	JOIN vehicles v ON v.id = i."vehicleId"
	JOIN resources r ON r.id = v."resourceId"
	JOIN users ru ON ru.id = i."reportedById"
	LEFT JOIN users hu ON hu.id = i."handledById"`

func scanIssue(row interface{ Scan(...any) error }) (VehicleIssue, error) {
	var i VehicleIssue
	err := row.Scan(&i.ID, &i.VehicleID, &i.VehicleName, &i.PlateNumber, &i.BookingID, &i.ReportedByID, &i.ReportedByName,
		&i.Description, &i.Location, &i.Photos, &i.CanContinue, &i.Status, &i.HandledByID, &i.HandledByName,
		&i.HandledNote, &i.HandledAt, &i.MaintenanceID, &i.CreatedAt)
	return i, err
}

type ListVehicleIssuesParams struct {
	Status     string
	VehicleID  sql.NullInt32
	ReportedBy sql.NullInt32 // supir hanya melihat laporannya sendiri
	Limit      int32
	Offset     int32
}

func (q *Queries) ListVehicleIssues(ctx context.Context, p ListVehicleIssuesParams) ([]VehicleIssue, int64, error) {
	w := ` WHERE ($1 = '' OR i.status = $1) AND ($2::int IS NULL OR i."vehicleId" = $2::int)
	       AND ($3::int IS NULL OR i."reportedById" = $3::int)`
	args := []any{p.Status, p.VehicleID, p.ReportedBy}
	var total int64
	if err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vehicle_issue_reports i`+w, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := q.db.QueryContext(ctx, issueSelect+w+` ORDER BY (i.status = 'OPEN') DESC, i."createdAt" DESC LIMIT $4 OFFSET $5`,
		append(args, p.Limit, p.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []VehicleIssue{}
	for rows.Next() {
		i, err := scanIssue(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, i)
	}
	return out, total, rows.Err()
}

func (q *Queries) GetVehicleIssue(ctx context.Context, id int32) (VehicleIssue, error) {
	return scanIssue(q.db.QueryRowContext(ctx, issueSelect+` WHERE i.id = $1`, id))
}

func (q *Queries) InsertVehicleIssue(ctx context.Context, vehicleID int32, bookingID sql.NullInt32, reportedBy int32,
	description string, location sql.NullString, photos []string, canContinue bool) (int32, error) {
	pj, _ := json.Marshal(photos)
	var id int32
	err := q.db.QueryRowContext(ctx, `
		INSERT INTO vehicle_issue_reports ("vehicleId", "bookingId", "reportedById", description, location, photos, "canContinue")
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7) RETURNING id`,
		vehicleID, bookingID, reportedBy, description, location, string(pj), canContinue).Scan(&id)
	return id, err
}

// ResolveVehicleIssue menutup laporan OPEN (CONVERTED/DISMISSED) secara bersyarat.
func (q *Queries) ResolveVehicleIssue(ctx context.Context, id int32, status string, handledBy int32, note sql.NullString, maintenanceID sql.NullInt32) (bool, error) {
	res, err := q.db.ExecContext(ctx, `
		UPDATE vehicle_issue_reports SET status = $2, "handledById" = $3, "handledNote" = $4,
		       "maintenanceId" = $5, "handledAt" = NOW()
		WHERE id = $1 AND status = 'OPEN'`, id, status, handledBy, note, maintenanceID)
	return affected(res, err)
}

func (q *Queries) CountOpenVehicleIssues(ctx context.Context) (int64, error) {
	var n int64
	err := q.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM vehicle_issue_reports WHERE status = 'OPEN'`).Scan(&n)
	return n, err
}

// DriverVehicle: kendaraan yang boleh dilaporkan kendalanya oleh supir —
// yang sedang/akan dipakainya di booking, sedang dipegangnya, atau supir tetapnya.
type DriverVehicle struct {
	VehicleID   int32         `json:"vehicleId"`
	Name        string        `json:"name"`
	PlateNumber string        `json:"plateNumber"`
	BookingID   sql.NullInt32 `json:"bookingId"`
	OnTrip      bool          `json:"onTrip"`
}

// MarshalJSON: bookingId sebagai angka/null (sql.NullInt32 bawaan
// terserialisasi {"Int32":0,"Valid":false}).
func (d DriverVehicle) MarshalJSON() ([]byte, error) {
	var bid *int32
	if d.BookingID.Valid {
		bid = &d.BookingID.Int32
	}
	return json.Marshal(struct {
		VehicleID   int32  `json:"vehicleId"`
		Name        string `json:"name"`
		PlateNumber string `json:"plateNumber"`
		BookingID   *int32 `json:"bookingId"`
		OnTrip      bool   `json:"onTrip"`
	}{d.VehicleID, d.Name, d.PlateNumber, bid, d.OnTrip})
}

func (q *Queries) ListDriverVehicles(ctx context.Context, driverID int32) ([]DriverVehicle, error) {
	rows, err := q.db.QueryContext(ctx, `
		WITH linked AS (
		    SELECT b."assignedVehicleId" AS vid, b.id AS bid, b.status IN ('ONGOING', 'OVERDUE') AS trip, b."startDate" AS sd
		    FROM bookings b
		    WHERE b."assignedDriverId" = $1 AND b."assignedVehicleId" IS NOT NULL
		      AND b.status IN ('ONGOING', 'OVERDUE', 'APPROVED')
		    UNION ALL
		    SELECT da."vehicleId", NULL, FALSE, NULL FROM driver_assignments da
		    WHERE da."driverId" = $1 AND da."releasedAt" IS NULL
		    UNION ALL
		    SELECT x.id, NULL, FALSE, NULL FROM vehicles x WHERE x."fixedDriverId" = $1
		)
		SELECT DISTINCT ON (l.vid) l.vid, r.name, v."plateNumber", l.bid, l.trip
		FROM linked l JOIN vehicles v ON v.id = l.vid JOIN resources r ON r.id = v."resourceId"
		ORDER BY l.vid, l.trip DESC, (l.bid IS NULL), l.sd`, driverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []DriverVehicle{}
	for rows.Next() {
		var d DriverVehicle
		if err := rows.Scan(&d.VehicleID, &d.Name, &d.PlateNumber, &d.BookingID, &d.OnTrip); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
