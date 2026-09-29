package util

import "time"

// Konvensi waktu aplikasi — berlaku untuk backend, web, dan mobile:
//
//   - Penyimpanan: kolom `timestamptz`, yaitu titik waktu absolut.
//   - Transport API: RFC3339 dalam UTC (`...Z`). cmd/api/main.go memaksa
//     time.Local = UTC supaya format keluaran JSON tidak bergantung pada zona
//     waktu OS server (sebelumnya ikut OS: `+07:00` di server WIB, `Z` di
//     server UTC).
//   - Logika yang bergantung "hari", "bulan", atau "jam kerja" (SPD satu hari
//     penuh, batas bulan laporan, lembur 18:00): SELALU WIB (UTC+7) secara
//     eksplisit — lewat WIB di sini, atau zona sesi DB / `AT TIME ZONE
//     'Asia/Jakarta'` di SQL. Jangan pernah bergantung pada zona nilai yang
//     masuk: klien mengirim UTC, dan nilai dari DB juga UTC.
//
// FixedZone, bukan time.LoadLocation("Asia/Jakarta"): Indonesia tidak memakai
// DST, dan LoadLocation butuh database zona waktu yang tidak selalu tersedia
// di server Windows tanpa instalasi Go.
var WIB = time.FixedZone("WIB", 7*60*60)

// StartOfDayWIB mengembalikan pukul 00:00 WIB pada hari kalender WIB yang
// memuat t, apa pun zona t.
func StartOfDayWIB(t time.Time) time.Time {
	w := t.In(WIB)
	return time.Date(w.Year(), w.Month(), w.Day(), 0, 0, 0, 0, WIB)
}
