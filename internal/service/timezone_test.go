package service

import (
	"testing"
	"time"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

// Jendela SPD harus sama persis apa pun zona nilai yang dikirim klien:
// mobile mengirim UTC ("Z"), web dulu mengirim +07:00.
func TestEffectiveConflictWindowIsWIBDay(t *testing.T) {
	// Perjalanan SPD 06:00–20:00 WIB tanggal 17 Sep.
	startUTC := time.Date(2026, 9, 16, 23, 0, 0, 0, time.UTC)
	endUTC := time.Date(2026, 9, 17, 13, 0, 0, 0, time.UTC)
	startWIB := startUTC.In(util.WIB)
	endWIB := endUTC.In(util.WIB)

	wantStart := "2026-09-16T17:00:00Z" // 17 Sep 00:00 WIB
	wantEnd := "2026-09-17T17:00:00Z"   // 18 Sep 00:00 WIB

	for name, in := range map[string][2]time.Time{
		"klien kirim UTC":    {startUTC, endUTC},
		"klien kirim +07:00": {startWIB, endWIB},
	} {
		s, e := effectiveConflictWindow(repository.BookingTypeSPD, in[0], in[1])
		if got := s.UTC().Format(time.RFC3339); got != wantStart {
			t.Errorf("%s: start = %s, want %s", name, got, wantStart)
		}
		if got := e.UTC().Format(time.RFC3339); got != wantEnd {
			t.Errorf("%s: end = %s, want %s", name, got, wantEnd)
		}
	}

	// NON_SPD tidak diperluas.
	s, e := effectiveConflictWindow(repository.BookingTypeNONSPD, startUTC, endUTC)
	if !s.Equal(startUTC) || !e.Equal(endUTC) {
		t.Errorf("NON_SPD window berubah: %s–%s", s, e)
	}
}

func TestPeriodBoundsUseWIBCalendar(t *testing.T) {
	// 1 Okt 03:00 WIB = 30 Sep 20:00 UTC. Secara UTC masih September, tetapi
	// "bulan ini" bagi pengguna WIB sudah Oktober.
	at := time.Date(2026, 9, 30, 20, 0, 0, 0, time.UTC)

	start, end := periodBoundsAt("monthly", at)
	if got := start.UTC().Format(time.RFC3339); got != "2026-09-30T17:00:00Z" {
		t.Errorf("monthly start = %s, want 1 Okt 00:00 WIB (2026-09-30T17:00:00Z)", got)
	}
	if got := end.Add(time.Nanosecond).UTC().Format(time.RFC3339); got != "2026-10-31T17:00:00Z" {
		t.Errorf("monthly end+1ns = %s, want 1 Nov 00:00 WIB (2026-10-31T17:00:00Z)", got)
	}

	start, _ = periodBoundsAt("yearly", time.Date(2026, 12, 31, 18, 0, 0, 0, time.UTC)) // 1 Jan 2027 01:00 WIB
	if got := start.UTC().Format(time.RFC3339); got != "2026-12-31T17:00:00Z" {
		t.Errorf("yearly start = %s, want 1 Jan 2027 00:00 WIB (2026-12-31T17:00:00Z)", got)
	}
}
