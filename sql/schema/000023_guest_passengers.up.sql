-- Tamu tanpa akun sebagai penumpang booking (nama + nomor HP + keterangan).
-- Baris tamu: "userId" NULL. Idempoten.
ALTER TABLE booking_passengers ALTER COLUMN "userId" DROP NOT NULL;
ALTER TABLE booking_passengers ADD COLUMN IF NOT EXISTS "guestName"  VARCHAR(120) NULL;
ALTER TABLE booking_passengers ADD COLUMN IF NOT EXISTS "guestPhone" VARCHAR(20)  NULL;
ALTER TABLE booking_passengers ADD COLUMN IF NOT EXISTS "guestNote"  VARCHAR(120) NULL;
DO $$ BEGIN
    ALTER TABLE booking_passengers ADD CONSTRAINT booking_passengers_user_or_guest
        CHECK ("userId" IS NOT NULL OR "guestName" IS NOT NULL);
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
