package service

import (
	"strings"
	"testing"
)

func TestNormalizeTripExpense_Parking(t *testing.T) {
	// Parkir bisa di kedua mode; tanpa cara bayar & titik tol, lokasi di note.
	for _, mode := range []string{TripModeSimple, TripModeFull} {
		n, err := normalizeTripExpense(mode, CreateTripExpenseRequest{
			Category: "parking", Amount: 5000, ProofPhotoURL: "/uploads/k.jpg",
			Note: "  Parkir RSUD Serang ", PaymentMethod: "PERSONAL", TollPoint: "ENTRY",
		})
		if err != nil {
			t.Fatalf("%s: parkir valid: %v", mode, err)
		}
		if n.Category != "PARKING" || n.PaymentMethod != "" || n.TollPoint != "" || n.Note != "Parkir RSUD Serang" {
			t.Errorf("%s: hasil = %+v", mode, n)
		}
	}
	if _, err := normalizeTripExpense(TripModeSimple, CreateTripExpenseRequest{Category: "PARKING", Amount: 5000}); err == nil ||
		!strings.Contains(err.Error(), "foto") {
		t.Errorf("parkir tanpa karcis harus ditolak: %v", err)
	}
}

func TestParkingBalance(t *testing.T) {
	cases := []struct {
		name            string
		allowance, used float64
		dir             string
		amount          float64
	}{
		{"sisa dikembalikan", 50000, 30000, parkingToOffice, 20000},
		{"kekurangan diganti", 50000, 65000, parkingToDriver, 15000},
		{"tidak dibekali", 0, 30000, parkingToDriver, 30000},
		{"pas", 30000, 30000, parkingNone, 0},
		{"dibekali tapi tak parkir", 20000, 0, parkingToOffice, 20000},
		{"pembulatan sen", 10000, 9999.995, parkingNone, 0},
	}
	for _, c := range cases {
		d, a := parkingBalance(c.allowance, c.used)
		if d != c.dir || a != c.amount {
			t.Errorf("%s: (%s, %v), mau (%s, %v)", c.name, d, a, c.dir, c.amount)
		}
	}
}
