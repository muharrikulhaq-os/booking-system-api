package service

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

func TestFormatLetterNo(t *testing.T) {
	// 30 Sep 2026 20.00 UTC = 1 Okt 2026 03.00 WIB → bulan X.
	at := time.Date(2026, 9, 30, 20, 0, 0, 0, time.UTC)
	if got := formatLetterNo(12, "KCE-MNT", at); got != "012/KCE-MNT/X/2026" {
		t.Errorf("got %q", got)
	}
	if got := formatLetterNo(7, " ", at); got != "007/KCE-MNT/X/2026" {
		t.Errorf("kode kosong harus jatuh ke default, got %q", got)
	}
}

func TestPlanWindow(t *testing.T) {
	start := time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC)
	s, e := planWindow(start, sql.NullInt32{Int32: 3, Valid: true})
	if !s.Equal(start) || !e.Valid || !e.Time.Equal(start.Add(72*time.Hour)) {
		t.Errorf("3 hari: %v %v", s, e)
	}
	_, e = planWindow(start, sql.NullInt32{})
	if !e.Time.Equal(start.Add(24 * time.Hour)) {
		t.Errorf("default 1 hari: %v", e)
	}
}

func TestValidFuelAndChecklist(t *testing.T) {
	if v, err := validFuel(" 1/2 "); err != nil || v.String != "1/2" {
		t.Errorf("1/2 harus valid: %v %v", v, err)
	}
	if _, err := validFuel("penuh"); !errors.Is(err, util.ErrBadRequest) {
		t.Errorf("level tak dikenal harus 400: %v", err)
	}
	if v, err := validFuel(""); err != nil || v.Valid {
		t.Errorf("kosong = tidak diisi: %v %v", v, err)
	}
	if _, err := validateChecklist(map[string]bool{"stnk": true, "spareKey": false}); err != nil {
		t.Errorf("checklist sah ditolak: %v", err)
	}
	if _, err := validateChecklist(map[string]bool{"tv": true}); !errors.Is(err, util.ErrBadRequest) {
		t.Errorf("item tak dikenal harus 400: %v", err)
	}
}

func TestRupiah(t *testing.T) {
	cases := map[float64]string{0: "Rp 0", 950: "Rp 950", 1500000: "Rp 1.500.000", 12345678.4: "Rp 12.345.678"}
	for in, want := range cases {
		if got := rupiah(in); got != want {
			t.Errorf("rupiah(%v) = %q, want %q", in, got, want)
		}
	}
	if rupiah(nil) != "-" {
		t.Error("nil harus '-'")
	}
}

type pdfFake struct {
	repository.ExtendedQuerier
	m repository.MaintenanceItem
}

func (f *pdfFake) GetMaintenanceRecord(ctx context.Context, id int32) (repository.MaintenanceItem, error) {
	return f.m, nil
}

func (f *pdfFake) GetDocumentSettings(ctx context.Context) (repository.DocumentSettings, error) {
	return repository.DocumentSettings{
		CompanyName: "PT Kereta Cepat Contoh", CompanyAddress: "Jl. Contoh No. 1, Jakarta",
		CompanyPhone: "021-000000", SignerName: "Budi Santoso", SignerTitle: "Kepala Bagian Umum", LetterCode: "KCE-MNT",
	}, nil
}

func TestGeneratePDF(t *testing.T) {
	now := time.Now()
	m := repository.MaintenanceItem{
		ID: 1, VehicleID: 2, VehicleName: "Innova Operasional", PlateNumber: "B 1234 KCE", Brand: "Toyota",
		Model: "Innova", Year: 2023, CurrentOdometer: 45210, Ownership: "VENDOR",
		OwnerVendorName: sql.NullString{String: "PT Sewa Mobil", Valid: true},
		VendorID:        sql.NullInt32{Int32: 3, Valid: true}, VendorName: sql.NullString{String: "PT Sewa Mobil", Valid: true},
		VendorAddress: sql.NullString{String: "Jl. Bengkel Raya 2", Valid: true},
		RequestNo:     sql.NullString{String: "001/KCE-MNT/X/2026", Valid: true},
		Category:      sql.NullString{String: "REPAIR", Valid: true}, Status: repository.MaintInProgress,
		Description: "Ganti kampas rem depan — bunyi mendecit", PlannedDate: sql.NullTime{Time: now, Valid: true},
		EstimatedDays: sql.NullInt32{Int32: 2, Valid: true}, PickupMethod: sql.NullString{String: "PICKUP", Valid: true},
		EstimatedCost: sql.NullString{String: "1500000.00", Valid: true}, SubmittedAt: sql.NullTime{Time: now, Valid: true},
		HandoverAt: sql.NullTime{Time: now, Valid: true}, HandoverOdo: sql.NullInt32{Int32: 45210, Valid: true},
		HandoverFuel: sql.NullString{String: "1/2", Valid: true}, HandoverBy: sql.NullString{String: "Andi (bengkel)", Valid: true},
		HandoverList: []byte(`{"stnk":true,"mainKey":true}`),
	}
	s := &MaintenanceService{q: &pdfFake{m: m}}
	for _, kind := range []string{PDFRequest, PDFHandover} {
		data, name, err := s.GeneratePDF(context.Background(), 1, kind)
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		if !bytes.HasPrefix(data, []byte("%PDF")) || len(data) < 1000 || name == "" {
			t.Errorf("%s: PDF tidak valid (%d byte, nama %q)", kind, len(data), name)
		}
	}
	// Berita acara pengembalian belum tersedia sebelum kendaraan kembali.
	if _, _, err := s.GeneratePDF(context.Background(), 1, PDFReturn); !errors.Is(err, util.ErrConflict) {
		t.Errorf("return sebelum kembali harus 409: %v", err)
	}
}
