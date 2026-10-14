-- Biaya perjalanan (tol, parkir, lainnya) per booking kendaraan.
-- Dua tingkat fitur (docs/RANCANGAN_BIAYA_PERJALANAN.md): SEDERHANA (tol
-- masuk/keluar + nominal + foto) dan LENGKAP (gerbang, cara bayar, reimburse,
-- anggaran tol SPD) - dipilih lewat master setting trip_expense_full_mode.
-- Idempoten: dijalankan ulang tiap deploy (psql -f).

CREATE TABLE IF NOT EXISTS trip_reimbursements (
    id            SERIAL         PRIMARY KEY,
    "driverId"    INTEGER        NOT NULL REFERENCES drivers(id),
    amount        NUMERIC(14,2)  NOT NULL CHECK (amount > 0),
    "paidAt"      TIMESTAMPTZ    NOT NULL DEFAULT now(),
    note          TEXT           NULL,
    "createdById" INTEGER        NOT NULL REFERENCES users(id),
    "createdAt"   TIMESTAMPTZ    NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS trip_expenses (
    id                SERIAL         PRIMARY KEY,
    "bookingId"       INTEGER        NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    "vehicleId"       INTEGER        NULL REFERENCES vehicles(id),
    "driverId"        INTEGER        NULL REFERENCES drivers(id),
    category          VARCHAR(20)    NOT NULL DEFAULT 'TOLL'
                                     CHECK (category IN ('TOLL', 'PARKING', 'OTHER')),
    -- Mode sederhana: titik bayar tol.
    "tollPoint"       VARCHAR(10)    NULL CHECK ("tollPoint" IN ('ENTRY', 'EXIT')),
    -- Mode lengkap: nama gerbang.
    "entryGate"       VARCHAR(120)   NULL,
    "exitGate"        VARCHAR(120)   NULL,
    amount            NUMERIC(14,2)  NOT NULL CHECK (amount > 0),
    -- Mode lengkap: sumber dana (PERSONAL = perlu reimburse).
    "paymentMethod"   VARCHAR(20)    NULL CHECK ("paymentMethod" IN ('COMPANY_EMONEY', 'PERSONAL')),
    "proofPhotoUrl"   VARCHAR(255)   NOT NULL,
    note              TEXT           NULL,
    "occurredAt"      TIMESTAMPTZ    NOT NULL DEFAULT now(),
    "recordedById"    INTEGER        NOT NULL REFERENCES users(id),
    "reimbursementId" INTEGER        NULL REFERENCES trip_reimbursements(id),
    "voidedAt"        TIMESTAMPTZ    NULL,
    "voidedById"      INTEGER        NULL REFERENCES users(id),
    "voidReason"      TEXT           NULL,
    "createdAt"       TIMESTAMPTZ    NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_trip_expenses_booking ON trip_expenses ("bookingId");
CREATE INDEX IF NOT EXISTS idx_trip_expenses_driver_pending
    ON trip_expenses ("driverId") WHERE "paymentMethod" = 'PERSONAL' AND "reimbursementId" IS NULL AND "voidedAt" IS NULL;
CREATE INDEX IF NOT EXISTS idx_trip_expenses_occurred ON trip_expenses ("occurredAt");

-- Anggaran tol SPD (mode lengkap): saldo e-money kantor yang dibawa supir.
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS "tollBudget" NUMERIC(14,2) NULL;

-- Saklar tingkat fitur - sengaja TIDAK ditampilkan di menu Pengaturan.
INSERT INTO master_settings (key, value, unit, description) VALUES
    ('trip_expense_full_mode', 0, 'bool',
     'Biaya perjalanan mode lengkap (gerbang, cara bayar, reimburse, anggaran tol SPD). 0 = mode sederhana')
ON CONFLICT (key) DO NOTHING;
