-- Fase 1 rancangan voucher BBM (docs/RANCANGAN_VOUCHER_BBM.md): buku saldo
-- BBM/listrik per kendaraan berbasis odometer, master SPBU mitra, dan
-- pembatalan (bukan hapus) catatan pengisian. Idempoten — CI menjalankan
-- ulang semua *.up.sql di setiap deploy.

-- ─── PROFIL BBM KENDARAAN ────────────────────────────────────────────────────
ALTER TABLE vehicles
    ADD COLUMN IF NOT EXISTS "kmPerLiter"           NUMERIC(6,2) NULL CHECK ("kmPerLiter" IS NULL OR "kmPerLiter" > 0),
    ADD COLUMN IF NOT EXISTS "tankCapacityLiter"    NUMERIC(6,2) NULL CHECK ("tankCapacityLiter" IS NULL OR "tankCapacityLiter" > 0),
    ADD COLUMN IF NOT EXISTS "kmPerKwh"             NUMERIC(6,2) NULL CHECK ("kmPerKwh" IS NULL OR "kmPerKwh" > 0),
    ADD COLUMN IF NOT EXISTS "batteryCapacityKwh"   NUMERIC(6,2) NULL CHECK ("batteryCapacityKwh" IS NULL OR "batteryCapacityKwh" > 0),
    -- Titik awal hitung saldo (odometer "isi terakhir" sebelum ada kejadian BBM).
    ADD COLUMN IF NOT EXISTS "fuelBaselineOdometer" INTEGER      NULL CHECK ("fuelBaselineOdometer" IS NULL OR "fuelBaselineOdometer" >= 0);

-- Nilai awal km/liter per kategori (admin menyesuaikan per kendaraan).
CREATE OR REPLACE FUNCTION default_km_per_liter(category_name TEXT)
RETURNS NUMERIC AS $$
    SELECT CASE
        WHEN category_name ILIKE 'MPV%'    THEN 12
        WHEN category_name ILIKE 'SUV%'    THEN 9
        WHEN category_name ILIKE 'Sedan%'  THEN 13
        WHEN category_name ILIKE 'Pickup%' THEN 10
        WHEN category_name ILIKE 'Bus%'    THEN 7
        ELSE 10
    END::NUMERIC;
$$ LANGUAGE sql IMMUTABLE;

-- Kendaraan baru: titik awal = odometer saat dibuat, km/L dari kategori.
CREATE OR REPLACE FUNCTION trigger_vehicle_fuel_defaults()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW."fuelBaselineOdometer" IS NULL THEN
        NEW."fuelBaselineOdometer" := NEW."currentOdometer";
    END IF;
    IF NEW."kmPerLiter" IS NULL AND NEW.energy_type <> 'LISTRIK' THEN
        NEW."kmPerLiter" := default_km_per_liter(
            (SELECT name FROM vehicle_categories WHERE id = NEW."categoryId"));
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS set_vehicle_fuel_defaults ON vehicles;
CREATE TRIGGER set_vehicle_fuel_defaults
    BEFORE INSERT ON vehicles FOR EACH ROW EXECUTE FUNCTION trigger_vehicle_fuel_defaults();

