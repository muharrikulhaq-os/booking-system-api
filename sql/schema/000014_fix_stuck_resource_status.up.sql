-- Perbaiki status kendaraan yang TELANJUR salah akibat bug B10–B12. Mulai
-- rilis ini status dihitung ulang dari keadaan nyata (syncResourceStatus),
-- jadi pembersihan ini cukup SEKALI (penanda di data_fixes — CI menjalankan
-- ulang semua *.up.sql di setiap deploy).

CREATE TABLE IF NOT EXISTS data_fixes (
    key        TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DO $$
BEGIN
    INSERT INTO data_fixes (key) VALUES ('000014_fix_stuck_resource_status')
    ON CONFLICT (key) DO NOTHING;
    IF NOT FOUND THEN
        RETURN; -- sudah pernah dijalankan
    END IF;

    -- 1) Sedang dipakai trip (ONGOING/OVERDUE) tapi tidak IN_USE → IN_USE
    --    (mis. maintenance dihapus saat trip berjalan menimpanya jadi AVAILABLE).
    UPDATE resources r SET status = 'IN_USE'
    WHERE r.status <> 'IN_USE'
      AND EXISTS (SELECT 1 FROM bookings b
                  WHERE b."resourceId" = r.id AND b.status IN ('ONGOING', 'OVERDUE'));

    -- 2) MAINTENANCE tanpa maintenance yang sedang berlangsung, padahal:
    --    (a) hanya punya maintenance TERJADWAL di masa depan (B11), atau
    --    (b) maintenance terakhirnya selesai SETELAH status diset (B12) —
    --        MAINTENANCE yang diset admin setelah itu tidak disentuh.
    UPDATE resources r SET status = 'AVAILABLE'
    WHERE r.status = 'MAINTENANCE'
      AND r.type = 'VEHICLE'
      AND NOT EXISTS (SELECT 1 FROM bookings b
                      WHERE b."resourceId" = r.id AND b.status IN ('ONGOING', 'OVERDUE'))
      AND NOT EXISTS (SELECT 1 FROM vehicles v JOIN maintenance_records m ON m."vehicleId" = v.id
                      WHERE v."resourceId" = r.id AND m.status <> 'completed'
                        AND m."startDate" <= NOW()
                        AND (m."endDate" IS NULL OR m."endDate" > NOW()))
      AND (
          EXISTS (SELECT 1 FROM vehicles v JOIN maintenance_records m ON m."vehicleId" = v.id
                  WHERE v."resourceId" = r.id AND m.status <> 'completed' AND m."startDate" > NOW())
       OR (NOT EXISTS (SELECT 1 FROM vehicles v JOIN maintenance_records m ON m."vehicleId" = v.id
                       WHERE v."resourceId" = r.id AND m.status <> 'completed')
           AND EXISTS (SELECT 1 FROM vehicles v JOIN maintenance_records m ON m."vehicleId" = v.id
                       WHERE v."resourceId" = r.id AND m.status = 'completed'
                         AND m."completedAt" >= r."updatedAt"))
      );

    -- 3) AVAILABLE padahal maintenance sedang berlangsung → MAINTENANCE
    --    (mis. booking selesai menimpa status saat maintenance masih terbuka).
    UPDATE resources r SET status = 'MAINTENANCE'
    WHERE r.status = 'AVAILABLE'
      AND NOT EXISTS (SELECT 1 FROM bookings b
                      WHERE b."resourceId" = r.id AND b.status IN ('ONGOING', 'OVERDUE'))
      AND EXISTS (SELECT 1 FROM vehicles v JOIN maintenance_records m ON m."vehicleId" = v.id
                  WHERE v."resourceId" = r.id AND m.status <> 'completed'
                    AND m."startDate" <= NOW()
                    AND (m."endDate" IS NULL OR m."endDate" > NOW()));
END $$;
