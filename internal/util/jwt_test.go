package util

import (
	"testing"

	"booking-system-api/internal/config"
)

// Dua refresh token untuk user yang sama di detik yang sama harus berbeda —
// kolom refresh_tokens.token unik (skenario AU-18: klik ganda / dua perangkat).
func TestCreateRefreshTokenUniqueWithinSameSecond(t *testing.T) {
	config.C.JWTSecret = "test-secret"
	config.C.JWTRefreshExpireDays = 7

	a, _, err := CreateRefreshToken(42)
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := CreateRefreshToken(42)
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("dua refresh token beruntun identik")
	}

	claims, err := ParseToken(a)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Type != "refresh" || claims.UserID != 42 || claims.ID == "" {
		t.Errorf("klaim refresh token salah: %+v", claims)
	}
}
