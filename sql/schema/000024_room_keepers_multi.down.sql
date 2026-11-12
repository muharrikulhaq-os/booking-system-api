-- Kembali ke satu penjaga per ruangan: ambil penjaga dengan id terkecil.
UPDATE rooms rm SET "roomKeeperId" = k.keeper
FROM (SELECT "roomId", MIN("roomKeeperId") AS keeper FROM room_keeper_rooms GROUP BY "roomId") k
WHERE k."roomId" = rm.id;
DROP TABLE IF EXISTS room_keeper_rooms;

-- Hanya bisa dikembalikan bila tiap booking punya paling banyak satu rating.
DROP INDEX IF EXISTS idx_room_ratings_booking_id;
DROP INDEX IF EXISTS uq_room_ratings_booking_keeper;
ALTER TABLE room_ratings ADD CONSTRAINT "room_ratings_bookingId_key" UNIQUE ("bookingId");
