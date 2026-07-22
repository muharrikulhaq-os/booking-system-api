-- Odometer perjalanan: awal diisi driver saat START (beserta lokasi + foto),
-- akhir diisi driver di LAPORAN PENGEMBALIAN. Dipakai untuk perhitungan laporan.

ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS "odometerStart" INTEGER      NULL CHECK ("odometerStart" IS NULL OR "odometerStart" >= 0),
    ADD COLUMN IF NOT EXISTS "startLocation" VARCHAR(500) NULL,
    ADD COLUMN IF NOT EXISTS "startPhotoUrl" TEXT         NULL;

ALTER TABLE booking_return_reports
    ADD COLUMN IF NOT EXISTS odometer INTEGER NULL CHECK (odometer IS NULL OR odometer >= 0);
