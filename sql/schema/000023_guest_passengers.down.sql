DELETE FROM booking_passengers WHERE "userId" IS NULL;
ALTER TABLE booking_passengers DROP CONSTRAINT IF EXISTS booking_passengers_user_or_guest;
ALTER TABLE booking_passengers DROP COLUMN IF EXISTS "guestNote";
ALTER TABLE booking_passengers DROP COLUMN IF EXISTS "guestPhone";
ALTER TABLE booking_passengers DROP COLUMN IF EXISTS "guestName";
ALTER TABLE booking_passengers ALTER COLUMN "userId" SET NOT NULL;
