-- ─── ROOM RATINGS ───────────────────────────────────────────────────────────
-- Analogous to driver_ratings (bintang 1-5 + ulasan per booking COMPLETED),
-- tapi targetnya RUANGAN, bukan orang - room_keepers adalah role mengambang
-- yang tidak terikat ke ruangan atau booking tertentu (lihat room_keepers),
-- jadi tidak ada satu individu spesifik untuk dinilai per booking ruangan
-- seperti driver. Pemilik booking menilai kondisi/kualitas ruangannya sendiri.
CREATE TABLE room_ratings (
    id          SERIAL      PRIMARY KEY,
    "bookingId" INTEGER     NOT NULL UNIQUE REFERENCES bookings(id) ON DELETE CASCADE,
    "roomId"    INTEGER     NOT NULL REFERENCES rooms(id),
    "ratedById" INTEGER     NOT NULL REFERENCES users(id),
    rating      SMALLINT    NOT NULL CHECK (rating >= 1 AND rating <= 5),
    review      TEXT        NULL,
    "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_room_ratings_room_id  ON room_ratings("roomId");
CREATE INDEX idx_room_ratings_rated_by ON room_ratings("ratedById");

COMMENT ON TABLE  room_ratings             IS 'Penilaian bintang 1-5 + ulasan ruangan oleh pengguna setelah booking selesai';
COMMENT ON COLUMN room_ratings.rating      IS '1 = buruk, 5 = sangat baik - evaluasi kondisi/kualitas ruangan';
COMMENT ON COLUMN room_ratings."bookingId" IS 'UNIQUE - satu booking hanya bisa dirating sekali';

CREATE OR REPLACE VIEW v_room_ratings_summary AS
SELECT
    rm.id                                             AS room_id,
    res.name                                          AS room_name,
    rm.location                                       AS location,
    res.status                                         AS status,
    COUNT(rr.id)                                       AS total_ratings,
    COALESCE(ROUND(AVG(rr.rating)::NUMERIC, 2), 0)     AS average_rating,
    SUM(CASE WHEN rr.rating = 5 THEN 1 ELSE 0 END)     AS bintang_5,
    SUM(CASE WHEN rr.rating = 4 THEN 1 ELSE 0 END)     AS bintang_4,
    SUM(CASE WHEN rr.rating = 3 THEN 1 ELSE 0 END)     AS bintang_3,
    SUM(CASE WHEN rr.rating = 2 THEN 1 ELSE 0 END)     AS bintang_2,
    SUM(CASE WHEN rr.rating = 1 THEN 1 ELSE 0 END)     AS bintang_1
FROM rooms rm
JOIN resources res ON res.id = rm."resourceId"
LEFT JOIN room_ratings rr ON rr."roomId" = rm.id
GROUP BY rm.id, res.name, rm.location, res.status;

COMMENT ON VIEW v_room_ratings_summary IS 'Ringkasan rating ruangan - rata-rata dan breakdown per bintang';
