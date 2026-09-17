-- Kebalikan 000012: kembali ke TIMESTAMP (jam dinding zona sesi). Idempotent
-- dengan alasan yang sama — hanya berjalan selama kolom masih timestamptz.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'notifications'
          AND column_name = 'created_at'
          AND data_type = 'timestamp with time zone'
    ) THEN
        ALTER TABLE notifications
            ALTER COLUMN created_at TYPE TIMESTAMP
                USING created_at AT TIME ZONE current_setting('TimeZone'),
            ALTER COLUMN created_at SET DEFAULT NOW();
    END IF;
END $$;
