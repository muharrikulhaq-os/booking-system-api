-- Perbaikan data satu kali: booking yang sudah digabung (merge) tetapi masih
-- PENDING. Dulu web/mobile menyetujui booking lewat panggilan terpisah setelah
-- merge; bila panggilan itu gagal (di mobile selalu gagal karena yang
-- disetujui booking utama yang sudah APPROVED), booking tertinggal "Menunggu".
-- Sejak perbaikan ini backend menyetujuinya langsung saat merge.
--
-- Hanya booking yang pasangan gabungnya masih APPROVED (trip belum berjalan)
-- dan jadwalnya belum lewat. Disetujui atas nama admin yang melakukan merge.
-- Cukup SEKALI (penanda di data_fixes - CI menjalankan ulang semua *.up.sql).

DO $$
BEGIN
    INSERT INTO data_fixes (key) VALUES ('000025_approve_merged_pending')
    ON CONFLICT (key) DO NOTHING;
    IF NOT FOUND THEN
        RETURN; -- sudah pernah dijalankan
    END IF;

    CREATE TEMP TABLE fix_merged ON COMMIT DROP AS
    SELECT DISTINCT ON (b.id) b.id, m."mergedById" AS admin_id,
           CASE WHEN m."mergedBookingId" = b.id THEN m."primaryBookingId" ELSE m."mergedBookingId" END AS partner_id
    FROM bookings b
    JOIN booking_merges m ON b.id IN (m."mergedBookingId", m."primaryBookingId")
    JOIN bookings p ON p.id = CASE WHEN m."mergedBookingId" = b.id THEN m."primaryBookingId" ELSE m."mergedBookingId" END
    WHERE b.status = 'PENDING'
      AND p.status = 'APPROVED'
      AND b."endDate" > NOW()
    ORDER BY b.id, m.id DESC;

    UPDATE bookings b
    SET status = 'APPROVED', "approvedById" = f.admin_id, "approvedAt" = NOW(), "updatedAt" = NOW()
    FROM fix_merged f
    WHERE b.id = f.id AND b.status = 'PENDING';

    INSERT INTO approval_logs ("bookingId", "approverId", action, note)
    SELECT f.id, f.admin_id, 'APPROVED',
           'Disetujui otomatis karena digabung dengan booking #' || f.partner_id || ' (perbaikan data)'
    FROM fix_merged f;

    INSERT INTO audit_logs ("userId", action, "entityType", "entityId", description)
    SELECT f.admin_id, 'APPROVE', 'Booking', f.id,
           'Booking disetujui: Disetujui otomatis karena digabung dengan booking #' || f.partner_id || ' (perbaikan data)'
    FROM fix_merged f;
END $$;
