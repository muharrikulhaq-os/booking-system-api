-- Nilai enum 'RETURNED' tidak bisa dihapus dari tipe PostgreSQL; booking
-- RETURNED dikembalikan ke ONGOING supaya kode lama tetap mengenalinya.
UPDATE bookings SET status = 'ONGOING' WHERE status = 'RETURNED';

DELETE FROM master_settings WHERE key IN (
    'booking_start_early_minutes_spd',
    'booking_start_early_minutes_non_spd',
    'booking_start_early_minutes_room'
);

ALTER TABLE bookings
    DROP COLUMN IF EXISTS "pickupLocation",
    DROP COLUMN IF EXISTS "destination";
