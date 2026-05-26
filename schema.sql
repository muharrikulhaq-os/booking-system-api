-- ═══════════════════════════════════════════════════════════════════════════════
-- RESOURCE BOOKING SYSTEM — Schema v2
-- Database  : PostgreSQL (Supabase / self-hosted)
-- Updated   : 2026-02-24
--
-- Perubahan dari v1:
--   [REQ 1]  vehicles      → +capacity (kapasitas penumpang)
--   [REQ 2]  bookings      → +assignedDriverId, assignedVehicleId, assignedAt
--   [REQ 3]  roles         → APPROVER dihapus (EMPLOYEE, ADMIN, DRIVER)
--   [REQ 4]  vehicle_categories sudah ada, seed diperluas
--   [REQ 5]  driver_ratings → tabel BARU (rating 1–5 + ulasan per booking)
--   [REQ 7]  fuel_expenses  → redesign: support BBM & LISTRIK (fuelType enum)
--   [REQ 8]  master_settings → tabel BARU (harga BBM/kWh default)
--   [REQ 10] attachments   → bisa diupload semua user (bukan admin-only)
--   [REQ 11] views         → laporan BBM, listrik, maintenance
--   Lain:    users +profilePhoto, guest_bookings & attachments inline
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE EXTENSION IF NOT EXISTS "pgcrypto";


-- ═══════════════════════════════════════════════════════════════════════════════
-- ENUMS
-- ═══════════════════════════════════════════════════════════════════════════════

-- [REQ 3] APPROVER dihapus — Admin saja yang approve
CREATE TYPE role_name AS ENUM (
    'EMPLOYEE',
    'ADMIN',
    'DRIVER'
);

CREATE TYPE resource_type AS ENUM (
    'VEHICLE',
    'ROOM'
);

CREATE TYPE resource_status AS ENUM (
    'AVAILABLE',
    'MAINTENANCE',
    'INACTIVE'
);

CREATE TYPE booking_status AS ENUM (
    'PENDING',
    'APPROVED',
    'REJECTED',
    'ONGOING',
    'COMPLETED',
    'CANCELLED',
    'OVERDUE'
);

CREATE TYPE approval_action AS ENUM (
    'APPROVED',
    'REJECTED'
);

-- [REQ 7] Tipe bahan bakar
CREATE TYPE fuel_type AS ENUM (
    'BBM',      -- Bensin / Solar
    'LISTRIK'   -- Kendaraan listrik / SPKLU
);


-- ═══════════════════════════════════════════════════════════════════════════════
-- ROLES
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE roles (
    id   SERIAL    PRIMARY KEY,
    name role_name NOT NULL UNIQUE
);

COMMENT ON TABLE  roles      IS 'Role user: EMPLOYEE | ADMIN | DRIVER';
COMMENT ON COLUMN roles.name IS 'APPROVER dihapus — Admin handles semua approval';


