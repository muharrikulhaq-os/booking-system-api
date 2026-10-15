-- Nomor telepon untuk SEMUA user (terutama karyawan/pemohon booking) supaya
-- supir/admin bisa menelepon pemohon, mis. tidak bertemu di titik jemput.
-- Supir & penjaga ruangan sudah punya nomor di tabel masing-masing - disalin
-- ke sini sebagai nilai awal; selanjutnya form user menulis ke keduanya.
ALTER TABLE users ADD COLUMN IF NOT EXISTS "phoneNumber" VARCHAR(20) NULL;

UPDATE users u SET "phoneNumber" = d."phoneNumber"
FROM drivers d
WHERE d."userId" = u.id AND u."phoneNumber" IS NULL AND d."phoneNumber" <> '';

UPDATE users u SET "phoneNumber" = rk."phoneNumber"
FROM room_keepers rk
WHERE rk."userId" = u.id AND u."phoneNumber" IS NULL AND rk."phoneNumber" <> '';
