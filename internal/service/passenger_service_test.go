package service

import (
	"context"
	"strings"
	"testing"

	"booking-system-api/internal/repository"
)

// lookupQuerier: karyawan aktif = id 2..9; penumpang booking 5 = user 7.
type lookupQuerier struct {
	*MockQuerier
}

func (lookupQuerier) LookupActiveUsers(_ context.Context, _ string, ids []int32, _ int32) ([]repository.UserLookup, error) {
	var out []repository.UserLookup
	for _, id := range ids {
		if id >= 2 && id <= 9 {
			out = append(out, repository.UserLookup{ID: id})
		}
	}
	return out, nil
}

func (lookupQuerier) IsBookingPassenger(_ context.Context, bookingID, userID int32) (bool, error) {
	return bookingID == 5 && userID == 7, nil
}

func TestNormalizePassengerIDs(t *testing.T) {
	q := lookupQuerier{&MockQuerier{}}
	ctx := context.Background()

	// Duplikat & pemesan (user 2) dibuang, urutan dipertahankan.
	got, err := normalizePassengerIDs(ctx, q, 2, []int32{3, 2, 4, 3, 0}, 6)
	if err != nil || len(got) != 2 || got[0] != 3 || got[1] != 4 {
		t.Fatalf("got %v, %v", got, err)
	}
	// Kosong boleh.
	if got, err := normalizePassengerIDs(ctx, q, 2, nil, 6); err != nil || len(got) != 0 {
		t.Fatalf("kosong: %v %v", got, err)
	}
	// Kapasitas 4 → maksimal 3 karyawan yang ikut.
	if _, err := normalizePassengerIDs(ctx, q, 2, []int32{3, 4, 5, 6}, 4); err == nil || !strings.Contains(err.Error(), "maksimal 3") {
		t.Fatalf("lewat kapasitas harus ditolak: %v", err)
	}
	// User tidak aktif / bukan karyawan.
	if _, err := normalizePassengerIDs(ctx, q, 2, []int32{3, 42}, 6); err == nil || !strings.Contains(err.Error(), "tidak aktif") {
		t.Fatalf("user tak dikenal harus ditolak: %v", err)
	}
}

func TestEmployeeCanView(t *testing.T) {
	q := lookupQuerier{&MockQuerier{}}
	ctx := context.Background()
	b := repository.GetBookingByIDRow{ID: 5, UserId: 2}
	if !employeeCanView(ctx, q, b, 2) {
		t.Error("pemesan harus bisa melihat")
	}
	if !employeeCanView(ctx, q, b, 7) {
		t.Error("penumpang harus bisa melihat")
	}
	if employeeCanView(ctx, q, b, 8) {
		t.Error("karyawan lain tidak boleh melihat")
	}
}

func TestNormalizeGuests(t *testing.T) {
	got, err := normalizeGuests([]GuestInput{{Name: "  Pak Joko ", PhoneNumber: "0812 3456 7890", Note: " Dinas PU "}, {Name: "Bu Ani"}})
	if err != nil || len(got) != 2 || got[0].Name != "Pak Joko" || got[0].Note != "Dinas PU" || got[1].PhoneNumber != "" {
		t.Fatalf("got %+v, %v", got, err)
	}
	if _, err := normalizeGuests([]GuestInput{{Name: " "}}); err == nil || !strings.Contains(err.Error(), "nama tamu") {
		t.Errorf("nama kosong harus ditolak: %v", err)
	}
	if _, err := normalizeGuests([]GuestInput{{Name: "X", PhoneNumber: "abc"}}); err == nil {
		t.Error("nomor HP tidak valid harus ditolak")
	}
	if err := checkSeats(3, 2, 6); err != nil {
		t.Errorf("5 orang + pemesan muat di 6: %v", err)
	}
	if err := checkSeats(3, 3, 6); err == nil || !strings.Contains(err.Error(), "maksimal 5") {
		t.Errorf("lewat kapasitas harus ditolak: %v", err)
	}
}
