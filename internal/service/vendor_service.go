package service

import (
	"context"
	"database/sql"
	"errors"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

// ─── Vendor / bengkel ────────────────────────────────────────────────────────

type VendorService struct {
	q repository.ExtendedQuerier
}

func NewVendorService(db *sql.DB) *VendorService {
	return &VendorService{q: repository.New(db)}
}

type VendorRequest struct {
	Name    string `json:"name"    validate:"required"`
	Type    string `json:"type"    validate:"required"`
	Address string `json:"address"`
	PicName string `json:"picName"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Note    string `json:"note"`
}

var vendorTypes = map[string]string{"OWNER": "Pemilik kendaraan sewa", "WORKSHOP": "Bengkel rekanan", "BOTH": "Pemilik & bengkel"}

func serializeVendor(v repository.Vendor) map[string]any {
	return map[string]any{
		"id": v.ID, "name": v.Name, "type": v.Type, "typeLabel": vendorTypes[v.Type],
		"address": nullStr(v.Address), "picName": nullStr(v.PicName), "phone": nullStr(v.Phone),
		"email": nullStr(v.Email), "note": nullStr(v.Note), "isActive": v.IsActive,
		"vehicleCount": v.VehicleCount, "maintenanceCount": v.MaintenanceCount,
		"createdAt": v.CreatedAt, "updatedAt": v.UpdatedAt,
	}
}

func (r VendorRequest) params() (repository.VendorParams, error) {
	t := strings.ToUpper(strings.TrimSpace(r.Type))
	if _, ok := vendorTypes[t]; !ok {
		return repository.VendorParams{}, util.NewError(400, "jenis vendor harus OWNER, WORKSHOP, atau BOTH", util.ErrBadRequest)
	}
	name := strings.TrimSpace(r.Name)
	if name == "" {
		return repository.VendorParams{}, util.NewError(400, "nama vendor wajib diisi", util.ErrBadRequest)
	}
	return repository.VendorParams{
		Name: name, Type: t, Address: nullStrOf(r.Address), PicName: nullStrOf(r.PicName),
		Phone: nullStrOf(r.Phone), Email: nullStrOf(r.Email), Note: nullStrOf(r.Note),
	}, nil
}

func duplicateVendor(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return util.NewError(409, "nama vendor sudah dipakai", util.ErrDuplicate)
	}
	return err
}

func (s *VendorService) List(ctx context.Context, search, vType string, active *bool) ([]map[string]any, error) {
	p := repository.ListVendorsParams{Search: search, Type: strings.ToUpper(vType)}
	if active != nil {
		p.IsActive = sql.NullBool{Bool: *active, Valid: true}
	}
	rows, err := s.q.ListVendors(ctx, p)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(rows))
	for i, v := range rows {
		out[i] = serializeVendor(v)
	}
	return out, nil
}

func (s *VendorService) GetByID(ctx context.Context, id int32) (map[string]any, error) {
	v, err := s.q.GetVendor(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	return serializeVendor(v), nil
}

func (s *VendorService) Create(ctx context.Context, req VendorRequest, actor AuditActor) (map[string]any, error) {
	p, err := req.params()
	if err != nil {
		return nil, err
	}
	v, err := s.q.CreateVendor(ctx, p)
	if err != nil {
		return nil, duplicateVendor(err)
	}
	logAudit(ctx, s.q, actor, "CREATE", "Vendor", v.ID, "Menambah vendor "+v.Name)
	return serializeVendor(v), nil
}

func (s *VendorService) Update(ctx context.Context, id int32, req VendorRequest, actor AuditActor) (map[string]any, error) {
	p, err := req.params()
	if err != nil {
		return nil, err
	}
	v, err := s.q.UpdateVendor(ctx, id, p)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, util.ErrNotFound
	}
	if err != nil {
		return nil, duplicateVendor(err)
	}
	logAudit(ctx, s.q, actor, "UPDATE", "Vendor", id, "Mengubah vendor "+v.Name)
	return serializeVendor(v), nil
}

func (s *VendorService) ToggleActive(ctx context.Context, id int32, actor AuditActor) (map[string]any, error) {
	v, err := s.q.GetVendor(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if err := s.q.SetVendorActive(ctx, id, !v.IsActive); err != nil {
		return nil, err
	}
	action := "ACTIVATE"
	if v.IsActive {
		action = "DEACTIVATE"
	}
	logAudit(ctx, s.q, actor, action, "Vendor", id, "Mengubah status aktif vendor "+v.Name)
	return s.GetByID(ctx, id)
}

func (s *VendorService) Delete(ctx context.Context, id int32, actor AuditActor) error {
	v, err := s.q.GetVendor(ctx, id)
	if err != nil {
		return util.ErrNotFound
	}
	if err := s.q.DeleteVendor(ctx, id); err != nil {
		return inUseError(err, "vendor ini sudah dipakai kendaraan/maintenance dan tidak bisa dihapus - nonaktifkan saja")
	}
	logAudit(ctx, s.q, actor, "DELETE", "Vendor", id, "Menghapus vendor "+v.Name)
	return nil
}

// ─── Pengaturan dokumen (kop surat & penandatangan) ─────────────────────────

type DocumentSettingsService struct {
	q repository.ExtendedQuerier
}

func NewDocumentSettingsService(db *sql.DB) *DocumentSettingsService {
	return &DocumentSettingsService{q: repository.New(db)}
}

type DocumentSettingsRequest struct {
	CompanyName    string `json:"companyName"`
	CompanyAddress string `json:"companyAddress"`
	CompanyPhone   string `json:"companyPhone"`
	CompanyEmail   string `json:"companyEmail"`
	SignerName     string `json:"signerName"`
	SignerTitle    string `json:"signerTitle"`
	LetterCode     string `json:"letterCode"`
}

func serializeDocSettings(s repository.DocumentSettings) map[string]any {
	return map[string]any{
		"companyName": s.CompanyName, "companyAddress": s.CompanyAddress, "companyPhone": s.CompanyPhone,
		"companyEmail": s.CompanyEmail, "logoUrl": nullStr(s.LogoURL), "signerName": s.SignerName,
		"signerTitle": s.SignerTitle, "letterCode": s.LetterCode, "updatedAt": s.UpdatedAt,
	}
}

func (s *DocumentSettingsService) Get(ctx context.Context) (map[string]any, error) {
	st, err := s.q.GetDocumentSettings(ctx)
	if err != nil {
		return nil, err
	}
	return serializeDocSettings(st), nil
}

func (s *DocumentSettingsService) Update(ctx context.Context, req DocumentSettingsRequest, actor AuditActor) (map[string]any, error) {
	code := strings.TrimSpace(req.LetterCode)
	if code == "" {
		code = "KCE-MNT"
	}
	if strings.ContainsAny(code, "/ ") || len(code) > 30 {
		return nil, util.NewError(400, "kode surat maks. 30 karakter tanpa spasi atau '/'", util.ErrBadRequest)
	}
	cur, _ := s.q.GetDocumentSettings(ctx)
	st := repository.DocumentSettings{
		CompanyName: strings.TrimSpace(req.CompanyName), CompanyAddress: strings.TrimSpace(req.CompanyAddress),
		CompanyPhone: strings.TrimSpace(req.CompanyPhone), CompanyEmail: strings.TrimSpace(req.CompanyEmail),
		SignerName: strings.TrimSpace(req.SignerName), SignerTitle: strings.TrimSpace(req.SignerTitle),
		LetterCode: code, LogoURL: cur.LogoURL,
	}
	if err := s.q.UpdateDocumentSettings(ctx, st); err != nil {
		return nil, err
	}
	logAudit(ctx, s.q, actor, "UPDATE", "DocumentSettings", 1, "Mengubah pengaturan kop surat & penandatangan")
	return s.Get(ctx)
}

// SetLogo: logo kop surat (PNG/JPG — format yang didukung pembuat PDF).
func (s *DocumentSettingsService) SetLogo(ctx context.Context, fh *multipart.FileHeader, actor AuditActor) (map[string]any, error) {
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
		return nil, util.NewError(400, "logo harus berformat PNG atau JPG", util.ErrBadRequest)
	}
	path, err := util.SaveUploadedFile(fh, "settings")
	if err != nil {
		return nil, util.NewError(400, err.Error(), util.ErrBadRequest)
	}
	url := "/uploads/" + strings.TrimPrefix(path, "/")
	cur, _ := s.q.GetDocumentSettings(ctx)
	if err := s.q.SetDocumentLogo(ctx, sql.NullString{String: url, Valid: true}); err != nil {
		return nil, err
	}
	if cur.LogoURL.Valid {
		util.DeleteUploadedFile(strings.TrimPrefix(cur.LogoURL.String, "/uploads/"))
	}
	logAudit(ctx, s.q, actor, "UPDATE", "DocumentSettings", 1, "Mengganti logo kop surat")
	return s.Get(ctx)
}

func (s *DocumentSettingsService) DeleteLogo(ctx context.Context, actor AuditActor) (map[string]any, error) {
	cur, _ := s.q.GetDocumentSettings(ctx)
	if err := s.q.SetDocumentLogo(ctx, sql.NullString{}); err != nil {
		return nil, err
	}
	if cur.LogoURL.Valid {
		util.DeleteUploadedFile(strings.TrimPrefix(cur.LogoURL.String, "/uploads/"))
	}
	logAudit(ctx, s.q, actor, "UPDATE", "DocumentSettings", 1, "Menghapus logo kop surat")
	return s.Get(ctx)
}