-- ═══════════════════════════════════════════════════════════════════════════════
-- DEPARTMENTS
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE departments (
    id          SERIAL       PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE,
    "createdAt" TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE departments IS 'Divisi / departemen perusahaan';


-- ═══════════════════════════════════════════════════════════════════════════════
-- USERS
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE users (
    id             SERIAL       PRIMARY KEY,
    "employeeId"   VARCHAR(50)  NOT NULL UNIQUE,
    name           VARCHAR(150) NOT NULL,
    email          VARCHAR(255) NOT NULL UNIQUE,
    password       VARCHAR(255) NOT NULL,
    "profilePhoto" VARCHAR(500) NULL,
    "isActive"     BOOLEAN      NOT NULL DEFAULT TRUE,
    "roleId"       INTEGER      NOT NULL REFERENCES roles(id),
    "departmentId" INTEGER      NOT NULL REFERENCES departments(id),
    "createdAt"    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    "updatedAt"    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email         ON users(email);
CREATE INDEX idx_users_employee_id   ON users("employeeId");
CREATE INDEX idx_users_role_id       ON users("roleId");
CREATE INDEX idx_users_department_id ON users("departmentId");
CREATE INDEX idx_users_is_active     ON users("isActive");

COMMENT ON TABLE  users               IS 'Semua user sistem (employee, admin, driver)';
COMMENT ON COLUMN users."employeeId"  IS 'ID karyawan unik, e.g. EMP001';
COMMENT ON COLUMN users."profilePhoto" IS 'URL atau path foto profil';
COMMENT ON COLUMN users."isActive"    IS 'FALSE = tidak bisa login';


-- ═══════════════════════════════════════════════════════════════════════════════
-- AUTH
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE refresh_tokens (
    id          SERIAL      PRIMARY KEY,
    "userId"    INTEGER     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       TEXT        NOT NULL UNIQUE,
    "expiresAt" TIMESTAMPTZ NOT NULL,
    revoked     BOOLEAN     NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens("userId");
CREATE INDEX idx_refresh_tokens_token   ON refresh_tokens(token);
CREATE INDEX idx_refresh_tokens_revoked ON refresh_tokens(revoked);

COMMENT ON TABLE refresh_tokens IS 'JWT refresh token — satu baris per sesi aktif';


CREATE TABLE password_reset_otps (
    id          SERIAL      PRIMARY KEY,
    "userId"    INTEGER     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    "otpCode"   VARCHAR(10) NOT NULL,
    "expiresAt" TIMESTAMPTZ NOT NULL,
    "isUsed"    BOOLEAN     NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_otp_user_id ON password_reset_otps("userId");
CREATE INDEX idx_otp_is_used ON password_reset_otps("isUsed");

COMMENT ON TABLE password_reset_otps IS 'OTP 6-digit untuk reset password';


-- ═══════════════════════════════════════════════════════════════════════════════
-- RESOURCES (abstraksi parent)
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE resources (
    id          SERIAL          PRIMARY KEY,
    name        VARCHAR(200)    NOT NULL,
    type        resource_type   NOT NULL,
    status      resource_status NOT NULL DEFAULT 'AVAILABLE',
    "createdAt" TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    "updatedAt" TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_resources_type   ON resources(type);
CREATE INDEX idx_resources_status ON resources(status);

COMMENT ON TABLE resources IS 'Parent abstrak dari vehicles dan rooms';


-- ═══════════════════════════════════════════════════════════════════════════════
-- VEHICLE CATEGORIES  [REQ 4]
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE vehicle_categories (
    id   SERIAL       PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE
);

COMMENT ON TABLE vehicle_categories IS 'Kategori kendaraan: MPV, SUV, Sedan, Pickup, Bus, EV, dll.';


-- ═══════════════════════════════════════════════════════════════════════════════
-- VEHICLES  [REQ 1 — +capacity]
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE vehicles (
    id                SERIAL       PRIMARY KEY,
    "resourceId"      INTEGER      NOT NULL UNIQUE REFERENCES resources(id) ON DELETE CASCADE,
    "plateNumber"     VARCHAR(20)  NOT NULL UNIQUE,
    brand             VARCHAR(100) NOT NULL,
    model             VARCHAR(100) NOT NULL,
    year              SMALLINT     NOT NULL CHECK (year >= 1900 AND year <= 2100),
    "currentOdometer" INTEGER      NOT NULL DEFAULT 0 CHECK ("currentOdometer" >= 0),
    "categoryId"      INTEGER      NOT NULL REFERENCES vehicle_categories(id),
    capacity          SMALLINT     NOT NULL DEFAULT 4 CHECK (capacity > 0)
);

CREATE INDEX idx_vehicles_plate_number ON vehicles("plateNumber");
CREATE INDEX idx_vehicles_category_id  ON vehicles("categoryId");

COMMENT ON TABLE  vehicles          IS 'Detail kendaraan — relasi 1:1 ke resources';
COMMENT ON COLUMN vehicles.capacity IS '[REQ 1] Kapasitas maksimal penumpang';


-- ═══════════════════════════════════════════════════════════════════════════════
-- ROOMS
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE rooms (
    id           SERIAL       PRIMARY KEY,
    "resourceId" INTEGER      NOT NULL UNIQUE REFERENCES resources(id) ON DELETE CASCADE,
    location     VARCHAR(255) NOT NULL,
    capacity     SMALLINT     NOT NULL CHECK (capacity > 0)
);

COMMENT ON TABLE rooms IS 'Detail ruang rapat — relasi 1:1 ke resources';


-- ═══════════════════════════════════════════════════════════════════════════════
-- DRIVERS
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE drivers (
    id              SERIAL       PRIMARY KEY,
    "userId"        INTEGER      NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    "licenseNumber" VARCHAR(100) NOT NULL,
    "phoneNumber"   VARCHAR(20)  NOT NULL,
    "isActive"      BOOLEAN      NOT NULL DEFAULT TRUE,
    "createdAt"     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_drivers_user_id   ON drivers("userId");
CREATE INDEX idx_drivers_is_active ON drivers("isActive");

COMMENT ON TABLE drivers IS 'Profil driver — extend user dengan role DRIVER';


-- ═══════════════════════════════════════════════════════════════════════════════
-- BOOKINGS  [REQ 2 & 6 — +assigned fields]
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE bookings (
    id                   SERIAL         PRIMARY KEY,
    "userId"             INTEGER        NOT NULL REFERENCES users(id),
    "resourceId"         INTEGER        NOT NULL REFERENCES resources(id),
    "startDate"          TIMESTAMPTZ    NOT NULL,
    "endDate"            TIMESTAMPTZ    NOT NULL,
    purpose              TEXT           NOT NULL,
    status               booking_status NOT NULL DEFAULT 'PENDING',
    "approvedById"       INTEGER        REFERENCES users(id),
    "approvedAt"         TIMESTAMPTZ,
    -- [REQ 2 & 6] Admin memilih kendaraan + driver setelah approve
    "assignedDriverId"   INTEGER        REFERENCES drivers(id),
    "assignedVehicleId"  INTEGER        REFERENCES vehicles(id),
    "assignedAt"         TIMESTAMPTZ,
    "returnedAt"         TIMESTAMPTZ,
    "createdAt"          TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    "updatedAt"          TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_booking_dates CHECK ("endDate" > "startDate")
);

CREATE INDEX idx_bookings_user_id          ON bookings("userId");
CREATE INDEX idx_bookings_resource_id      ON bookings("resourceId");
CREATE INDEX idx_bookings_status           ON bookings(status);
CREATE INDEX idx_bookings_start_date       ON bookings("startDate");
CREATE INDEX idx_bookings_end_date         ON bookings("endDate");
CREATE INDEX idx_bookings_approved_by      ON bookings("approvedById");
CREATE INDEX idx_bookings_assigned_driver  ON bookings("assignedDriverId");
CREATE INDEX idx_bookings_assigned_vehicle ON bookings("assignedVehicleId");

CREATE INDEX idx_bookings_active ON bookings("resourceId", "startDate", "endDate")
    WHERE status IN ('PENDING', 'APPROVED', 'ONGOING');

COMMENT ON TABLE  bookings                     IS 'Booking resource — lifecycle PENDING → COMPLETED';
COMMENT ON COLUMN bookings."assignedDriverId"  IS '[REQ 2] Driver yang dipilih admin setelah approve';
COMMENT ON COLUMN bookings."assignedVehicleId" IS '[REQ 2] Kendaraan spesifik yang dipilih admin';
COMMENT ON COLUMN bookings."assignedAt"        IS 'Waktu admin melakukan assignment';
COMMENT ON COLUMN bookings."returnedAt"        IS 'Waktu aktual kendaraan dikembalikan';


-- ─── APPROVAL LOGS ────────────────────────────────────────────────────────────

CREATE TABLE approval_logs (
    id           SERIAL          PRIMARY KEY,
    "bookingId"  INTEGER         NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    "approverId" INTEGER         NOT NULL REFERENCES users(id),
    action       approval_action NOT NULL,
    note         TEXT,
    "createdAt"  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_approval_logs_booking_id  ON approval_logs("bookingId");
CREATE INDEX idx_approval_logs_approver_id ON approval_logs("approverId");

COMMENT ON TABLE approval_logs IS 'Riwayat aksi approve/reject pada booking';


-- ═══════════════════════════════════════════════════════════════════════════════
-- DRIVER ASSIGNMENTS
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE driver_assignments (
    id           SERIAL      PRIMARY KEY,
    "driverId"   INTEGER     NOT NULL REFERENCES drivers(id),
    "vehicleId"  INTEGER     NOT NULL REFERENCES vehicles(id),
    "assignedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    "releasedAt" TIMESTAMPTZ
);

CREATE INDEX idx_driver_assignments_driver_id  ON driver_assignments("driverId");
CREATE INDEX idx_driver_assignments_vehicle_id ON driver_assignments("vehicleId");

CREATE UNIQUE INDEX idx_driver_assignments_active_driver
    ON driver_assignments("driverId") WHERE "releasedAt" IS NULL;

CREATE UNIQUE INDEX idx_driver_assignments_active_vehicle
    ON driver_assignments("vehicleId") WHERE "releasedAt" IS NULL;

COMMENT ON TABLE  driver_assignments              IS 'Riwayat penugasan permanen driver ke kendaraan';
COMMENT ON COLUMN driver_assignments."releasedAt" IS 'NULL = masih aktif ditugaskan';


-- ═══════════════════════════════════════════════════════════════════════════════
-- DRIVER RATINGS  [REQ 5]
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE driver_ratings (
    id          SERIAL      PRIMARY KEY,
    "bookingId" INTEGER     NOT NULL UNIQUE REFERENCES bookings(id) ON DELETE CASCADE,
    "driverId"  INTEGER     NOT NULL REFERENCES drivers(id),
    "ratedById" INTEGER     NOT NULL REFERENCES users(id),
    rating      SMALLINT    NOT NULL CHECK (rating >= 1 AND rating <= 5),
    review      TEXT        NULL,
    "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_driver_ratings_driver_id ON driver_ratings("driverId");
CREATE INDEX idx_driver_ratings_rated_by  ON driver_ratings("ratedById");

COMMENT ON TABLE  driver_ratings             IS '[REQ 5] Penilaian bintang 1–5 + ulasan driver oleh pengguna';
COMMENT ON COLUMN driver_ratings.rating      IS '1 = buruk, 5 = sangat baik — untuk evaluasi driver';
COMMENT ON COLUMN driver_ratings."bookingId" IS 'UNIQUE — satu booking hanya bisa dirating sekali';


-- ═══════════════════════════════════════════════════════════════════════════════
-- FUEL EXPENSES  [REQ 7] — BBM & LISTRIK
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE fuel_expenses (
    id               SERIAL        PRIMARY KEY,
    "driverId"       INTEGER       NOT NULL REFERENCES drivers(id),
    "vehicleId"      INTEGER       NOT NULL REFERENCES vehicles(id),
    "bookingId"      INTEGER       REFERENCES bookings(id),
    "fuelType"       fuel_type     NOT NULL DEFAULT 'BBM',

    -- BBM fields
    liter            NUMERIC(10,2) NULL CHECK (liter IS NULL OR liter > 0),
    "pricePerLiter"  NUMERIC(12,2) NULL CHECK ("pricePerLiter" IS NULL OR "pricePerLiter" > 0),
    "odometerBefore" INTEGER       NULL CHECK ("odometerBefore" IS NULL OR "odometerBefore" >= 0),
    "odometerAfter"  INTEGER       NULL,

    -- LISTRIK fields
    kwh              NUMERIC(10,2) NULL CHECK (kwh IS NULL OR kwh > 0),
    "pricePerKwh"    NUMERIC(12,2) NULL CHECK ("pricePerKwh" IS NULL OR "pricePerKwh" > 0),
    "batteryBefore"  NUMERIC(5,2)  NULL CHECK ("batteryBefore" IS NULL OR ("batteryBefore" >= 0 AND "batteryBefore" <= 100)),
    "batteryAfter"   NUMERIC(5,2)  NULL CHECK ("batteryAfter"  IS NULL OR ("batteryAfter"  >= 0 AND "batteryAfter"  <= 100)),

    -- Common
    "totalAmount"    NUMERIC(14,2) NOT NULL CHECK ("totalAmount" > 0),
    note             TEXT,
    "createdAt"      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_bbm_required CHECK (
        "fuelType" <> 'BBM' OR (
            liter IS NOT NULL AND "pricePerLiter" IS NOT NULL AND
            "odometerBefore" IS NOT NULL AND "odometerAfter" IS NOT NULL AND
            "odometerAfter" > "odometerBefore"
        )
    ),
    CONSTRAINT chk_listrik_required CHECK (
        "fuelType" <> 'LISTRIK' OR (
            kwh IS NOT NULL AND "pricePerKwh" IS NOT NULL
        )
    )
);

CREATE INDEX idx_fuel_expenses_driver_id  ON fuel_expenses("driverId");
CREATE INDEX idx_fuel_expenses_vehicle_id ON fuel_expenses("vehicleId");
CREATE INDEX idx_fuel_expenses_booking_id ON fuel_expenses("bookingId");
CREATE INDEX idx_fuel_expenses_fuel_type  ON fuel_expenses("fuelType");
CREATE INDEX idx_fuel_expenses_created_at ON fuel_expenses("createdAt");

COMMENT ON TABLE  fuel_expenses                IS '[REQ 7] Pengeluaran BBM & Listrik yang diinput driver';
COMMENT ON COLUMN fuel_expenses."fuelType"     IS 'BBM = bensin/solar | LISTRIK = pengisian EV (SPKLU)';
COMMENT ON COLUMN fuel_expenses.liter          IS 'Jumlah liter — hanya BBM';
COMMENT ON COLUMN fuel_expenses.kwh            IS 'Jumlah kWh — hanya LISTRIK';
COMMENT ON COLUMN fuel_expenses."batteryBefore" IS 'Persentase baterai sebelum charge (%) — hanya LISTRIK';
COMMENT ON COLUMN fuel_expenses."batteryAfter"  IS 'Persentase baterai setelah charge (%) — hanya LISTRIK';
COMMENT ON COLUMN fuel_expenses."totalAmount"  IS 'liter × pricePerLiter ATAU kWh × pricePerKwh';


-- ═══════════════════════════════════════════════════════════════════════════════
-- MASTER SETTINGS  [REQ 8]
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE master_settings (
    id          SERIAL        PRIMARY KEY,
    key         VARCHAR(100)  NOT NULL UNIQUE,
    value       NUMERIC(14,4) NOT NULL,
    unit        VARCHAR(50)   NULL,
    description TEXT          NULL,
    "updatedAt" TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_master_settings_key ON master_settings(key);

COMMENT ON TABLE  master_settings             IS '[REQ 8] Konfigurasi global yang hanya bisa diubah Admin';
COMMENT ON COLUMN master_settings.key         IS 'Kunci unik, e.g. price_per_liter_bbm / price_per_kwh_listrik';
COMMENT ON COLUMN master_settings.value       IS 'Nilai numerik';
COMMENT ON COLUMN master_settings.unit        IS 'Satuan, e.g. IDR/liter atau IDR/kWh';


-- ═══════════════════════════════════════════════════════════════════════════════
-- MAINTENANCE RECORDS
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE maintenance_records (
    id            SERIAL        PRIMARY KEY,
    "resourceId"  INTEGER       NOT NULL REFERENCES resources(id),
    description   TEXT          NOT NULL,
    "startDate"   TIMESTAMPTZ   NOT NULL,
    "endDate"     TIMESTAMPTZ,
    cost          NUMERIC(12,2) CHECK (cost >= 0),
    "createdById" INTEGER       NOT NULL REFERENCES users(id),
    "createdAt"   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_maintenance_dates CHECK (
        "endDate" IS NULL OR "endDate" > "startDate"
    )
);

CREATE INDEX idx_maintenance_resource_id   ON maintenance_records("resourceId");
CREATE INDEX idx_maintenance_created_by_id ON maintenance_records("createdById");
CREATE INDEX idx_maintenance_start_date    ON maintenance_records("startDate");

COMMENT ON TABLE  maintenance_records           IS 'Catatan servis/perawatan kendaraan & ruangan';
COMMENT ON COLUMN maintenance_records."endDate" IS 'NULL = masih dalam perawatan';


-- ═══════════════════════════════════════════════════════════════════════════════
-- AUDIT LOGS
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE audit_logs (
    id           SERIAL       PRIMARY KEY,
    "userId"     INTEGER      REFERENCES users(id),
    action       VARCHAR(100) NOT NULL,
    "entityType" VARCHAR(100) NOT NULL,
    "entityId"   INTEGER,
    description  TEXT,
    "createdAt"  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user_id     ON audit_logs("userId");
CREATE INDEX idx_audit_logs_entity_type ON audit_logs("entityType");
CREATE INDEX idx_audit_logs_entity_id   ON audit_logs("entityId");
CREATE INDEX idx_audit_logs_action      ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at  ON audit_logs("createdAt");

COMMENT ON TABLE  audit_logs              IS 'Log immutable semua aksi penting';
COMMENT ON COLUMN audit_logs."userId"     IS 'NULL jika aksi sistem/scheduler';
COMMENT ON COLUMN audit_logs.action       IS 'CREATE|UPDATE|DELETE|APPROVE|REJECT|ASSIGN|START|COMPLETE|RATE_DRIVER|LOGIN|LOGOUT';


-- ═══════════════════════════════════════════════════════════════════════════════
-- GUEST BOOKINGS
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE guest_bookings (
    id               SERIAL         PRIMARY KEY,
    "guestName"      VARCHAR(150)   NOT NULL,
    "guestEmail"     VARCHAR(255)   NOT NULL,
    "guestPhone"     VARCHAR(20)    NOT NULL,
    "departmentName" VARCHAR(100)   NOT NULL,
    "resourceId"     INTEGER        NOT NULL REFERENCES resources(id),
    "startDate"      TIMESTAMPTZ    NOT NULL,
    "endDate"        TIMESTAMPTZ    NOT NULL,
    purpose          TEXT           NOT NULL,
    status           booking_status NOT NULL DEFAULT 'PENDING',
    "accessToken"    VARCHAR(64)    NOT NULL UNIQUE,
    "approvedById"   INTEGER        REFERENCES users(id),
    "approvedAt"     TIMESTAMPTZ,
    "rejectionNote"  TEXT,
    "returnedAt"     TIMESTAMPTZ,
    "createdAt"      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    "updatedAt"      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_guest_dates CHECK ("endDate" > "startDate")
);

CREATE INDEX idx_guest_bookings_token       ON guest_bookings("accessToken");
CREATE INDEX idx_guest_bookings_email       ON guest_bookings("guestEmail");
CREATE INDEX idx_guest_bookings_status      ON guest_bookings(status);
CREATE INDEX idx_guest_bookings_resource_id ON guest_bookings("resourceId");

COMMENT ON TABLE guest_bookings IS 'Booking dari tamu eksternal tanpa akun';


-- ═══════════════════════════════════════════════════════════════════════════════
-- ATTACHMENTS  [REQ 10] — semua user bisa upload
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE TABLE attachments (
    id             SERIAL       PRIMARY KEY,
    "uploadedById" INTEGER      NOT NULL REFERENCES users(id),
    "vehicleId"    INTEGER      REFERENCES vehicles(id) ON DELETE CASCADE,
    "roomId"       INTEGER      REFERENCES rooms(id)    ON DELETE CASCADE,
    "bookingId"    INTEGER      REFERENCES bookings(id) ON DELETE CASCADE,
    "fileUrl"      VARCHAR(500) NOT NULL,
    "fileName"     VARCHAR(255) NOT NULL,
    "fileType"     VARCHAR(100) NOT NULL,
    "fileSize"     INTEGER      NULL,
    description    TEXT         NULL,
    "createdAt"    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_one_target CHECK (
        (CASE WHEN "vehicleId" IS NOT NULL THEN 1 ELSE 0 END +
         CASE WHEN "roomId"    IS NOT NULL THEN 1 ELSE 0 END +
         CASE WHEN "bookingId" IS NOT NULL THEN 1 ELSE 0 END) = 1
    )
);

CREATE INDEX idx_att_vehicle  ON attachments("vehicleId");
CREATE INDEX idx_att_room     ON attachments("roomId");
CREATE INDEX idx_att_booking  ON attachments("bookingId");
CREATE INDEX idx_att_uploader ON attachments("uploadedById");

COMMENT ON TABLE attachments IS '[REQ 10] Lampiran untuk kendaraan/ruangan/booking — bisa diupload semua user';


-- ═══════════════════════════════════════════════════════════════════════════════
-- TRIGGERS
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW."updatedAt" = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER set_updated_at_users
    BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_updated_at_resources
    BEFORE UPDATE ON resources FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_updated_at_bookings
    BEFORE UPDATE ON bookings FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_updated_at_guest_bookings
    BEFORE UPDATE ON guest_bookings FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();

CREATE TRIGGER set_updated_at_master_settings
    BEFORE UPDATE ON master_settings FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();


-- ═══════════════════════════════════════════════════════════════════════════════
-- SEED DATA
-- ═══════════════════════════════════════════════════════════════════════════════

-- ─── Roles ────────────────────────────────────────────────────────────────────
INSERT INTO roles (name) VALUES
    ('EMPLOYEE'),   -- id 1
    ('ADMIN'),      -- id 2
    ('DRIVER');     -- id 3

-- ─── Departments ──────────────────────────────────────────────────────────────
INSERT INTO departments (name) VALUES
    ('Information Technology'),   -- id 1
    ('Human Resources'),          -- id 2
    ('Finance & Accounting'),     -- id 3
    ('Operations'),               -- id 4
    ('Marketing');                -- id 5

-- ─── Users ────────────────────────────────────────────────────────────────────
-- Semua password adalah bcrypt hash dari "Password123!"
-- Kecuali Admin: password masih plaintext "admin" — GANTI sebelum production!
INSERT INTO users ("employeeId", name, email, password, "isActive", "roleId", "departmentId") VALUES
    ('ADM001', 'Admin Utama',     'admin@company.com',          'admin',                                                         TRUE, 2, 1),  -- id 1
    ('EMP001', 'John Doe',        'john.doe@company.com',       '$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewKyNiAYQkGDsQtW', TRUE, 1, 1),  -- id 2
    ('EMP002', 'Jane Smith',      'jane.smith@company.com',     '$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewKyNiAYQkGDsQtW', TRUE, 1, 3),  -- id 3
    ('EMP003', 'Dewi Lestari',    'dewi.lestari@company.com',   '$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewKyNiAYQkGDsQtW', TRUE, 1, 5),  -- id 4
    ('EMP004', 'Reza Firmansyah', 'reza.firmansyah@company.com','$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewKyNiAYQkGDsQtW', TRUE, 1, 4),  -- id 5
    ('DRV001', 'Pak Supir Satu',  'supir1@company.com',         '$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewKyNiAYQkGDsQtW', TRUE, 3, 4),  -- id 6
    ('DRV002', 'Pak Supir Dua',   'supir2@company.com',         '$2b$12$LQv3c1yqBWVHxkd0LHAkCOYz6TtxMQJqhN8/LewKyNiAYQkGDsQtW', TRUE, 3, 4);  -- id 7

-- ─── Vehicle Categories ───────────────────────────────────────────────────────
INSERT INTO vehicle_categories (name) VALUES
    ('MPV'),            -- id 1
    ('SUV'),            -- id 2
    ('Sedan'),          -- id 3
    ('Pickup'),         -- id 4
    ('Bus / Minibus'),  -- id 5
    ('Listrik / EV');   -- id 6

-- ─── Resources — Kendaraan ────────────────────────────────────────────────────
INSERT INTO resources (name, type, status) VALUES
    ('Toyota Avanza - B 1234 XY',   'VEHICLE', 'AVAILABLE'),    -- id 1
    ('Honda CR-V - B 5678 AB',       'VEHICLE', 'AVAILABLE'),    -- id 2
    ('Toyota Fortuner - B 9999 CD',  'VEHICLE', 'AVAILABLE'),    -- id 3
    ('Mitsubishi L300 - B 2222 EF',  'VEHICLE', 'MAINTENANCE'),  -- id 4
    ('Daihatsu Xenia - B 3333 GH',   'VEHICLE', 'AVAILABLE'),    -- id 5
    ('Toyota HiAce - B 4444 IJ',     'VEHICLE', 'INACTIVE'),     -- id 6
    ('Hyundai Ioniq 5 - B 5555 EV',  'VEHICLE', 'AVAILABLE');    -- id 7 (EV)

-- ─── Vehicles (dengan capacity) ───────────────────────────────────────────────
INSERT INTO vehicles ("resourceId", "plateNumber", brand, model, year, "currentOdometer", "categoryId", capacity) VALUES
    (1, 'B 1234 XY', 'Toyota',     'Avanza',    2022, 15000, 1, 7),
    (2, 'B 5678 AB', 'Honda',      'CR-V',      2021, 28500, 2, 5),
    (3, 'B 9999 CD', 'Toyota',     'Fortuner',  2023,  5200, 2, 7),
    (4, 'B 2222 EF', 'Mitsubishi', 'L300',       2020, 72000, 4, 8),
    (5, 'B 3333 GH', 'Daihatsu',   'Xenia',     2022, 18300, 1, 7),
    (6, 'B 4444 IJ', 'Toyota',     'HiAce',     2019, 95000, 5, 15),
    (7, 'B 5555 EV', 'Hyundai',    'Ioniq 5',   2024,  3200, 6, 5);

-- ─── Resources — Ruangan ──────────────────────────────────────────────────────
INSERT INTO resources (name, type, status) VALUES
    ('Meeting Room A - Lt.2',     'ROOM', 'AVAILABLE'),   -- id 8
    ('Meeting Room B - Lt.3',     'ROOM', 'AVAILABLE'),   -- id 9
    ('Board Room - Lt.5',          'ROOM', 'AVAILABLE'),   -- id 10
    ('Training Room - Annex Lt.1', 'ROOM', 'INACTIVE');    -- id 11

-- ─── Rooms ────────────────────────────────────────────────────────────────────
INSERT INTO rooms ("resourceId", location, capacity) VALUES
    ( 8, 'Gedung Utama Lt. 2',  10),
    ( 9, 'Gedung Utama Lt. 3',  20),
    (10, 'Gedung Utama Lt. 5',  50),
    (11, 'Gedung Annex Lt. 1',  30);

-- ─── Drivers ──────────────────────────────────────────────────────────────────
-- userId 6 = DRV001, userId 7 = DRV002
INSERT INTO drivers ("userId", "licenseNumber", "phoneNumber", "isActive") VALUES
    (6, 'SIM-B1-2024-001', '+6281234567890', TRUE),  -- driverId 1
    (7, 'SIM-B1-2024-002', '+6287654321098', TRUE);  -- driverId 2

-- ─── Driver Assignments ───────────────────────────────────────────────────────
INSERT INTO driver_assignments ("driverId", "vehicleId", "assignedAt") VALUES
    (1, 1, NOW() - INTERVAL '30 days'),  -- Pak Supir Satu → Avanza
    (2, 2, NOW() - INTERVAL '15 days');  -- Pak Supir Dua  → CR-V

-- ─── Master Settings [REQ 8] ──────────────────────────────────────────────────
INSERT INTO master_settings (key, value, unit, description) VALUES
    ('price_per_liter_bbm',   10000.0000, 'IDR/liter', 'Harga bensin Pertalite per liter (default driver bisa override)'),
    ('price_per_kwh_listrik',  2466.0000, 'IDR/kWh',   'Tarif listrik PLN per kWh — tarif R-2 non-subsidi');

-- ─── Bookings ─────────────────────────────────────────────────────────────────
INSERT INTO bookings (
    "userId", "resourceId", "startDate", "endDate", purpose, status,
    "approvedById", "approvedAt",
    "assignedDriverId", "assignedVehicleId", "assignedAt",
    "returnedAt"
) VALUES
    -- [1] COMPLETED — John Doe, Avanza, Pak Supir Satu
    (2, 1,
     NOW() - INTERVAL '10 days', NOW() - INTERVAL '9 days',
     'Kunjungan klien ke site proyek', 'COMPLETED',
     1, NOW() - INTERVAL '11 days',
     1, 1, NOW() - INTERVAL '11 days',
     NOW() - INTERVAL '9 days'),

    -- [2] APPROVED — Jane Smith, Meeting Room B (ruangan, tidak perlu assign)
    (3, 9,
     NOW() + INTERVAL '2 days', NOW() + INTERVAL '2 days' + INTERVAL '3 hours',
     'Rapat koordinasi tim Finance Q1', 'APPROVED',
     1, NOW() - INTERVAL '1 day',
     NULL, NULL, NULL, NULL),

    -- [3] PENDING — John Doe, Fortuner (menunggu admin assign)
    (2, 3,
     NOW() + INTERVAL '5 days', NOW() + INTERVAL '6 days',
     'Perjalanan dinas ke Bandung', 'PENDING',
     NULL, NULL, NULL, NULL, NULL, NULL),

    -- [4] PENDING — Dewi, Board Room (ruangan)
    (4, 10,
     NOW() + INTERVAL '3 days', NOW() + INTERVAL '3 days' + INTERVAL '4 hours',
     'Presentasi Marketing Campaign Q2', 'PENDING',
     NULL, NULL, NULL, NULL, NULL, NULL),

    -- [5] REJECTED — Reza, Avanza
    (5, 1,
     NOW() - INTERVAL '5 days', NOW() - INTERVAL '4 days',
     'Acara keluarga (bukan keperluan kantor)', 'REJECTED',
     1, NOW() - INTERVAL '6 days',
     NULL, NULL, NULL, NULL),

    -- [6] ONGOING — John Doe, Xenia, Pak Supir Dua
    (2, 5,
     NOW() - INTERVAL '1 hour', NOW() + INTERVAL '6 hours',
     'Antar dokumen ke kantor pusat', 'ONGOING',
     1, NOW() - INTERVAL '2 days',
     2, 5, NOW() - INTERVAL '2 days',
     NULL),

    -- [7] OVERDUE — Reza, CR-V
    (5, 2,
     NOW() - INTERVAL '3 days', NOW() - INTERVAL '1 day',
     'Perjalanan survey lokasi', 'OVERDUE',
     1, NOW() - INTERVAL '4 days',
     NULL, NULL, NULL, NULL),

    -- [8] CANCELLED — Jane Smith, Meeting Room A
    (3, 8,
     NOW() + INTERVAL '1 day', NOW() + INTERVAL '1 day' + INTERVAL '2 hours',
     'Meeting yang dibatalkan', 'CANCELLED',
     NULL, NULL, NULL, NULL, NULL, NULL),

    -- [9] APPROVED + assigned — Dewi, Ioniq 5 (EV), Pak Supir Satu
    (4, 7,
     NOW() + INTERVAL '1 day', NOW() + INTERVAL '2 days',
     'Kunjungan ke pameran EV Jakarta', 'APPROVED',
     1, NOW() - INTERVAL '12 hours',
     1, 7, NOW() - INTERVAL '12 hours',
     NULL);

-- ─── Approval Logs ────────────────────────────────────────────────────────────
INSERT INTO approval_logs ("bookingId", "approverId", action, note) VALUES
    (1, 1, 'APPROVED', 'Disetujui — keperluan klien prioritas'),
    (2, 1, 'APPROVED', 'OK, silakan'),
    (5, 1, 'REJECTED', 'Booking untuk keperluan pribadi tidak diizinkan'),
    (7, 1, 'APPROVED', 'Disetujui untuk survey lokasi proyek'),
    (9, 1, 'APPROVED', 'Disetujui — kendaraan listrik tersedia');

-- ─── Fuel Expenses [REQ 7] ────────────────────────────────────────────────────

-- BBM
INSERT INTO fuel_expenses (
    "driverId", "vehicleId", "bookingId", "fuelType",
    liter, "pricePerLiter", "odometerBefore", "odometerAfter",
    "totalAmount", note
) VALUES
    (1, 1, 1, 'BBM', 40.50, 10000.00, 14600, 15000, 405000.00, 'SPBU Pertamina Jl. Sudirman'),
    (1, 1, 6, 'BBM', 35.00, 10000.00, 15000, 15320, 350000.00, 'SPBU Shell Jl. Gatot Subroto'),
    (2, 2, 7, 'BBM', 50.00, 10200.00, 28000, 28500, 510000.00, 'SPBU Pertamina Bekasi');

-- LISTRIK
INSERT INTO fuel_expenses (
    "driverId", "vehicleId", "bookingId", "fuelType",
    kwh, "pricePerKwh", "batteryBefore", "batteryAfter",
    "totalAmount", note
) VALUES
    (1, 7, 9, 'LISTRIK', 45.00, 2466.00, 20.00, 95.00, 110970.00, 'SPKLU PLN Kemayoran — charge 75%');

-- ─── Driver Ratings [REQ 5] ───────────────────────────────────────────────────
-- Booking #1 sudah COMPLETED, John Doe rating Pak Supir Satu
INSERT INTO driver_ratings ("bookingId", "driverId", "ratedById", rating, review) VALUES
    (1, 1, 2, 5, 'Driver sangat profesional, tepat waktu dan ramah. Sangat direkomendasikan!');

-- ─── Maintenance Records ──────────────────────────────────────────────────────
INSERT INTO maintenance_records ("resourceId", description, "startDate", "endDate", cost, "createdById") VALUES
    (4,  'Ganti oli mesin, filter oli, dan filter udara — servis berkala 70.000 km',
     NOW() - INTERVAL '2 days', NULL, 850000.00, 1),
    (1,  'Ganti ban depan 2 buah — ban aus',
     NOW() - INTERVAL '20 days', NOW() - INTERVAL '20 days' + INTERVAL '4 hours', 1200000.00, 1),
    (10, 'Perbaikan AC Board Room — kompresor bermasalah',
     NOW() - INTERVAL '7 days', NOW() - INTERVAL '5 days', 2500000.00, 1);

-- ─── Audit Logs ───────────────────────────────────────────────────────────────
INSERT INTO audit_logs ("userId", action, "entityType", "entityId", description) VALUES
    (1, 'CREATE',      'User',              2, 'Admin membuat user John Doe (EMP001)'),
    (1, 'CREATE',      'Vehicle',           1, 'Admin mendaftarkan Toyota Avanza B 1234 XY — kapasitas 7'),
    (1, 'CREATE',      'Vehicle',           7, 'Admin mendaftarkan Hyundai Ioniq 5 B 5555 EV — kapasitas 5'),
    (1, 'CREATE',      'Room',              1, 'Admin mendaftarkan Meeting Room A Lt.2'),
    (1, 'APPROVE',     'Booking',           1, 'Admin menyetujui booking #1 — John Doe (Avanza)'),
    (1, 'ASSIGN',      'Booking',           1, 'Admin assign Pak Supir Satu + Avanza ke booking #1'),
    (1, 'REJECT',      'Booking',           5, 'Admin menolak booking #5 — keperluan pribadi'),
    (2, 'CREATE',      'Booking',           3, 'John Doe membuat booking #3 — Fortuner ke Bandung'),
    (1, 'CREATE',      'MaintenanceRecord', 1, 'Admin mencatat servis L300 (ganti oli)'),
    (1, 'UPDATE',      'MasterSetting',     1, 'Admin set harga BBM default Rp 10.000/liter'),
    (1, 'UPDATE',      'MasterSetting',     2, 'Admin set harga listrik default Rp 2.466/kWh'),
    (1, 'APPROVE',     'Booking',           9, 'Admin menyetujui booking #9 — Ioniq 5 pameran EV'),
    (1, 'ASSIGN',      'Booking',           9, 'Admin assign Pak Supir Satu + Ioniq 5 ke booking #9'),
    (2, 'RATE_DRIVER', 'DriverRating',      1, 'John Doe rating 5/5 untuk Pak Supir Satu (booking #1)');


-- ═══════════════════════════════════════════════════════════════════════════════
-- VIEWS  [REQ 11]
-- ═══════════════════════════════════════════════════════════════════════════════

-- ─── Booking aktif + info assignment ─────────────────────────────────────────
CREATE OR REPLACE VIEW v_active_bookings AS
SELECT
    b.id                AS booking_id,
    b.status,
    b."startDate",
    b."endDate",
    b.purpose,
    u.name              AS user_name,
    u."employeeId",
    dept.name           AS department,
    r.name              AS resource_name,
    r.type              AS resource_type,
    drv.id              AS assigned_driver_id,
    du.name             AS assigned_driver_name,
    drv."phoneNumber"   AS driver_phone,
    v.id                AS assigned_vehicle_id,
    v."plateNumber"     AS vehicle_plate,
    v.capacity          AS vehicle_capacity
FROM bookings b
JOIN users       u    ON u.id    = b."userId"
JOIN departments dept ON dept.id = u."departmentId"
JOIN resources   r    ON r.id    = b."resourceId"
LEFT JOIN drivers  drv ON drv.id = b."assignedDriverId"
LEFT JOIN users    du  ON du.id  = drv."userId"
LEFT JOIN vehicles v   ON v.id   = b."assignedVehicleId"
WHERE b.status IN ('PENDING', 'APPROVED', 'ONGOING');

COMMENT ON VIEW v_active_bookings IS 'Booking aktif dengan info driver dan kendaraan yang diassign';


-- ─── Ringkasan kendaraan — utilitas + biaya BBM & listrik ─────────────────────
CREATE OR REPLACE VIEW v_vehicle_summary AS
SELECT
    v.id,
    r.name                                                                          AS vehicle_name,
    v."plateNumber",
    vc.name                                                                         AS category,
    v.capacity,
    r.status,
    v."currentOdometer",
    COUNT(DISTINCT b.id)                                                            AS total_bookings,
    SUM(CASE WHEN b.status = 'COMPLETED' THEN 1 ELSE 0 END)                        AS completed_bookings,
    COALESCE(SUM(CASE WHEN fe."fuelType" = 'BBM'     THEN fe.liter ELSE 0 END), 0) AS total_liter_bbm,
    COALESCE(SUM(CASE WHEN fe."fuelType" = 'BBM'     THEN fe."totalAmount" ELSE 0 END), 0) AS total_cost_bbm,
    COALESCE(SUM(CASE WHEN fe."fuelType" = 'LISTRIK' THEN fe.kwh   ELSE 0 END), 0) AS total_kwh_listrik,
    COALESCE(SUM(CASE WHEN fe."fuelType" = 'LISTRIK' THEN fe."totalAmount" ELSE 0 END), 0) AS total_cost_listrik,
    COALESCE(SUM(fe."totalAmount"), 0)                                              AS total_fuel_cost
FROM vehicles v
JOIN resources          r  ON r.id  = v."resourceId"
JOIN vehicle_categories vc ON vc.id = v."categoryId"
LEFT JOIN bookings      b  ON b."resourceId" = r.id
LEFT JOIN fuel_expenses fe ON fe."vehicleId" = v.id
GROUP BY v.id, r.name, v."plateNumber", vc.name, v.capacity, r.status, v."currentOdometer";

COMMENT ON VIEW v_vehicle_summary IS '[REQ 11] Ringkasan utilisasi kendaraan — booking, kapasitas, BBM, listrik';


-- ─── Ringkasan rating driver ──────────────────────────────────────────────────
CREATE OR REPLACE VIEW v_driver_ratings_summary AS
SELECT
    d.id                                              AS driver_id,
    u.name                                            AS driver_name,
    u."employeeId",
    d."isActive",
    COUNT(dr.id)                                      AS total_ratings,
    ROUND(AVG(dr.rating)::NUMERIC, 2)                 AS average_rating,
    SUM(CASE WHEN dr.rating = 5 THEN 1 ELSE 0 END)   AS bintang_5,
    SUM(CASE WHEN dr.rating = 4 THEN 1 ELSE 0 END)   AS bintang_4,
    SUM(CASE WHEN dr.rating = 3 THEN 1 ELSE 0 END)   AS bintang_3,
    SUM(CASE WHEN dr.rating = 2 THEN 1 ELSE 0 END)   AS bintang_2,
    SUM(CASE WHEN dr.rating = 1 THEN 1 ELSE 0 END)   AS bintang_1
FROM drivers d
JOIN users u ON u.id = d."userId"
LEFT JOIN driver_ratings dr ON dr."driverId" = d.id
GROUP BY d.id, u.name, u."employeeId", d."isActive";

COMMENT ON VIEW v_driver_ratings_summary IS '[REQ 5 & 11] Dashboard evaluasi driver — rata-rata dan breakdown per bintang';


-- ─── Laporan pengeluaran BBM & listrik per kendaraan ─────────────────────────
CREATE OR REPLACE VIEW v_fuel_expense_summary AS
SELECT
    v.id                                                                             AS vehicle_id,
    v."plateNumber",
    r.name                                                                           AS vehicle_name,
    vc.name                                                                          AS category,
    COUNT(CASE WHEN fe."fuelType" = 'BBM'     THEN 1 END)                           AS bbm_entries,
    COALESCE(SUM(CASE WHEN fe."fuelType" = 'BBM'     THEN fe.liter          END), 0) AS total_liter,
    COALESCE(SUM(CASE WHEN fe."fuelType" = 'BBM'     THEN fe."totalAmount"  END), 0) AS total_cost_bbm,
    COUNT(CASE WHEN fe."fuelType" = 'LISTRIK' THEN 1 END)                           AS listrik_entries,
    COALESCE(SUM(CASE WHEN fe."fuelType" = 'LISTRIK' THEN fe.kwh            END), 0) AS total_kwh,
    COALESCE(SUM(CASE WHEN fe."fuelType" = 'LISTRIK' THEN fe."totalAmount"  END), 0) AS total_cost_listrik,
    COALESCE(SUM(fe."totalAmount"), 0)                                               AS grand_total
FROM vehicles v
JOIN resources          r  ON r.id  = v."resourceId"
JOIN vehicle_categories vc ON vc.id = v."categoryId"
LEFT JOIN fuel_expenses fe ON fe."vehicleId" = v.id
GROUP BY v.id, v."plateNumber", r.name, vc.name;

COMMENT ON VIEW v_fuel_expense_summary IS '[REQ 11] Laporan pengeluaran BBM & listrik SPKLU per kendaraan';