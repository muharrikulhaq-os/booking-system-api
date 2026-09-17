-- notifications.created_at dibuat sebagai TIMESTAMP (tanpa zona) di 000001,
-- satu-satunya kolom waktu yang tidak timestamptz. DEFAULT NOW() menyimpan JAM
-- DINDING zona server DB (mis. 23:25 WIB), tetapi driver membacanya sebagai
-- UTC, sehingga API mengirim "23:25Z" — 7 jam lebih maju dari kejadian aslinya.
--
-- Nilai lama ditafsirkan ulang dalam zona sesi saat migrasi dijalankan. CI
-- menjalankan migrasi lewat psql tanpa mengubah zona sesi, jadi zona yang
-- dipakai = zona default server — zona yang sama yang dipakai NOW() ketika
-- baris-baris itu ditulis (koneksi aplikasi dulu juga tidak mengubahnya).
--
-- WAJIB idempotent: CI menjalankan SEMUA *.up.sql pada setiap deploy.
-- ALTER ... TYPE ... USING yang diulang pada kolom yang SUDAH timestamptz
-- tidak gagal, melainkan menggeser nilainya lagi setiap deploy. Karena itu
-- konversi hanya dilakukan selama kolom masih bertipe TIMESTAMP.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'notifications'
          AND column_name = 'created_at'
          AND data_type = 'timestamp without time zone'
    ) THEN
        ALTER TABLE notifications
            ALTER COLUMN created_at TYPE TIMESTAMPTZ
                USING created_at AT TIME ZONE current_setting('TimeZone'),
            ALTER COLUMN created_at SET DEFAULT NOW();
    END IF;
END $$;
