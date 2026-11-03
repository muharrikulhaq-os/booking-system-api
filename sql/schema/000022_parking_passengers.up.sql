-- Uang parkir per booking + penumpang booking — docs/RANCANGAN_PARKIR_PENUMPANG.md.
-- Idempoten: dijalankan ulang tiap deploy (psql -f).

-- ── Uang parkir ──────────────────────────────────────────────────────────────
-- Bekal uang parkir dari kantor (diatur admin). NULL / 0 = tidak dibekali.
-- Catatan parkir sendiri memakai trip_expenses (category = 'PARKING').
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS "parkingAllowance" NUMERIC(14,2) NULL;

-- Penyelesaian selisih parkir: satu kali per perjalanan (booking utama bila digabung).
--   TO_DRIVER = kekurangan / tanpa bekal diganti ke supir (ikut pembayaran reimburse)
--   TO_OFFICE = sisa bekal dikembalikan supir ke kantor
CREATE TABLE IF NOT EXISTS parking_settlements (
    id                SERIAL         PRIMARY KEY,
    "bookingId"       INTEGER        NOT NULL UNIQUE REFERENCES bookings(id) ON DELETE CASCADE,
    "driverId"        INTEGER        NOT NULL REFERENCES drivers(id),
    direction         VARCHAR(10)    NOT NULL CHECK (direction IN ('TO_DRIVER', 'TO_OFFICE')),
    amount            NUMERIC(14,2)  NOT NULL CHECK (amount > 0),
    allowance         NUMERIC(14,2)  NOT NULL DEFAULT 0,  -- bekal saat diselesaikan (jejak)
    used              NUMERIC(14,2)  NOT NULL DEFAULT 0,  -- parkir terpakai saat diselesaikan
    "reimbursementId" INTEGER        NULL REFERENCES trip_reimbursements(id),
    "settledAt"       TIMESTAMPTZ    NOT NULL DEFAULT now(),
    note              TEXT           NULL,
    "createdById"     INTEGER        NOT NULL REFERENCES users(id),
    "createdAt"       TIMESTAMPTZ    NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_parking_settlements_driver ON parking_settlements ("driverId");
CREATE INDEX IF NOT EXISTS idx_parking_settlements_reimbursement ON parking_settlements ("reimbursementId");
CREATE INDEX IF NOT EXISTS idx_trip_expenses_parking
    ON trip_expenses ("bookingId") WHERE category = 'PARKING' AND "voidedAt" IS NULL;

-- ── Penumpang (karyawan yang ikut) ──────────────────────────────────────────
-- Booking tetap satu (atas nama pemesan); tabel ini hanya daftar yang ikut.
CREATE TABLE IF NOT EXISTS booking_passengers (
    id          SERIAL       PRIMARY KEY,
    "bookingId" INTEGER      NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    "userId"    INTEGER      NOT NULL REFERENCES users(id),
    "addedById" INTEGER      NULL REFERENCES users(id),
    "createdAt" TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE ("bookingId", "userId")
);
CREATE INDEX IF NOT EXISTS idx_booking_passengers_user ON booking_passengers ("userId");
