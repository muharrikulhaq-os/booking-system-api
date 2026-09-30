package middleware

import (
	"context"
	"errors"
	"testing"
)

func TestAccountAllowed(t *testing.T) {
	t.Cleanup(func() { accountLookup = nil; accountCache.Clear() })
	states := map[int]AccountState{
		1: {Active: true, Role: "ADMIN"},
		2: {Active: false, Role: "EMPLOYEE"},
		3: {Active: true, Role: "EMPLOYEE"},
	}
	calls := 0
	SetAccountLookup(func(ctx context.Context, userID int) (AccountState, error) {
		calls++
		if userID == 9 {
			return AccountState{}, errors.New("db down")
		}
		return states[userID], nil
	})
	ctx := context.Background()
	if !accountAllowed(ctx, 1, "ADMIN") {
		t.Error("akun aktif dengan role sama harus lolos")
	}
	if accountAllowed(ctx, 2, "EMPLOYEE") {
		t.Error("akun nonaktif harus ditolak")
	}
	if accountAllowed(ctx, 3, "ADMIN") {
		t.Error("role di token beda dengan database harus ditolak")
	}
	if !accountAllowed(ctx, 9, "ADMIN") {
		t.Error("database bermasalah: token sah tetap diteruskan")
	}
	before := calls
	accountAllowed(ctx, 1, "ADMIN")
	if calls != before {
		t.Error("status akun harus diambil dari cache dalam masa TTL")
	}
}
