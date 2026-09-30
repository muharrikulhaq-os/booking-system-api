-- Lepaskan supir yang TERTAHAN "memegang" kendaraan akibat bug B4/B5
-- (booking EXPIRED/IGNORED dan pindah supir tidak melepas penugasan).
-- Mulai rilis ini penugasan diselaraskan otomatis dari booking aktif
-- (syncDriverHold), jadi pembersihan ini cukup SEKALI.
--
-- CI menjalankan ulang semua *.up.sql di setiap deploy → penanda di
-- data_fixes membuat UPDATE hanya berjalan pada deploy pertama.
--
-- Yang dilepas: penugasan terbuka milik supir yang TIDAK punya booking aktif
-- (APPROVED/ONGOING/OVERDUE), dan kendaraannya punya booking yang sudah
-- berakhir SETELAH penugasan dibuat (jejak alur booking). Penugasan manual
-- dari menu Driver yang tidak pernah terkait booking dibiarkan.

CREATE TABLE IF NOT EXISTS data_fixes (
    key        TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

WITH marker AS (
    INSERT INTO data_fixes (key) VALUES ('000013_release_stuck_driver_assignments')
    ON CONFLICT (key) DO NOTHING
    RETURNING key
)
UPDATE driver_assignments da
SET "releasedAt" = NOW()
WHERE EXISTS (SELECT 1 FROM marker)
  AND da."releasedAt" IS NULL
  AND NOT EXISTS (
      SELECT 1 FROM bookings b
      WHERE b."assignedDriverId" = da."driverId"
        AND b.status IN ('APPROVED', 'ONGOING', 'OVERDUE'))
  AND EXISTS (
      SELECT 1 FROM bookings b
      WHERE b."assignedVehicleId" = da."vehicleId"
        AND b.status IN ('COMPLETED', 'EXPIRED', 'IGNORED', 'CANCELLED', 'REJECTED')
        AND b."updatedAt" >= da."assignedAt");
