package util

import (
	"testing"
	"time"
)

func TestStartOfDayWIB(t *testing.T) {
	cases := []struct {
		name string
		in   time.Time
		want string // RFC3339 dalam UTC
	}{
		// 06:00 WIB 17 Sep = 23:00 UTC 16 Sep → harinya tetap 17 Sep WIB.
		{"pagi WIB dikirim sebagai UTC", time.Date(2026, 9, 16, 23, 0, 0, 0, time.UTC), "2026-09-16T17:00:00Z"},
		{"sama, dikirim sebagai +07:00", time.Date(2026, 9, 17, 6, 0, 0, 0, WIB), "2026-09-16T17:00:00Z"},
		// 23:30 WIB 17 Sep = 16:30 UTC 17 Sep.
		{"malam WIB", time.Date(2026, 9, 17, 16, 30, 0, 0, time.UTC), "2026-09-16T17:00:00Z"},
		// 00:00 WIB tepat adalah awal harinya sendiri.
		{"tepat tengah malam WIB", time.Date(2026, 9, 16, 17, 0, 0, 0, time.UTC), "2026-09-16T17:00:00Z"},
	}
	for _, c := range cases {
		got := StartOfDayWIB(c.in).UTC().Format(time.RFC3339)
		if got != c.want {
			t.Errorf("%s: StartOfDayWIB(%s) = %s, want %s", c.name, c.in.Format(time.RFC3339), got, c.want)
		}
	}
}