-- Kendaraan yang sudah ada: sekali saja (penanda data_fixes), supaya nilai
-- yang sengaja dikosongkan admin tidak terisi lagi di deploy berikutnya.
CREATE TABLE IF NOT EXISTS data_fixes (
    key        TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DO $$
BEGIN
    INSERT INTO data_fixes (key) VALUES ('000016_vehicle_fuel_profile')
    ON CONFLICT (key) DO NOTHING;
    IF NOT FOUND THEN
        RETURN;
    END IF;

    UPDATE vehicles SET "fuelBaselineOdometer" = "currentOdometer"
    WHERE "fuelBaselineOdometer" IS NULL;

    UPDATE vehicles v SET "kmPerLiter" = default_km_per_liter(vc.name)
    FROM vehicle_categories vc
    WHERE vc.id = v."categoryId" AND v."kmPerLiter" IS NULL AND v.energy_type <> 'LISTRIK';
END $$;

-- ─── SPBU MITRA ──────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS fuel_stations (
    id              SERIAL       PRIMARY KEY,
    name            VARCHAR(150) NOT NULL UNIQUE,
    address         TEXT         NULL,
    phone           VARCHAR(30)  NULL,
    "contactPerson" VARCHAR(100) NULL,
    "isActive"      BOOLEAN      NOT NULL DEFAULT TRUE,
    "createdAt"     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    "updatedAt"     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

DROP TRIGGER IF EXISTS set_updated_at_fuel_stations ON fuel_stations;
CREATE TRIGGER set_updated_at_fuel_stations
    BEFORE UPDATE ON fuel_stations FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

-- ─── PENGISIAN: sumber, SPBU, alasan, pembatalan ─────────────────────────────
ALTER TABLE fuel_expenses
    ADD COLUMN IF NOT EXISTS source       VARCHAR(10) NOT NULL DEFAULT 'DIRECT',  -- DIRECT | VOUCHER
    ADD COLUMN IF NOT EXISTS "stationId"  INTEGER     NULL REFERENCES fuel_stations(id),
    ADD COLUMN IF NOT EXISTS reason       VARCHAR(20) NULL,                       -- SPD | LONG_TRIP | EMERGENCY | OTHER
    ADD COLUMN IF NOT EXISTS "voidedAt"   TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS "voidedById" INTEGER     NULL REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS "voidReason" TEXT        NULL;

CREATE INDEX IF NOT EXISTS idx_fuel_expenses_station_id ON fuel_expenses("stationId");

-- ─── BUKU SALDO (append-only) ────────────────────────────────────────────────
DO $$ BEGIN
    CREATE TYPE fuel_ledger_type AS ENUM (
        'OPENING', 'VOUCHER', 'DIRECT_FILL', 'VOUCHER_RETURN',
        'VOUCHER_REINSTATE', 'VOID', 'ADJUSTMENT');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS fuel_ledger (
    id              SERIAL           PRIMARY KEY,
    "vehicleId"     INTEGER          NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    energy          fuel_category    NOT NULL,              -- BBM = liter, LISTRIK = kWh
    "entryType"     fuel_ledger_type NOT NULL,
    -- Bacaan odometer kejadian ini (NULL = tidak menggeser titik hitung).
    odometer        INTEGER          NULL CHECK (odometer IS NULL OR odometer >= 0),
    "distanceKm"    INTEGER          NULL,
    "kmPerUnit"     NUMERIC(6,2)     NULL,                  -- snapshot km/L atau km/kWh
    accrued         NUMERIC(10,2)    NOT NULL DEFAULT 0,    -- hak dari jarak tempuh
    debit           NUMERIC(10,2)    NOT NULL DEFAULT 0,    -- keluar (negatif = kembali)
    "balanceAfter"  NUMERIC(10,2)    NOT NULL,
    "fuelExpenseId" INTEGER          NULL REFERENCES fuel_expenses(id),
    "reversesId"    INTEGER          NULL REFERENCES fuel_ledger(id),
    "createdById"   INTEGER          NULL REFERENCES users(id),  -- NULL = aksi sistem
    note            TEXT             NULL,
    "createdAt"     TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fuel_ledger_vehicle ON fuel_ledger("vehicleId", energy, id DESC);
CREATE INDEX IF NOT EXISTS idx_fuel_ledger_expense ON fuel_ledger("fuelExpenseId");

-- ─── HARGA: satu master (fuel_types.default_price) ───────────────────────────
DELETE FROM master_settings WHERE key LIKE 'fuel\_price\_%';

-- ─── VIEW LAPORAN: catatan yang dibatalkan tidak dihitung ────────────────────
CREATE OR REPLACE VIEW v_vehicle_summary AS
SELECT
    v.id, r.name AS vehicle_name, v."plateNumber",
    vc.name AS category, v.capacity, r.status, v."currentOdometer",
    COUNT(DISTINCT b.id) AS total_bookings,
    SUM(CASE WHEN b.status = 'COMPLETED' THEN 1 ELSE 0 END) AS completed_bookings,
    COALESCE(SUM(CASE WHEN ft.type = 'BBM' THEN fe.quantity ELSE 0 END), 0) AS total_liter_bbm,
    COALESCE(SUM(CASE WHEN ft.type = 'BBM' THEN fe."totalCost" ELSE 0 END), 0) AS total_cost_bbm,
    COALESCE(SUM(CASE WHEN ft.type = 'LISTRIK' THEN fe.quantity ELSE 0 END), 0) AS total_kwh_listrik,
    COALESCE(SUM(CASE WHEN ft.type = 'LISTRIK' THEN fe."totalCost" ELSE 0 END), 0) AS total_cost_listrik,
    COALESCE(SUM(fe."totalCost"), 0) AS total_fuel_cost
FROM vehicles v
JOIN resources r ON r.id = v."resourceId"
JOIN vehicle_categories vc ON vc.id = v."categoryId"
LEFT JOIN bookings b ON b."resourceId" = r.id
LEFT JOIN fuel_expenses fe ON fe."vehicleId" = v.id AND fe."voidedAt" IS NULL
LEFT JOIN fuel_types ft ON ft.id = fe."fuelTypeId"
GROUP BY v.id, r.name, v."plateNumber", vc.name, v.capacity, r.status, v."currentOdometer";

CREATE OR REPLACE VIEW v_fuel_expense_summary AS
SELECT
    v.id AS vehicle_id, v."plateNumber", r.name AS vehicle_name, vc.name AS category,
    COUNT(CASE WHEN ft.type = 'BBM' THEN 1 END) AS bbm_entries,
    COALESCE(SUM(CASE WHEN ft.type = 'BBM' THEN fe.quantity END), 0) AS total_liter,
    COALESCE(SUM(CASE WHEN ft.type = 'BBM' THEN fe."totalCost" END), 0) AS total_cost_bbm,
    COUNT(CASE WHEN ft.type = 'LISTRIK' THEN 1 END) AS listrik_entries,
    COALESCE(SUM(CASE WHEN ft.type = 'LISTRIK' THEN fe.quantity END), 0) AS total_kwh,
    COALESCE(SUM(CASE WHEN ft.type = 'LISTRIK' THEN fe."totalCost" END), 0) AS total_cost_listrik,
    COALESCE(SUM(fe."totalCost"), 0) AS grand_total
FROM vehicles v
JOIN resources r ON r.id = v."resourceId"
JOIN vehicle_categories vc ON vc.id = v."categoryId"
LEFT JOIN fuel_expenses fe ON fe."vehicleId" = v.id AND fe."voidedAt" IS NULL
LEFT JOIN fuel_types ft ON ft.id = fe."fuelTypeId"
GROUP BY v.id, v."plateNumber", r.name, vc.name;

-- ─── LISTRIK: kWh bebas (meter charger / langsung / estimasi % baterai) ──────
ALTER TABLE fuel_expenses
    ADD COLUMN IF NOT EXISTS "meterStartKwh"  NUMERIC(12,2) NULL,
    ADD COLUMN IF NOT EXISTS "meterEndKwh"    NUMERIC(12,2) NULL,
    ADD COLUMN IF NOT EXISTS "quantitySource" VARCHAR(10)   NULL;  -- INPUT | METER | ESTIMATE

-- ─── VOUCHER BBM ─────────────────────────────────────────────────────────────
DO $$ BEGIN
    CREATE TYPE fuel_voucher_status AS ENUM ('ISSUED', 'USED', 'EXPIRED', 'CANCELLED');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS fuel_vouchers (
    id                 SERIAL              PRIMARY KEY,
    code               VARCHAR(20)         NOT NULL UNIQUE,
    "vehicleId"        INTEGER             NOT NULL REFERENCES vehicles(id),
    "fuelTypeId"       INTEGER             NOT NULL REFERENCES fuel_types(id),
    "stationId"        INTEGER             NOT NULL REFERENCES fuel_stations(id),
    "driverId"         INTEGER             NULL REFERENCES drivers(id),
    "bookingId"        INTEGER             NULL REFERENCES bookings(id),
    "issuedById"       INTEGER             NOT NULL REFERENCES users(id),
    odometer           INTEGER             NOT NULL CHECK (odometer >= 0),
    "distanceKm"       INTEGER             NOT NULL,
    "kmPerLiter"       NUMERIC(6,2)        NOT NULL,
    "accruedLiter"     NUMERIC(10,2)       NOT NULL,
    "carriedLiter"     NUMERIC(10,2)       NOT NULL,
    "tankCapacityLiter" NUMERIC(6,2)       NULL,
    liter              NUMERIC(10,2)       NOT NULL CHECK (liter > 0),
    "pricePerLiter"    NUMERIC(12,2)       NOT NULL,
    amount             NUMERIC(14,2)       NOT NULL,
    "validUntil"       TIMESTAMPTZ         NOT NULL,
    status             fuel_voucher_status NOT NULL DEFAULT 'ISSUED',
    "usedAt"           TIMESTAMPTZ         NULL,
    "usedById"         INTEGER             NULL REFERENCES users(id),
    "usedOdometer"     INTEGER             NULL,
    "receiptPhotoUrl"  VARCHAR(255)        NULL,
    "fuelExpenseId"    INTEGER             NULL REFERENCES fuel_expenses(id),
    "cancelledAt"      TIMESTAMPTZ         NULL,
    "cancelledById"    INTEGER             NULL REFERENCES users(id),
    "cancelReason"     TEXT                NULL,
    "reconciledAt"     TIMESTAMPTZ         NULL,
    "reconciledById"   INTEGER             NULL REFERENCES users(id),
    "invoiceNumber"    VARCHAR(100)        NULL,
    note               TEXT                NULL,
    "createdAt"        TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    "updatedAt"        TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);

-- Maksimal satu voucher aktif per kendaraan.
CREATE UNIQUE INDEX IF NOT EXISTS uq_fuel_vouchers_one_active
    ON fuel_vouchers("vehicleId") WHERE status = 'ISSUED';
CREATE INDEX IF NOT EXISTS idx_fuel_vouchers_status_valid ON fuel_vouchers(status, "validUntil");
CREATE INDEX IF NOT EXISTS idx_fuel_vouchers_station ON fuel_vouchers("stationId", "createdAt");
CREATE INDEX IF NOT EXISTS idx_fuel_vouchers_driver ON fuel_vouchers("driverId");

DROP TRIGGER IF EXISTS set_updated_at_fuel_vouchers ON fuel_vouchers;
CREATE TRIGGER set_updated_at_fuel_vouchers
    BEFORE UPDATE ON fuel_vouchers FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

ALTER TABLE fuel_ledger ADD COLUMN IF NOT EXISTS "voucherId" INTEGER NULL REFERENCES fuel_vouchers(id);
CREATE INDEX IF NOT EXISTS idx_fuel_ledger_voucher ON fuel_ledger("voucherId");

ALTER TABLE fuel_expenses ADD COLUMN IF NOT EXISTS "voucherId" INTEGER NULL REFERENCES fuel_vouchers(id);

INSERT INTO master_settings (key, value, unit, description) VALUES
    ('fuel_voucher_validity_days', 1, 'hari', 'Masa berlaku voucher BBM (1 = hanya hari terbit, WIB)')
ON CONFLICT (key) DO NOTHING;
