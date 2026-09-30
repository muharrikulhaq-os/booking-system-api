-- Isi ulang riwayat persetujuan yang hilang akibat bug B13: Approve dulu
-- menulis action "APPROVE" ke kolom enum approval_action (nilai sah:
-- APPROVED/REJECTED) sehingga insert selalu gagal diam-diam. Data persetujuan
-- tetap ada di bookings ("approvedById", "approvedAt") dan catatan admin di
-- audit_logs ("Booking disetujui: <catatan>"). Cukup SEKALI (penanda di
-- data_fixes — CI menjalankan ulang semua *.up.sql di setiap deploy).

CREATE TABLE IF NOT EXISTS data_fixes (
    key        TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DO $$
BEGIN
    INSERT INTO data_fixes (key) VALUES ('000015_backfill_approval_logs')
    ON CONFLICT (key) DO NOTHING;
    IF NOT FOUND THEN
        RETURN; -- sudah pernah dijalankan
    END IF;

    INSERT INTO approval_logs ("bookingId", "approverId", action, note, "createdAt")
    SELECT b.id, b."approvedById", 'APPROVED',
           (SELECT NULLIF(substring(a.description FROM '^Booking disetujui: (.*)$'), '')
              FROM audit_logs a
             WHERE a."entityType" = 'Booking' AND a."entityId" = b.id AND a.action = 'APPROVE'
             ORDER BY a.id DESC LIMIT 1),
           b."approvedAt"
    FROM bookings b
    WHERE b."approvedById" IS NOT NULL
      AND b."approvedAt" IS NOT NULL
      AND b.status NOT IN ('PENDING', 'REJECTED')
      AND NOT EXISTS (SELECT 1 FROM approval_logs l
                      WHERE l."bookingId" = b.id AND l.action = 'APPROVED');
END $$;
