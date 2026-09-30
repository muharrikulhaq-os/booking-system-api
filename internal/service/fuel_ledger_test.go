package service

import (
	"database/sql"
	"testing"
	"time"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

func nf(f float64) sql.NullFloat64 { return sql.NullFloat64{Float64: f, Valid: true} }

func TestAccrue(t *testing.T) {
	cases := []struct {
		km   int32
		kpu  sql.NullFloat64
		want float64
	}{
		{360, nf(12), 30},
		{400, nf(12), 33.33},
		{0, nf(12), 0},
		{-5, nf(12), 0},
		{500, sql.NullFloat64{}, 0}, // km/L belum diatur → tidak dihitung
	}
	for _, c := range cases {
		if got := accrue(c.km, c.kpu); got != c.want {
			t.Errorf("accrue(%d, %v) = %v, want %v", c.km, c.kpu, got, c.want)
		}
	}
}

// Contoh §2 rancangan: Avanza 12 km/L, tangki 45 L.
func TestLedgerExampleFromDesign(t *testing.T) {
	type step struct {
		odo      int32
		out      float64 // liter keluar (voucher = seluruh saldo tersedia, dibatasi tangki)
		voucher  bool
		wantBal  float64
		wantLitr float64
	}
	steps := []step{
		{50360, 0, true, 0, 30},
		{50900, 40, false, 5, 0},
		{51200, 0, true, 0, 30},
		{51600, 50, false, -16.67, 0},
		{52200, 0, true, 0, 33.33},
	}
	kpu, tank := nf(12), 45.0
	cp, bal := int32(50000), 0.0
	for i, s := range steps {
		accr := accrue(s.odo-cp, kpu)
		out := s.out
		if s.voucher {
			out = round2(bal + accr)
			if out > tank {
				out = tank
			}
			if out != s.wantLitr {
				t.Fatalf("step %d: voucher %.2f L, want %.2f", i+1, out, s.wantLitr)
			}
		}
		bal = round2(bal + accr - out)
		cp = s.odo
		if bal != s.wantBal {
			t.Fatalf("step %d: saldo %.2f, want %.2f", i+1, bal, s.wantBal)
		}
	}
}

func TestVoucherValidUntilIsEndOfWIBDay(t *testing.T) {
	// 2026-10-01 23:30 WIB = 16:30 UTC
	now := time.Date(2026, 10, 1, 16, 30, 0, 0, time.UTC)
	got := voucherValidUntil(now, 1).In(util.WIB)
	if got.Day() != 1 || got.Hour() != 23 || got.Minute() != 59 || got.Second() != 59 {
		t.Fatalf("1 hari → %v, want 2026-10-01 23:59:59 WIB", got)
	}
	// 2026-10-01 00:30 WIB (masih 30 Sep UTC)
	now = time.Date(2026, 9, 30, 17, 30, 0, 0, time.UTC)
	got = voucherValidUntil(now, 3).In(util.WIB)
	if got.Month() != 10 || got.Day() != 3 || got.Hour() != 23 {
		t.Fatalf("3 hari → %v, want 2026-10-03 23:59:59 WIB", got)
	}
}

func TestResolveQuantityElectricity(t *testing.T) {
	p := repository.VehicleFuelProfile{BatteryCapacityKwh: nf(60)}
	el := repository.FuelCategoryLISTRIK

	q, err := resolveQuantity(CreateFuelExpenseRequest{Kwh: 20}, el, p)
	if err != nil || q.Quantity != 20 || q.Source != "INPUT" {
		t.Fatalf("kWh langsung: %+v %v", q, err)
	}
	q, err = resolveQuantity(CreateFuelExpenseRequest{MeterStartKwh: 1200.5, MeterEndKwh: 1231}, el, p)
	if err != nil || q.Quantity != 30.5 || q.Source != "METER" {
		t.Fatalf("meter: %+v %v", q, err)
	}
	q, err = resolveQuantity(CreateFuelExpenseRequest{BatteryBefore: 20, BatteryAfter: 80}, el, p)
	if err != nil || q.Quantity != 40 || q.Source != "ESTIMATE" { // 60% × 60 kWh ÷ 0,9
		t.Fatalf("estimasi: %+v %v", q, err)
	}
	if _, err := resolveQuantity(CreateFuelExpenseRequest{BatteryBefore: 20, BatteryAfter: 80}, el,
		repository.VehicleFuelProfile{}); err == nil {
		t.Fatal("estimasi tanpa kapasitas baterai harus ditolak")
	}
	if _, err := resolveQuantity(CreateFuelExpenseRequest{}, el, p); err == nil {
		t.Fatal("tanpa kWh/meter/% baterai harus ditolak")
	}
	if _, err := resolveQuantity(CreateFuelExpenseRequest{}, repository.FuelCategoryBBM, p); err == nil {
		t.Fatal("BBM tanpa liter harus ditolak")
	}
}

func TestBalanceViewVoucherable(t *testing.T) {
	p := repository.VehicleFuelProfile{
		VehicleID: 1, EnergyType: repository.EnergyTypeBBM, CurrentOdometer: 51000,
		KmPerLiter: nf(10), TankCapacityLiter: nf(40),
	}
	v := balanceView(p, repository.FuelCategoryBBM, ledgerState{CheckpointOdometer: 50000, Balance: 5}, true)
	if v["available"] != 105.0 || v["voucherable"] != 40.0 {
		t.Fatalf("available=%v voucherable=%v, want 105 / 40 (dibatasi tangki)", v["available"], v["voucherable"])
	}
	if w := v["warnings"].([]string); len(w) != 1 {
		t.Fatalf("saldo > 1,5× tangki harus diberi peringatan, got %v", w)
	}
}
