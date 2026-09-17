package http

import (
	"database/sql"
	"testing"
	"time"
)

// Nilai waktu untuk Excel harus berzona WIB, apa pun zona asalnya — excelize
// menulis jam dinding menurut zona nilai tersebut.
func TestExcelCellValueUsesWIB(t *testing.T) {
	instant := time.Date(2026, 9, 16, 23, 0, 0, 0, time.UTC) // 06:00 WIB 17 Sep

	got, ok := excelCellValue(instant).(time.Time)
	if !ok || got.Format("2006-01-02 15:04") != "2026-09-17 06:00" {
		t.Errorf("time.Time → %v, want jam dinding 2026-09-17 06:00 WIB", got)
	}

	gotNull, ok := excelCellValue(sql.NullTime{Time: instant, Valid: true}).(time.Time)
	if !ok || gotNull.Format("2006-01-02 15:04") != "2026-09-17 06:00" {
		t.Errorf("sql.NullTime valid → %v, want jam dinding 2026-09-17 06:00 WIB", gotNull)
	}

	if v := excelCellValue(sql.NullTime{}); v != "" {
		t.Errorf("sql.NullTime kosong → %v, want string kosong", v)
	}
	if v := excelCellValue("teks"); v != "teks" {
		t.Errorf("nilai non-waktu berubah: %v", v)
	}
}
