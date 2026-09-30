package service

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"booking-system-api/internal/repository"
)

// Pratinjau lokal: PDF_PREVIEW_DIR=<dir> menulis ketiga PDF ke folder itu.
func TestPDFPreview(t *testing.T) {
	dir := os.Getenv("PDF_PREVIEW_DIR")
	if dir == "" {
		t.Skip("set PDF_PREVIEW_DIR untuk menulis pratinjau")
	}
	now := time.Now()
	m := repository.MaintenanceItem{
		ID: 1, VehicleID: 2, VehicleName: "Innova Operasional", PlateNumber: "B 1234 KCE", Brand: "Toyota",
		Model: "Kijang Innova", Year: 2023, CurrentOdometer: 45210, Ownership: "VENDOR",
		OwnerVendorName: sql.NullString{String: "PT Sewa Mobil Nusantara", Valid: true},
		RentalContract:  sql.NullString{String: "SWA/2025/088", Valid: true},
		VendorID:        sql.NullInt32{Int32: 3, Valid: true}, VendorName: sql.NullString{String: "PT Sewa Mobil Nusantara", Valid: true},
		VendorAddress: sql.NullString{String: "Jl. Bengkel Raya No. 2, Bekasi", Valid: true},
		VendorPic:     sql.NullString{String: "Bpk. Hendra", Valid: true},
		RequestNo:     sql.NullString{String: "001/KCE-MNT/X/2026", Valid: true},
		Category:      sql.NullString{String: "REPAIR", Valid: true}, Status: repository.MaintCompleted,
		Description: "Ganti kampas rem depan dan periksa sistem pengereman secara menyeluruh karena berbunyi mendecit saat pengereman — mohon sekalian cek minyak rem.",
		Complaint:   sql.NullString{String: "Rem berbunyi & terasa kurang pakem", Valid: true},
		PlannedDate: sql.NullTime{Time: now, Valid: true}, EstimatedDays: sql.NullInt32{Int32: 2, Valid: true},
		PickupMethod: sql.NullString{String: "PICKUP", Valid: true}, EstimatedCost: sql.NullString{String: "1500000.00", Valid: true},
		CostBearer: sql.NullString{String: "VENDOR", Valid: true}, SubmittedAt: sql.NullTime{Time: now, Valid: true},
		HandoverAt: sql.NullTime{Time: now, Valid: true}, HandoverOdo: sql.NullInt32{Int32: 45210, Valid: true},
		HandoverFuel: sql.NullString{String: "1/2", Valid: true}, HandoverBy: sql.NullString{String: "Andi Wijaya", Valid: true},
		HandoverList: []byte(`{"stnk":true,"mainKey":true,"spareTire":true,"jack":true}`),
		HandoverNote: sql.NullString{String: "Baret halus di bumper belakang kiri (sudah ada sebelumnya).", Valid: true},
		ReturnedAt:   sql.NullTime{Time: now.Add(48 * time.Hour), Valid: true}, ReturnOdo: sql.NullInt32{Int32: 45236, Valid: true},
		ReturnFuel: sql.NullString{String: "1/4", Valid: true}, ReturnBy: sql.NullString{String: "Andi Wijaya", Valid: true},
		ReturnList:    []byte(`{"stnk":true,"mainKey":true,"spareTire":true,"jack":true}`),
		WorkDone:      sql.NullString{String: "Ganti kampas rem depan, kuras minyak rem, setel rem belakang.", Valid: true},
		PartsReplaced: sql.NullString{String: "Kampas rem depan (1 set), minyak rem DOT 3 (1 L)", Valid: true},
		TotalCost:     sql.NullString{String: "1250000.00", Valid: true},
	}
	s := &MaintenanceService{q: &pdfFake{m: m}}
	for _, kind := range []string{PDFRequest, PDFHandover, PDFReturn} {
		data, _, err := s.GeneratePDF(context.Background(), 1, kind)
		if err != nil {
			t.Fatal(err)
		}
		_ = os.WriteFile(filepath.Join(dir, kind+".pdf"), data, 0o644)
	}
}
