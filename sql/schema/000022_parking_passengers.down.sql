DROP TABLE IF EXISTS booking_passengers;
DROP INDEX IF EXISTS idx_trip_expenses_parking;
DROP TABLE IF EXISTS parking_settlements;
ALTER TABLE bookings DROP COLUMN IF EXISTS "parkingAllowance";
