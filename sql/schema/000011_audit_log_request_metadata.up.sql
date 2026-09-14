-- Menambahkan metadata request (IP + User-Agent) ke audit_logs, supaya
-- setiap entri log bisa dilacak "dari mana" selain "siapa" - sebelumnya
-- audit_logs cuma menyimpan userId, tidak ada jejak device/browser/IP.
ALTER TABLE audit_logs ADD COLUMN "ipAddress" VARCHAR(64);
ALTER TABLE audit_logs ADD COLUMN "userAgent" TEXT;
