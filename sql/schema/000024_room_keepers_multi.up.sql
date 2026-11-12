-- ─── BANYAK PENJAGA PER RUANGAN ─────────────────────────────────────────────
-- Satu ruangan bisa dijaga beberapa room keeper, dan satu room keeper bisa
-- menjaga beberapa ruangan (N:N). Menggantikan rooms."roomKeeperId" (N:1).
-- Ruangan tanpa penjaga aktif tidak bisa dibooking.
-- Idempoten: CI menjalankan ulang semua migrasi setiap deploy.
CREATE TABLE IF NOT EXISTS room_keeper_rooms (
    "roomId"       INTEGER     NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    "roomKeeperId" INTEGER     NOT NULL REFERENCES room_keepers(id) ON DELETE CASCADE,
    "createdAt"    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY ("roomId", "roomKeeperId")
);
CREATE INDEX IF NOT EXISTS idx_room_keeper_rooms_keeper ON room_keeper_rooms("roomKeeperId");

COMMENT ON TABLE room_keeper_rooms IS 'Penjaga ruangan (N:N) - semua penjaga ruangan menerima booking, notifikasi, dan rating yang sama';

-- Pindahkan penjaga lama, lalu kosongkan kolomnya supaya migrasi yang
-- dijalankan ulang tidak memasukkannya kembali setelah admin mengubahnya.
INSERT INTO room_keeper_rooms ("roomId", "roomKeeperId")
SELECT id, "roomKeeperId" FROM rooms WHERE "roomKeeperId" IS NOT NULL
ON CONFLICT DO NOTHING;
UPDATE rooms SET "roomKeeperId" = NULL WHERE "roomKeeperId" IS NOT NULL;
COMMENT ON COLUMN rooms."roomKeeperId" IS 'TIDAK DIPAKAI sejak migrasi 000024 - lihat tabel room_keeper_rooms';

-- Rating ruangan kini per penjaga: satu booking bisa punya beberapa rating,
-- masing-masing untuk satu penjaga.
ALTER TABLE room_ratings DROP CONSTRAINT IF EXISTS "room_ratings_bookingId_key";
CREATE UNIQUE INDEX IF NOT EXISTS uq_room_ratings_booking_keeper ON room_ratings("bookingId", "roomKeeperId");
CREATE INDEX IF NOT EXISTS idx_room_ratings_booking_id ON room_ratings("bookingId");
COMMENT ON COLUMN room_ratings."bookingId" IS 'Satu rating per (booking, penjaga) - lihat uq_room_ratings_booking_keeper';
