-- ─── SUPIR TETAP KENDARAAN ──────────────────────────────────────────────────
-- Pasangan tetap opsional 1:1 antara kendaraan dan supir, terpisah dari
-- driver_assignments (yang cuma mencatat siapa SEDANG memegang kendaraan
-- lewat booking aktif, dilepas saat booking selesai). fixedDriverId di sini
-- permanen sampai admin ubah - dipakai untuk auto-pilih supir saat kendaraan
-- ini dibooking (tidak ada pilihan supir lain untuk kendaraan ini). UNIQUE
-- memastikan satu supir cuma bisa jadi supir tetap satu kendaraan.
ALTER TABLE vehicles ADD COLUMN "fixedDriverId" INTEGER NULL UNIQUE REFERENCES drivers(id);
CREATE INDEX idx_vehicles_fixed_driver_id ON vehicles("fixedDriverId");

COMMENT ON COLUMN vehicles."fixedDriverId" IS 'Supir tetap kendaraan ini (opsional) - saat terisi, booking kendaraan ini otomatis pakai supir ini, admin tetap bisa alihkan manual saat approve/assign kalau supir berhalangan.';
