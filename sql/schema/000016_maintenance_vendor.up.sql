-- Maintenance oleh vendor/bengkel luar (docs/RANCANGAN_MAINTENANCE_VENDOR.md).
-- CI menjalankan ulang semua *.up.sql di setiap deploy → semua DDL idempoten;
-- perpindahan data lama dijaga penanda di data_fixes.

-- ─── Vendor / bengkel ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS vendors (
    id          SERIAL       PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    type        VARCHAR(20)  NOT NULL DEFAULT 'WORKSHOP'
                CHECK (type IN ('OWNER', 'WORKSHOP', 'BOTH')),
    address     TEXT         NULL,
    "picName"   VARCHAR(255) NULL,
    phone       VARCHAR(50)  NULL,
    email       VARCHAR(255) NULL,
    note        TEXT         NULL,
    "isActive"  BOOLEAN      NOT NULL DEFAULT TRUE,
    "createdAt" TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    "updatedAt" TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_vendors_name ON vendors (LOWER(name));

-- ─── Kepemilikan kendaraan ───────────────────────────────────────────────────
ALTER TABLE vehicles ADD COLUMN IF NOT EXISTS ownership VARCHAR(20) NOT NULL DEFAULT 'COMPANY';
ALTER TABLE vehicles ADD COLUMN IF NOT EXISTS "ownerVendorId" INTEGER NULL REFERENCES vendors(id);
ALTER TABLE vehicles ADD COLUMN IF NOT EXISTS "rentalContractNo" VARCHAR(100) NULL;
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_vehicles_ownership') THEN
        ALTER TABLE vehicles ADD CONSTRAINT chk_vehicles_ownership CHECK (
            (ownership = 'COMPANY') OR (ownership = 'VENDOR' AND "ownerVendorId" IS NOT NULL));
    END IF;
END $$;

-- ─── Pengaturan dokumen (kop surat & penandatangan) — satu baris ─────────────
CREATE TABLE IF NOT EXISTS document_settings (
    id                 INTEGER      PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    "companyName"      VARCHAR(255) NOT NULL DEFAULT '',
    "companyAddress"   TEXT         NOT NULL DEFAULT '',
    "companyPhone"     VARCHAR(100) NOT NULL DEFAULT '',
    "companyEmail"     VARCHAR(255) NOT NULL DEFAULT '',
    "logoUrl"          TEXT         NULL,
    "signerName"       VARCHAR(255) NOT NULL DEFAULT '',
    "signerTitle"      VARCHAR(255) NOT NULL DEFAULT '',
    "letterCode"       VARCHAR(50)  NOT NULL DEFAULT 'KCE-MNT',
    "updatedAt"        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
INSERT INTO document_settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING;

-- Penomoran surat berurutan per tahun (UPSERT ... RETURNING = atomik).
CREATE TABLE IF NOT EXISTS document_counters (
    key   VARCHAR(50) PRIMARY KEY,
    value INTEGER     NOT NULL DEFAULT 0
);

-- ─── Laporan kerusakan kendaraan (supir) ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS vehicle_issue_reports (
    id              SERIAL       PRIMARY KEY,
    "vehicleId"     INTEGER      NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    "bookingId"     INTEGER      NULL REFERENCES bookings(id) ON DELETE SET NULL,
    "reportedById"  INTEGER      NOT NULL REFERENCES users(id),
    description     TEXT         NOT NULL,
    location        VARCHAR(255) NULL,
    photos          JSONB        NULL,
    "canContinue"   BOOLEAN      NOT NULL DEFAULT TRUE,
    status          VARCHAR(20)  NOT NULL DEFAULT 'OPEN'
                    CHECK (status IN ('OPEN', 'CONVERTED', 'DISMISSED')),
    "handledById"   INTEGER      NULL REFERENCES users(id),
    "handledNote"   TEXT         NULL,
    "handledAt"     TIMESTAMPTZ  NULL,
    "maintenanceId" INTEGER      NULL,
    "createdAt"     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_vehicle_issue_reports_status ON vehicle_issue_reports(status);
CREATE INDEX IF NOT EXISTS idx_vehicle_issue_reports_vehicle ON vehicle_issue_reports("vehicleId");

-- ─── Perluasan maintenance ───────────────────────────────────────────────────
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "requestNo"            VARCHAR(60)   NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "vendorId"             INTEGER       NULL REFERENCES vendors(id);
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS category               VARCHAR(30)   NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS complaint              TEXT          NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "plannedDate"          TIMESTAMPTZ   NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "estimatedDays"        INTEGER       NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "scheduledDate"        TIMESTAMPTZ   NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "pickupMethod"         VARCHAR(20)   NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "estimatedCost"        NUMERIC(14,2) NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "costBearer"           VARCHAR(20)   NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "sourceIssueId"        INTEGER       NULL REFERENCES vehicle_issue_reports(id) ON DELETE SET NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "submittedAt"          TIMESTAMPTZ   NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "scheduleNote"         TEXT          NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "handoverAt"           TIMESTAMPTZ   NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "handoverOdometer"     INTEGER       NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "handoverFuelLevel"    VARCHAR(10)   NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "handoverReceiverName" VARCHAR(255)  NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "handoverChecklist"    JSONB         NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "handoverNote"         TEXT          NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "returnedAt"           TIMESTAMPTZ   NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "returnOdometer"       INTEGER       NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "returnFuelLevel"      VARCHAR(10)   NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "returnHandlerName"    VARCHAR(255)  NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "returnChecklist"      JSONB         NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "workDone"             TEXT          NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "partsReplaced"        TEXT          NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "returnNote"           TEXT          NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "cancelledAt"          TIMESTAMPTZ   NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "cancelReason"         TEXT          NULL;
ALTER TABLE maintenance_records ADD COLUMN IF NOT EXISTS "updatedAt"            TIMESTAMPTZ   NOT NULL DEFAULT NOW();
-- Lokasi kini opsional (tujuan = alamat vendor).
ALTER TABLE maintenance_records ALTER COLUMN location DROP NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_maintenance_request_no ON maintenance_records("requestNo") WHERE "requestNo" IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_maintenance_status ON maintenance_records(status);
CREATE INDEX IF NOT EXISTS idx_maintenance_vendor ON maintenance_records("vendorId");

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_issue_reports_maintenance') THEN
        ALTER TABLE vehicle_issue_reports ADD CONSTRAINT fk_issue_reports_maintenance
            FOREIGN KEY ("maintenanceId") REFERENCES maintenance_records(id) ON DELETE SET NULL;
    END IF;
END $$;

-- ─── Dokumen maintenance (invoice, scan bertanda tangan, foto) ───────────────
CREATE TABLE IF NOT EXISTS maintenance_documents (
    id              SERIAL       PRIMARY KEY,
    "maintenanceId" INTEGER      NOT NULL REFERENCES maintenance_records(id) ON DELETE CASCADE,
    kind            VARCHAR(30)  NOT NULL
                    CHECK (kind IN ('INVOICE', 'SIGNED_REQUEST', 'SIGNED_HANDOVER', 'SIGNED_RETURN', 'PHOTO', 'OTHER')),
    "fileUrl"       TEXT         NOT NULL,
    "fileName"      VARCHAR(255) NOT NULL,
    "uploadedById"  INTEGER      NOT NULL REFERENCES users(id),
    "createdAt"     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_maintenance_documents_mid ON maintenance_documents("maintenanceId");

-- ─── Perpindahan data lama (sekali) ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS data_fixes (
    key        TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

DO $$
BEGIN
    INSERT INTO data_fixes (key) VALUES ('000016_maintenance_vendor')
    ON CONFLICT (key) DO NOTHING;
    IF NOT FOUND THEN
        RETURN; -- sudah pernah dijalankan
    END IF;

    -- Nama bengkel lama → vendor WORKSHOP (per nama unik).
    INSERT INTO vendors (name, type)
    SELECT DISTINCT ON (LOWER(TRIM("vendorName"))) TRIM("vendorName"), 'WORKSHOP'
    FROM maintenance_records
    WHERE "vendorName" IS NOT NULL AND TRIM("vendorName") <> ''
    ORDER BY LOWER(TRIM("vendorName"))
    ON CONFLICT DO NOTHING;

    UPDATE maintenance_records m SET "vendorId" = v.id
    FROM vendors v
    WHERE m."vendorId" IS NULL AND m."vendorName" IS NOT NULL
      AND LOWER(TRIM(m."vendorName")) = LOWER(v.name);

    -- Status lama: selesai → COMPLETED; lainnya (pending/ongoing) → IN_PROGRESS.
    UPDATE maintenance_records SET
        status = 'COMPLETED',
        "handoverAt" = COALESCE("handoverAt", "startDate"),
        "handoverOdometer" = COALESCE("handoverOdometer", odometer),
        "returnedAt" = COALESCE("returnedAt", "completedAt", "endDate", "startDate")
    WHERE LOWER(status) = 'completed';

    UPDATE maintenance_records SET
        status = 'IN_PROGRESS',
        "handoverAt" = COALESCE("handoverAt", "startDate"),
        "handoverOdometer" = COALESCE("handoverOdometer", odometer),
        "endDate" = NULL
    WHERE status NOT IN ('DRAFT', 'SUBMITTED', 'SCHEDULED', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED');

    UPDATE maintenance_records SET category = CASE LOWER(type)
            WHEN 'routine' THEN 'ROUTINE' WHEN 'repair' THEN 'REPAIR'
            WHEN 'inspection' THEN 'ROUTINE' ELSE 'OTHER' END
    WHERE category IS NULL;
END $$;

ALTER TABLE maintenance_records ALTER COLUMN status SET DEFAULT 'DRAFT';
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_maintenance_status') THEN
        ALTER TABLE maintenance_records ADD CONSTRAINT chk_maintenance_status
            CHECK (status IN ('DRAFT', 'SUBMITTED', 'SCHEDULED', 'IN_PROGRESS', 'COMPLETED', 'CANCELLED'));
    END IF;
END $$;
