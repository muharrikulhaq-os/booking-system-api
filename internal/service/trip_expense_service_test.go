package service

import (
	"database/sql"
	"strings"
	"testing"

	"booking-system-api/internal/repository"
)

func TestNormalizeTripExpense_Simple(t *testing.T) {
	ok := CreateTripExpenseRequest{Amount: 12500, TollPoint: "entry", ProofPhotoURL: "/uploads/x.jpg",
		// isian mode lengkap dibuang di mode sederhana
		EntryGate: "Cikunir", PaymentMethod: "PERSONAL"}
	n, err := normalizeTripExpense(TripModeSimple, ok)
	if err != nil {
		t.Fatalf("valid: %v", err)
	}
	if n.Category != "TOLL" || n.TollPoint != "ENTRY" || n.EntryGate != "" || n.PaymentMethod != "" {
		t.Errorf("hasil = %+v", n)
	}

	bad := []struct {
		name string
		req  CreateTripExpenseRequest
		msg  string
	}{
		{"tanpa titik tol", CreateTripExpenseRequest{Amount: 1, ProofPhotoURL: "p"}, "Tol Masuk atau Tol Keluar"},
		{"tanpa foto", CreateTripExpenseRequest{Amount: 1, TollPoint: "EXIT"}, "foto struk"},
		{"nominal nol", CreateTripExpenseRequest{TollPoint: "EXIT", ProofPhotoURL: "p"}, "nominal wajib"},
		{"parkir di mode sederhana", CreateTripExpenseRequest{Category: "PARKING", Amount: 1, ProofPhotoURL: "p"}, "hanya biaya tol"},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := normalizeTripExpense(TripModeSimple, tc.req); err == nil || !strings.Contains(err.Error(), tc.msg) {
				t.Fatalf("err = %v, mau memuat %q", err, tc.msg)
			}
		})
	}
}

func TestNormalizeTripExpense_Full(t *testing.T) {
	n, err := normalizeTripExpense(TripModeFull, CreateTripExpenseRequest{
		Amount: 27000, EntryGate: "  Cikunir 1 ", ExitGate: "Cikampek Utama", PaymentMethod: "personal",
		ProofPhotoURL: "p", TollPoint: "ENTRY", // titik tol dibuang di mode lengkap
	})
	if err != nil {
		t.Fatalf("valid: %v", err)
	}
	if n.Category != "TOLL" || n.EntryGate != "Cikunir 1" || n.PaymentMethod != "PERSONAL" || n.TollPoint != "" {
		t.Errorf("hasil = %+v", n)
	}
	if _, err := normalizeTripExpense(TripModeFull, CreateTripExpenseRequest{
		Category: "PARKING", Amount: 5000, PaymentMethod: "COMPANY_EMONEY", ProofPhotoURL: "p"}); err != nil {
		t.Errorf("parkir valid: %v", err)
	}

	bad := []struct {
		name string
		req  CreateTripExpenseRequest
		msg  string
	}{
		{"tanpa cara bayar", CreateTripExpenseRequest{Amount: 1, EntryGate: "A", ProofPhotoURL: "p"}, "cara bayar"},
		{"tol tanpa gerbang", CreateTripExpenseRequest{Amount: 1, PaymentMethod: "PERSONAL", ProofPhotoURL: "p"}, "gerbang"},
		{"lainnya tanpa keterangan", CreateTripExpenseRequest{Category: "OTHER", Amount: 1, PaymentMethod: "PERSONAL", ProofPhotoURL: "p"}, "keterangan"},
		{"jenis tak dikenal", CreateTripExpenseRequest{Category: "HOTEL", Amount: 1, PaymentMethod: "PERSONAL", ProofPhotoURL: "p"}, "tidak dikenal"},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := normalizeTripExpense(TripModeFull, tc.req); err == nil || !strings.Contains(err.Error(), tc.msg) {
				t.Fatalf("err = %v, mau memuat %q", err, tc.msg)
			}
		})
	}
}

func TestCanRecordTrip(t *testing.T) {
	booking := func(status repository.BookingStatus) repository.GetBookingByIDRow {
		return repository.GetBookingByIDRow{
			ResourceType: repository.ResourceTypeVEHICLE, Status: status,
			AssignedDriverId: sql.NullInt32{Int32: 7, Valid: true},
		}
	}
	cases := []struct {
		name     string
		b        repository.GetBookingByIDRow
		role     string
		driverID int32
		allowed  bool
	}{
		{"supir pemegang, berjalan", booking(repository.BookingStatusONGOING), "DRIVER", 7, true},
		{"supir pemegang, terlambat", booking(repository.BookingStatusOVERDUE), "DRIVER", 7, true},
		{"supir pemegang, sudah kembali", booking(repository.BookingStatusRETURNED), "DRIVER", 7, false},
		{"supir lain", booking(repository.BookingStatusONGOING), "DRIVER", 8, false},
		{"admin, sudah selesai (susulan)", booking(repository.BookingStatusCOMPLETED), "ADMIN", 0, true},
		{"admin, belum dimulai", booking(repository.BookingStatusAPPROVED), "ADMIN", 0, false},
		{"karyawan", booking(repository.BookingStatusONGOING), "EMPLOYEE", 0, false},
		{"booking ruangan", repository.GetBookingByIDRow{ResourceType: repository.ResourceTypeROOM, Status: repository.BookingStatusONGOING}, "ADMIN", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := canRecordTrip(tc.b, tc.role, tc.driverID)
			if (err == nil) != tc.allowed {
				t.Fatalf("err = %v, boleh = %v", err, tc.allowed)
			}
		})
	}
}

func TestTripSummary(t *testing.T) {
	pay := func(m string) sql.NullString { return sql.NullString{String: m, Valid: m != ""} }
	items := []repository.TripExpense{
		{Category: "TOLL", Amount: 100000, PaymentMethod: pay("COMPANY_EMONEY")},
		{Category: "TOLL", Amount: 50000, PaymentMethod: pay("PERSONAL")},
		{Category: "PARKING", Amount: 10000, PaymentMethod: pay("PERSONAL"), ReimbursementID: sql.NullInt32{Int32: 1, Valid: true}},
		{Category: "TOLL", Amount: 999999, PaymentMethod: pay("PERSONAL"), VoidedAt: sql.NullTime{Valid: true}}, // dibatalkan
		{Category: "TOLL", Amount: 7500}, // mode sederhana: tanpa cara bayar
	}
	s := tripSummary(items, sql.NullFloat64{Float64: 120000, Valid: true})
	want := map[string]any{
		"total": 167500.0, "count": 4, "companyPaid": 100000.0, "personalPaid": 60000.0,
		"reimbursePending": 50000.0, "reimbursePaid": 10000.0, "tollBudget": 120000.0, "budgetRemaining": 20000.0,
	}
	for k, v := range want {
		if s[k] != v {
			t.Errorf("%s = %v, mau %v", k, s[k], v)
		}
	}
	if s["byCategory"].(map[string]float64)["PARKING"] != 10000 {
		t.Errorf("byCategory = %v", s["byCategory"])
	}
	if noBudget := tripSummary(items, sql.NullFloat64{}); noBudget["tollBudget"] != nil || noBudget["budgetRemaining"] != nil {
		t.Errorf("tanpa anggaran = %v", noBudget)
	}
}
