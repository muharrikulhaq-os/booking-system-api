-- ─── ROOM KEEPER ASSIGNMENT ─────────────────────────────────────────────────
-- Each room can have one room keeper responsible for it, but unlike the
-- vehicle<->driver fixed pairing (strict 1:1), a room keeper CAN be
-- responsible for more than one room - so this is N:1 (many rooms, one
-- room keeper each), no UNIQUE constraint on "roomKeeperId".
ALTER TABLE rooms ADD COLUMN "roomKeeperId" INTEGER NULL REFERENCES room_keepers(id);
CREATE INDEX idx_rooms_room_keeper_id ON rooms("roomKeeperId");

COMMENT ON COLUMN rooms."roomKeeperId" IS 'Room keeper penanggung jawab ruangan ini (opsional) - satu room keeper boleh bertanggung jawab atas banyak ruangan.';

-- Room ratings now target the room KEEPER, not the room itself - rating
-- evaluates the person's service, not the room's physical condition.
-- Nullable: a booking for a room with no assigned keeper can still be
-- rated (keeps the "sudah dinilai" state), it just won't count toward
-- anyone's summary.
ALTER TABLE room_ratings ADD COLUMN "roomKeeperId" INTEGER NULL REFERENCES room_keepers(id);
CREATE INDEX idx_room_ratings_room_keeper_id ON room_ratings("roomKeeperId");
