-- Jumlah penumpang booking kendaraan kini SELALU = pemesan + karyawan yang
-- ikut + tamu (keputusan pemilik produk 2026-11-19; supir tidak dihitung).
-- Dulu angkanya diisi manual & hanya bisa naik, sehingga tidak cocok dengan
-- daftar (mis. "2 orang · 0 karyawan · 0 tamu", atau tetap 7 setelah satu
-- penumpang dihapus).
--
-- Perbaikan data satu kali untuk booking kendaraan yang BELUM selesai
-- (PENDING / APPROVED / ONGOING / OVERDUE). Riwayat yang sudah selesai
-- dibiarkan apa adanya. Penanda di data_fixes - CI menjalankan ulang semua
-- *.up.sql setiap deploy.

DO $$
BEGIN
    INSERT INTO data_fixes (key) VALUES ('000027_passenger_count_from_list')
    ON CONFLICT (key) DO NOTHING;
    IF NOT FOUND THEN
        RETURN; -- sudah pernah dijalankan
    END IF;

    UPDATE bookings b
    SET "passengerCount" = 1 + (SELECT COUNT(*) FROM booking_passengers bp WHERE bp."bookingId" = b.id),
        "updatedAt" = NOW()
    FROM resources r
    WHERE r.id = b."resourceId"
      AND r.type = 'VEHICLE'
      AND b.status IN ('PENDING', 'APPROVED', 'ONGOING', 'OVERDUE')
      AND b."passengerCount" <> 1 + (SELECT COUNT(*) FROM booking_passengers bp WHERE bp."bookingId" = b.id);
END $$;
