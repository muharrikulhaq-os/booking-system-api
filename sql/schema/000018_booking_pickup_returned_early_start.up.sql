-- 000018: lokasi penjemputan & tujuan, status RETURNED, setting mulai lebih awal.
-- Idempoten — seluruh file migrasi dijalankan ulang setiap deploy (psql -f,
-- tiap statement autocommit), jadi nilai enum baru bisa langsung dipakai.

-- ─── 1. Lokasi penjemputan & tujuan (booking kendaraan) ─────────────────────
ALTER TABLE bookings
    ADD COLUMN IF NOT EXISTS "pickupLocation" VARCHAR(255) NULL,
    ADD COLUMN IF NOT EXISTS "destination"    VARCHAR(255) NULL;

-- ─── 2. Status RETURNED ("Sudah Kembali") ───────────────────────────────────
-- Supir sudah mengirim laporan pengembalian: kendaraan sudah kembali ke kantor
-- dan kendaraan + supir langsung bebas; admin tinggal menyelesaikan
-- (RETURNED → COMPLETED). Booking lama yang sudah berlaporan tapi masih
-- ONGOING/OVERDUE sengaja TIDAK diubah — admin menyelesaikannya seperti biasa.
ALTER TYPE booking_status ADD VALUE IF NOT EXISTS 'RETURNED';

-- ─── 3. Setting: batas mulai lebih awal (menit sebelum jadwal) ──────────────
INSERT INTO master_settings (key, value, unit, description) VALUES
    ('booking_start_early_minutes_spd',     180, 'menit', 'Booking kendaraan SPD boleh dimulai paling cepat sekian menit sebelum jadwal'),
    ('booking_start_early_minutes_non_spd',  15, 'menit', 'Booking kendaraan Non-SPD boleh dimulai paling cepat sekian menit sebelum jadwal'),
    ('booking_start_early_minutes_room',     30, 'menit', 'Booking ruangan boleh dimulai paling cepat sekian menit sebelum jadwal')
ON CONFLICT (key) DO NOTHING;
