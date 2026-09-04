-- ═══════════════════════════════════════════════════════════════════════════════
-- RESOURCE BOOKING SYSTEM — Schema v3 (Go Fiber + SQLC)
-- Changes from v2:
--   [NEW] vehicles +photoUrl (catalog thumbnail)
--   [NEW] rooms    +photoUrl (catalog thumbnail)
--   [CHG] attachments: filePath (local disk) instead of fileUrl (remote URL)
-- ═══════════════════════════════════════════════════════════════════════════════

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ─── ENUMS ───────────────────────────────────────────────────────────────────
CREATE TYPE role_name AS ENUM ('EMPLOYEE', 'ADMIN', 'DRIVER', 'ROOM_KEEPER');
CREATE TYPE resource_type AS ENUM ('VEHICLE', 'ROOM');
CREATE TYPE resource_status AS ENUM ('AVAILABLE', 'MAINTENANCE', 'INACTIVE');
CREATE TYPE booking_status AS ENUM (
    'PENDING', 'APPROVED', 'REJECTED', 'ONGOING',
    'COMPLETED', 'CANCELLED', 'OVERDUE', 'EXPIRED', 'IGNORED'
);
CREATE TYPE approval_action AS ENUM ('APPROVED', 'REJECTED');
CREATE TYPE fuel_category AS ENUM ('BBM', 'LISTRIK');
CREATE TYPE energy_type AS ENUM ('BBM', 'LISTRIK', 'HYBRID');
CREATE TYPE fuel_unit AS ENUM ('LITER', 'KWH');
CREATE TYPE booking_type AS ENUM ('SPD', 'NON_SPD');

-- ─── FUNCTIONS ────────────────────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION trigger_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW."updatedAt" = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION trigger_set_updated_at_snake_case()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ─── ROLES ───────────────────────────────────────────────────────────────────
CREATE TABLE roles (
    id   SERIAL    PRIMARY KEY,
    name role_name NOT NULL UNIQUE
);

-- ─── DEPARTMENTS ─────────────────────────────────────────────────────────────
CREATE TABLE departments (
    id          SERIAL       PRIMARY KEY,
    name        VARCHAR(100) NOT NULL UNIQUE,
    "createdAt" TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ─── USERS ───────────────────────────────────────────────────────────────────
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

-- ─── AUTH ─────────────────────────────────────────────────────────────────────
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

CREATE TABLE password_reset_otps (
    id          SERIAL      PRIMARY KEY,
    "userId"    INTEGER     NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    "otpCode"   VARCHAR(10) NOT NULL,
    "expiresAt" TIMESTAMPTZ NOT NULL,
    "isUsed"    BOOLEAN     NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_otp_user_id ON password_reset_otps("userId");
CREATE INDEX idx_otp_is_used ON password_reset_otps("isUsed");

-- ─── RESOURCES ────────────────────────────────────────────────────────────────
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


-- ─── FUEL TYPES ───────────────────────────────────────────────────────────────
CREATE TABLE fuel_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type fuel_category NOT NULL,
    unit fuel_unit NOT NULL,
    default_price NUMERIC(12,2) NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_updated_at_fuel_types
    BEFORE UPDATE ON fuel_types FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at_snake_case();

-- ─── VEHICLE CATEGORIES ───────────────────────────────────────────────────────
CREATE TABLE vehicle_categories (
    id   SERIAL       PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE
);

-- ─── VEHICLES ─────────────────────────────────────────────────────────────────
CREATE TABLE vehicles (
    id                SERIAL        PRIMARY KEY,
    "resourceId"      INTEGER       NOT NULL UNIQUE REFERENCES resources(id) ON DELETE CASCADE,
    "plateNumber"     VARCHAR(20)   NOT NULL UNIQUE,
    brand             VARCHAR(100)  NOT NULL,
    model             VARCHAR(100)  NOT NULL,
    year              SMALLINT      NOT NULL CHECK (year >= 1900 AND year <= 2100),
    "currentOdometer" INTEGER       NOT NULL DEFAULT 0 CHECK ("currentOdometer" >= 0),
    "categoryId"      INTEGER       NOT NULL REFERENCES vehicle_categories(id),
    capacity          SMALLINT      NOT NULL DEFAULT 4 CHECK (capacity > 0),
    "photoUrl"        VARCHAR(500)  NULL,
    "energy_type"     energy_type   NOT NULL DEFAULT 'BBM',
    "maintenanceIntervalKm"   INTEGER NOT NULL DEFAULT 10000 CHECK ("maintenanceIntervalKm" > 0),
    "lastMaintenanceOdometer" INTEGER NOT NULL DEFAULT 0 CHECK ("lastMaintenanceOdometer" >= 0)
);

CREATE INDEX idx_vehicles_plate_number ON vehicles("plateNumber");
CREATE INDEX idx_vehicles_category_id  ON vehicles("categoryId");

-- ─── ROOMS ────────────────────────────────────────────────────────────────────
CREATE TABLE rooms (
    id           SERIAL       PRIMARY KEY,
    "resourceId" INTEGER      NOT NULL UNIQUE REFERENCES resources(id) ON DELETE CASCADE,
    location     VARCHAR(255) NOT NULL,
    capacity     SMALLINT     NOT NULL CHECK (capacity > 0),
    "photoUrl"   VARCHAR(500) NULL
);

-- ─── DRIVERS ──────────────────────────────────────────────────────────────────
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

-- ─── ROOM KEEPERS ─────────────────────────────────────────────────────────────
CREATE TABLE room_keepers (
    id            SERIAL      PRIMARY KEY,
    "userId"      INTEGER     NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    "phoneNumber" VARCHAR(20) NOT NULL,
    "isActive"    BOOLEAN     NOT NULL DEFAULT TRUE,
    "createdAt"   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_room_keepers_user_id   ON room_keepers("userId");
CREATE INDEX idx_room_keepers_is_active ON room_keepers("isActive");

-- ─── BOOKINGS ─────────────────────────────────────────────────────────────────
CREATE TABLE bookings (
    id                   SERIAL         PRIMARY KEY,
    "userId"             INTEGER        NOT NULL REFERENCES users(id),
    "resourceId"         INTEGER        NOT NULL REFERENCES resources(id),
    "startDate"          TIMESTAMPTZ    NOT NULL,
    "endDate"            TIMESTAMPTZ    NOT NULL,
    purpose              TEXT           NOT NULL,
    "passengerCount"     INTEGER        NOT NULL DEFAULT 1,
    status               booking_status NOT NULL DEFAULT 'PENDING',
    "approvedById"       INTEGER        REFERENCES users(id),
    "approvedAt"         TIMESTAMPTZ,
    "assignedDriverId"   INTEGER        REFERENCES drivers(id),
    "assignedVehicleId"  INTEGER        REFERENCES vehicles(id),
    "assignedAt"           TIMESTAMPTZ,
    "returnedAt"           TIMESTAMPTZ,
    "createdAt"            TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    "updatedAt"            TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    "originalResourceId"   INTEGER        REFERENCES resources(id),
    "bookingType"          booking_type   NOT NULL DEFAULT 'NON_SPD',
    CONSTRAINT chk_booking_dates CHECK ("endDate" > "startDate")
);
-- Kolom odometer perjalanan ditambahkan di migration 000002 (additive).

CREATE INDEX idx_bookings_user_id          ON bookings("userId");
CREATE INDEX idx_bookings_resource_id      ON bookings("resourceId");
CREATE INDEX idx_bookings_status           ON bookings(status);
CREATE INDEX idx_bookings_start_date       ON bookings("startDate");
CREATE INDEX idx_bookings_end_date         ON bookings("endDate");
CREATE INDEX idx_bookings_assigned_driver  ON bookings("assignedDriverId");
CREATE INDEX idx_bookings_assigned_vehicle ON bookings("assignedVehicleId");
CREATE INDEX idx_bookings_active ON bookings("resourceId", "startDate", "endDate")
    WHERE status IN ('PENDING', 'APPROVED', 'ONGOING');

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

-- ─── BOOKING MERGES ───────────────────────────────────────────────────────────
-- Links two APPROVED vehicle bookings into a shared trip without changing their
-- individual status flow. The primary booking is the one whose vehicle/driver is used.
-- For reporting, only the primary booking counts as a distinct trip.
CREATE TABLE booking_merges (
    id                  SERIAL       PRIMARY KEY,
    "primaryBookingId"  INTEGER      NOT NULL REFERENCES bookings(id),
    "mergedBookingId"   INTEGER      NOT NULL REFERENCES bookings(id),
    "mergedById"        INTEGER      NOT NULL REFERENCES users(id),
    reason              TEXT,
    "createdAt"         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_different_bookings CHECK ("primaryBookingId" != "mergedBookingId"),
    UNIQUE("primaryBookingId", "mergedBookingId")
);

CREATE INDEX idx_booking_merges_primary ON booking_merges("primaryBookingId");
CREATE INDEX idx_booking_merges_merged  ON booking_merges("mergedBookingId");

-- ─── DRIVER ASSIGNMENTS ───────────────────────────────────────────────────────
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

-- ─── DRIVER RATINGS ───────────────────────────────────────────────────────────
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

-- ─── FUEL EXPENSES ────────────────────────────────────────────────────────────
CREATE TABLE fuel_expenses (
    id               SERIAL        PRIMARY KEY,
    "vehicleId"      INTEGER       NOT NULL REFERENCES vehicles(id),
    "fuelTypeId"     INTEGER       NOT NULL REFERENCES fuel_types(id),
    "bookingId"      INTEGER       REFERENCES bookings(id),
    "driverId"       INTEGER       REFERENCES drivers(id),
    "recordedById"   INTEGER       NOT NULL REFERENCES users(id),
    "fuelGrade"      VARCHAR(50)   NULL,
    "proofPhotoUrl"  VARCHAR(255)  NULL,
    "odometerBefore" INTEGER       NULL CHECK ("odometerBefore" IS NULL OR "odometerBefore" >= 0),
    "odometerAfter"  INTEGER       NULL CHECK ("odometerAfter" IS NULL OR "odometerAfter" >= 0),
    "distanceKm"     INTEGER       NULL CHECK ("distanceKm" IS NULL OR "distanceKm" >= 0),
    quantity         NUMERIC(10,2) NULL CHECK (quantity IS NULL OR quantity > 0),
    "pricePerUnit"   NUMERIC(12,2) NULL CHECK ("pricePerUnit" IS NULL OR "pricePerUnit" > 0),
    "totalCost"      NUMERIC(14,2) NULL CHECK ("totalCost" IS NULL OR "totalCost" > 0),
    "batteryBefore"  NUMERIC(5,2)  NULL CHECK ("batteryBefore" IS NULL OR ("batteryBefore" >= 0 AND "batteryBefore" <= 100)),
    "batteryAfter"   NUMERIC(5,2)  NULL CHECK ("batteryAfter"  IS NULL OR ("batteryAfter"  >= 0 AND "batteryAfter"  <= 100)),
    location         VARCHAR(255)  NOT NULL,
    "stationName"    VARCHAR(255)  NULL,
    note             TEXT          NULL,
    "createdAt"      TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_fuel_expenses_vehicle_id ON fuel_expenses("vehicleId");
CREATE INDEX idx_fuel_expenses_fuel_type_id ON fuel_expenses("fuelTypeId");
CREATE INDEX idx_fuel_expenses_booking_id ON fuel_expenses("bookingId");
CREATE INDEX idx_fuel_expenses_driver_id  ON fuel_expenses("driverId");
CREATE INDEX idx_fuel_expenses_created_at ON fuel_expenses("createdAt");

-- ─── MASTER SETTINGS ──────────────────────────────────────────────────────────
CREATE TABLE master_settings (
    id          SERIAL        PRIMARY KEY,
    key         VARCHAR(100)  NOT NULL UNIQUE,
    value       NUMERIC(14,4) NOT NULL, 
    unit        VARCHAR(50)   NULL,
    description TEXT          NULL,
    "updatedAt" TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_master_settings_key ON master_settings(key);

-- ─── MAINTENANCE RECORDS ──────────────────────────────────────────────────────
CREATE TABLE maintenance_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE maintenance_records (
    id                  SERIAL        PRIMARY KEY,
    "vehicleId"         INTEGER       NOT NULL REFERENCES vehicles(id),
    "maintenanceTypeId" INTEGER       NULL REFERENCES maintenance_types(id),
    description         TEXT          NOT NULL,
    type                VARCHAR(50)   NOT NULL,
    status              VARCHAR(50)   NOT NULL DEFAULT 'ONGOING',
    "vendorName"        VARCHAR(255)  NULL,
    "proofPhotos"       JSONB         NULL,
    odometer            INTEGER       NULL CHECK (odometer >= 0),
    "totalCost"         NUMERIC(12,2) CHECK ("totalCost" IS NULL OR "totalCost" >= 0),
    location            VARCHAR(255)  NOT NULL,
    "startDate"         TIMESTAMPTZ   NOT NULL,
    "endDate"           TIMESTAMPTZ   NULL,
    "completedAt"       TIMESTAMPTZ   NULL,
    "recordedById"      INTEGER       NOT NULL REFERENCES users(id),
    "createdAt"         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    "isAutoGenerated"   BOOLEAN       NOT NULL DEFAULT FALSE,
    CONSTRAINT chk_maintenance_dates CHECK ("endDate" IS NULL OR "endDate" > "startDate")
);

CREATE INDEX idx_maintenance_vehicle_id    ON maintenance_records("vehicleId");
CREATE INDEX idx_maintenance_recorded_by_id ON maintenance_records("recordedById");
CREATE INDEX idx_maintenance_start_date    ON maintenance_records("startDate");

-- ─── DRIVER OVERTIMES ─────────────────────────────────────────────────────────
-- Recorded when a NON_SPD vehicle booking is completed after its scheduled
-- endDate (normal working hours = 8h, i.e. up to the booking's own endDate).
CREATE TABLE driver_overtimes (
    id                SERIAL       PRIMARY KEY,
    "bookingId"       INTEGER      NOT NULL UNIQUE REFERENCES bookings(id) ON DELETE CASCADE,
    "driverId"        INTEGER      NOT NULL REFERENCES drivers(id),
    "scheduledEndAt"  TIMESTAMPTZ  NOT NULL,
    "actualEndAt"     TIMESTAMPTZ  NOT NULL,
    "overtimeMinutes" INTEGER      NOT NULL CHECK ("overtimeMinutes" > 0),
    "createdAt"        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_overtime_dates CHECK ("actualEndAt" > "scheduledEndAt")
);

CREATE INDEX idx_driver_overtimes_driver_id  ON driver_overtimes("driverId");
CREATE INDEX idx_driver_overtimes_booking_id ON driver_overtimes("bookingId");

-- ─── AUDIT LOGS ───────────────────────────────────────────────────────────────
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

-- ─── GUEST BOOKINGS ───────────────────────────────────────────────────────────
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

-- ─── BOOKING RETURN REPORTS ───────────────────────────────────────────────────
-- Driver submits end-of-trip note, location, and photos (stored as booking
-- attachments with description='return_photo'). Admin reviews before completing.
CREATE TABLE booking_return_reports (
    id               SERIAL       PRIMARY KEY,
    "bookingId"      INTEGER      NOT NULL UNIQUE REFERENCES bookings(id) ON DELETE CASCADE,
    "submittedById"  INTEGER      NOT NULL REFERENCES users(id),
    note             TEXT         NOT NULL,
    location         VARCHAR(500) NOT NULL,
    "submittedAt"    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
    -- kolom odometer ditambahkan di migration 000002 (additive)
);

CREATE INDEX idx_return_reports_booking ON booking_return_reports("bookingId");

COMMENT ON TABLE booking_return_reports IS 'Laporan akhir perjalanan dari driver — note, lokasi, foto dikirim sebelum admin complete booking';


-- ─── ATTACHMENTS ──────────────────────────────────────────────────────────────
-- filePath: relative path in uploads/ directory (e.g. "vehicle/2024/01/uuid.jpg")
CREATE TABLE attachments (
    id             SERIAL       PRIMARY KEY,
    "uploadedById" INTEGER      NOT NULL REFERENCES users(id),
    "vehicleId"    INTEGER      REFERENCES vehicles(id)  ON DELETE CASCADE,
    "roomId"       INTEGER      REFERENCES rooms(id)     ON DELETE CASCADE,
    "bookingId"    INTEGER      REFERENCES bookings(id)  ON DELETE CASCADE,
    "fuelExpenseId" INTEGER     REFERENCES fuel_expenses(id) ON DELETE CASCADE,
    "maintenanceId" INTEGER     REFERENCES maintenance_records(id) ON DELETE CASCADE,
    "filePath"     VARCHAR(500) NOT NULL,
    "fileName"     VARCHAR(255) NOT NULL,
    "fileType"     VARCHAR(100) NOT NULL,
    "fileSize"     INTEGER      NULL,
    description    TEXT         NULL,
    "createdAt"    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_one_target CHECK (
        (CASE WHEN "vehicleId" IS NOT NULL THEN 1 ELSE 0 END +
         CASE WHEN "roomId"    IS NOT NULL THEN 1 ELSE 0 END +
         CASE WHEN "bookingId" IS NOT NULL THEN 1 ELSE 0 END +
         CASE WHEN "fuelExpenseId" IS NOT NULL THEN 1 ELSE 0 END +
         CASE WHEN "maintenanceId" IS NOT NULL THEN 1 ELSE 0 END) = 1
    )
);

CREATE INDEX idx_att_vehicle  ON attachments("vehicleId");
CREATE INDEX idx_att_room     ON attachments("roomId");
CREATE INDEX idx_att_booking  ON attachments("bookingId");
CREATE INDEX idx_att_uploader ON attachments("uploadedById");

-- ─── TRIGGERS ─────────────────────────────────────────────────────────────────

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

-- ─── VIEWS ────────────────────────────────────────────────────────────────────
CREATE OR REPLACE VIEW v_active_bookings AS
SELECT
    b.id AS booking_id, b.status, b."startDate", b."endDate", b.purpose,
    u.name AS user_name, u."employeeId",
    dept.name AS department,
    r.name AS resource_name, r.type AS resource_type,
    drv.id AS assigned_driver_id,
    du.name AS assigned_driver_name,
    drv."phoneNumber" AS driver_phone,
    v.id AS assigned_vehicle_id,
    v."plateNumber" AS vehicle_plate,
    v.capacity AS vehicle_capacity
FROM bookings b
JOIN users u ON u.id = b."userId"
JOIN departments dept ON dept.id = u."departmentId"
JOIN resources r ON r.id = b."resourceId"
LEFT JOIN drivers drv ON drv.id = b."assignedDriverId"
LEFT JOIN users du ON du.id = drv."userId"
LEFT JOIN vehicles v ON v.id = b."assignedVehicleId"
WHERE b.status IN ('PENDING', 'APPROVED', 'ONGOING');

CREATE OR REPLACE VIEW v_vehicle_summary AS
SELECT
    v.id, r.name AS vehicle_name, v."plateNumber",
    vc.name AS category, v.capacity, r.status, v."currentOdometer",
    COUNT(DISTINCT b.id) AS total_bookings,
    SUM(CASE WHEN b.status = 'COMPLETED' THEN 1 ELSE 0 END) AS completed_bookings,
    COALESCE(SUM(CASE WHEN ft.type = 'BBM' THEN fe.quantity ELSE 0 END), 0) AS total_liter_bbm,
    COALESCE(SUM(CASE WHEN ft.type = 'BBM' THEN fe."totalCost" ELSE 0 END), 0) AS total_cost_bbm,
    COALESCE(SUM(CASE WHEN ft.type = 'LISTRIK' THEN fe.quantity ELSE 0 END), 0) AS total_kwh_listrik,
    COALESCE(SUM(CASE WHEN ft.type = 'LISTRIK' THEN fe."totalCost" ELSE 0 END), 0) AS total_cost_listrik,
    COALESCE(SUM(fe."totalCost"), 0) AS total_fuel_cost
FROM vehicles v
JOIN resources r ON r.id = v."resourceId"
JOIN vehicle_categories vc ON vc.id = v."categoryId"
LEFT JOIN bookings b ON b."resourceId" = r.id
LEFT JOIN fuel_expenses fe ON fe."vehicleId" = v.id
LEFT JOIN fuel_types ft ON ft.id = fe."fuelTypeId"
GROUP BY v.id, r.name, v."plateNumber", vc.name, v.capacity, r.status, v."currentOdometer";

CREATE OR REPLACE VIEW v_driver_ratings_summary AS
SELECT 
    d.id AS driver_id, u.name AS driver_name, u."employeeId", d."isActive",
    COUNT(dr.id) AS total_ratings,
    ROUND(AVG(dr.rating)::NUMERIC, 2) AS average_rating,
    SUM(CASE WHEN dr.rating = 5 THEN 1 ELSE 0 END) AS bintang_5,
    SUM(CASE WHEN dr.rating = 4 THEN 1 ELSE 0 END) AS bintang_4,
    SUM(CASE WHEN dr.rating = 3 THEN 1 ELSE 0 END) AS bintang_3,
    SUM(CASE WHEN dr.rating = 2 THEN 1 ELSE 0 END) AS bintang_2,
    SUM(CASE WHEN dr.rating = 1 THEN 1 ELSE 0 END) AS bintang_1
FROM drivers d
JOIN users u ON u.id = d."userId"
LEFT JOIN driver_ratings dr ON dr."driverId" = d.id
GROUP BY d.id, u.name, u."employeeId", d."isActive";

CREATE OR REPLACE VIEW v_fuel_expense_summary AS
SELECT
    v.id AS vehicle_id, v."plateNumber", r.name AS vehicle_name, vc.name AS category,
    COUNT(CASE WHEN ft.type = 'BBM' THEN 1 END) AS bbm_entries,
    COALESCE(SUM(CASE WHEN ft.type = 'BBM' THEN fe.quantity END), 0) AS total_liter,
    COALESCE(SUM(CASE WHEN ft.type = 'BBM' THEN fe."totalCost" END), 0) AS total_cost_bbm,
    COUNT(CASE WHEN ft.type = 'LISTRIK' THEN 1 END) AS listrik_entries,
    COALESCE(SUM(CASE WHEN ft.type = 'LISTRIK' THEN fe.quantity END), 0) AS total_kwh,
    COALESCE(SUM(CASE WHEN ft.type = 'LISTRIK' THEN fe."totalCost" END), 0) AS total_cost_listrik,
    COALESCE(SUM(fe."totalCost"), 0) AS grand_total
FROM vehicles v
JOIN resources r ON r.id = v."resourceId"
JOIN vehicle_categories vc ON vc.id = v."categoryId"
LEFT JOIN fuel_expenses fe ON fe."vehicleId" = v.id
LEFT JOIN fuel_types ft ON ft.id = fe."fuelTypeId"
GROUP BY v.id, v."plateNumber", r.name, vc.name;

-- ─── SEED DATA ────────────────────────────────────────────────────────────────
INSERT INTO roles (name) VALUES
    ('EMPLOYEE'),     -- id 1
    ('ADMIN'),        -- id 2
    ('DRIVER'),       -- id 3
    ('ROOM_KEEPER');  -- id 4

INSERT INTO departments (name) VALUES
    ('Information Technology'), ('Human Resources'),
    ('Finance & Accounting'), ('Operations'), ('Marketing');

-- Passwords: bcrypt of "Password123!" (except admin = "admin")
INSERT INTO users ("employeeId", name, email, password, "isActive", "roleId", "departmentId") VALUES
    ('ADM001', 'Admin Utama',     'admin@company.com',           '$2a$10$v8DgunH6sxHTB4GkapmmIOz97bOUKt1u/mfdrI55s6MiLrLeUjlNG',                                                         TRUE, 2, 1),
    ('EMP001', 'John Doe',        'john.doe@company.com',        '$2a$10$v8DgunH6sxHTB4GkapmmIOz97bOUKt1u/mfdrI55s6MiLrLeUjlNG', TRUE, 1, 1),
    ('EMP002', 'Jane Smith',      'jane.smith@company.com',      '$2a$10$v8DgunH6sxHTB4GkapmmIOz97bOUKt1u/mfdrI55s6MiLrLeUjlNG', TRUE, 1, 3),
    ('EMP003', 'Dewi Lestari',    'dewi.lestari@company.com',    '$2a$10$v8DgunH6sxHTB4GkapmmIOz97bOUKt1u/mfdrI55s6MiLrLeUjlNG', TRUE, 1, 5),
    ('EMP004', 'Andi Supriadi',   'andi.supriadi@company.com',   '$2a$10$v8DgunH6sxHTB4GkapmmIOz97bOUKt1u/mfdrI55s6MiLrLeUjlNG', TRUE, 1, 4),
    ('DRV001', 'Pak Supir Satu',  'supir1@company.com',          '$2a$10$v8DgunH6sxHTB4GkapmmIOz97bOUKt1u/mfdrI55s6MiLrLeUjlNG', TRUE, 3, 4),
    ('DRV002', 'Pak Supir Dua',   'supir2@company.com',          '$2a$10$v8DgunH6sxHTB4GkapmmIOz97bOUKt1u/mfdrI55s6MiLrLeUjlNG', TRUE, 3, 4);

INSERT INTO vehicle_categories (name) VALUES
    ('MPV'), ('SUV'), ('Sedan'), ('Pickup'), ('Bus / Minibus'), ('Listrik / EV');

-- Plain INSERT (no ON CONFLICT) on purpose - 000008_dedupe_fuel_types.up.sql
-- runs after this file and adds UNIQUE(name), which doesn't exist yet the
-- first time this statement executes, so ON CONFLICT (name) would error on
-- every fresh deploy including the very first one. Once that constraint
-- exists, reruns of this INSERT just fail-and-skip on the duplicate key
-- violation (same harmless "error and move on" pattern every other
-- re-applied CREATE TABLE/etc. in this migrate step already relies on).
INSERT INTO fuel_types (name, type, unit, default_price, is_active) VALUES
    ('Pertalite', 'BBM', 'LITER', 10000.00, TRUE),
    ('Pertamax', 'BBM', 'LITER', 13500.00, TRUE),
    ('Dexlite', 'BBM', 'LITER', 14500.00, TRUE),
    ('Listrik PLN', 'LISTRIK', 'KWH', 2466.00, TRUE);

INSERT INTO resources (name, type, status) VALUES
    ('Toyota Avanza - B 1234 XY',   'VEHICLE', 'AVAILABLE'),
    ('Honda CR-V - B 5678 AB',       'VEHICLE', 'AVAILABLE'),
    ('Toyota Fortuner - B 9999 CD',  'VEHICLE', 'AVAILABLE'),
    ('Mitsubishi L300 - B 2222 EF',  'VEHICLE', 'MAINTENANCE'),
    ('Daihatsu Xenia - B 3333 GH',   'VEHICLE', 'AVAILABLE'),
    ('Toyota HiAce - B 4444 IJ',     'VEHICLE', 'INACTIVE'),
    ('Hyundai Ioniq 5 - B 5555 EV',  'VEHICLE', 'AVAILABLE'),
    ('Meeting Room A - Lt.2',        'ROOM',    'AVAILABLE'),
    ('Meeting Room B - Lt.3',        'ROOM',    'AVAILABLE'),
    ('Board Room - Lt.5',             'ROOM',    'AVAILABLE'),
    ('Training Room - Annex Lt.1',   'ROOM',    'INACTIVE');

INSERT INTO vehicles ("resourceId", "plateNumber", brand, model, year, "currentOdometer", "categoryId", capacity, energy_type) VALUES
    (1, 'B 1234 XY', 'Toyota',     'Avanza',   2022, 15000, 1, 7, 'BBM'),
    (2, 'B 5678 AB', 'Honda',      'CR-V',     2021, 28500, 2, 5, 'BBM'),
    (3, 'B 9999 CD', 'Toyota',     'Fortuner', 2023,  5200, 2, 7, 'BBM'),
    (4, 'B 2222 EF', 'Mitsubishi', 'L300',     2020, 72000, 4, 8, 'BBM'),
    (5, 'B 3333 GH', 'Daihatsu',   'Xenia',    2022, 18300, 1, 7, 'BBM'),
    (6, 'B 4444 IJ', 'Toyota',     'HiAce',    2019, 95000, 5, 15, 'BBM'),
    (7, 'B 5555 EV', 'Hyundai',    'Ioniq 5',  2024,  3200, 6, 5, 'LISTRIK');

INSERT INTO rooms ("resourceId", location, capacity) VALUES
    ( 8, 'Gedung Utama Lt. 2', 10),
    ( 9, 'Gedung Utama Lt. 3', 20),
    (10, 'Gedung Utama Lt. 5', 50),
    (11, 'Gedung Annex Lt. 1', 30);

INSERT INTO drivers ("userId", "licenseNumber", "phoneNumber", "isActive") VALUES
    (6, 'SIM-B1-2024-001', '+6281234567890', TRUE),
    (7, 'SIM-B1-2024-002', '+6287654321098', TRUE);

INSERT INTO driver_assignments ("driverId", "vehicleId", "assignedAt") VALUES
    (1, 1, NOW() - INTERVAL '30 days'),
    (2, 2, NOW() - INTERVAL '15 days');

INSERT INTO master_settings (key, value, unit, description) VALUES
    ('fuel_price_pertalite',   10000.0000, 'IDR/liter', 'Harga BBM Pertalite'),
    ('fuel_price_pertamax',    12950.0000, 'IDR/liter', 'Harga BBM Pertamax'),
    ('fuel_price_pertamax_turbo', 14400.0000, 'IDR/liter', 'Harga BBM Pertamax Turbo'),
    ('fuel_price_solar',       6800.0000,  'IDR/liter', 'Harga BBM Solar Subsidi'),
    ('fuel_price_dexlite',     14550.0000, 'IDR/liter', 'Harga BBM Dexlite'),
    ('fuel_price_pertamina_dex', 15100.0000, 'IDR/liter', 'Harga BBM Pertamina Dex'),
    ('fuel_price_listrik',     2466.0000,  'IDR/kWh',   'Tarif listrik PLN per kWh');

-- ─── BOOKINGS ─────────────────────────────────────────────────────────────────
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

    -- [2] APPROVED — Jane Smith, Meeting Room B (ruangan)
    (3, 9,
     NOW() + INTERVAL '2 days', NOW() + INTERVAL '2 days' + INTERVAL '3 hours',
     'Rapat koordinasi tim Finance Q1', 'APPROVED',
     1, NOW() - INTERVAL '1 day',
     NULL, NULL, NULL, NULL),

    -- [3] PENDING — John Doe, Fortuner
    (2, 3,
     NOW() + INTERVAL '5 days', NOW() + INTERVAL '6 days',
     'Perjalanan dinas ke Bandung', 'PENDING',
     NULL, NULL, NULL, NULL, NULL, NULL),

    -- [4] PENDING — Dewi, Board Room
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

-- ─── APPROVAL LOGS ────────────────────────────────────────────────────────────
INSERT INTO approval_logs ("bookingId", "approverId", action, note) VALUES
    (1, 1, 'APPROVED', 'Disetujui — keperluan klien prioritas'),
    (2, 1, 'APPROVED', 'OK, silakan'),
    (5, 1, 'REJECTED', 'Booking untuk keperluan pribadi tidak diizinkan'),
    (7, 1, 'APPROVED', 'Disetujui untuk survey lokasi proyek'),
    (9, 1, 'APPROVED', 'Disetujui — kendaraan listrik tersedia');

-- ─── FUEL EXPENSES ────────────────────────────────────────────────────────────
INSERT INTO fuel_expenses (
    "driverId", "vehicleId", "bookingId", "fuelTypeId", "recordedById",
    "odometerBefore", "odometerAfter", "distanceKm", quantity, "pricePerUnit", "totalCost",
    location, "stationName", note
) VALUES
    (1, 1, 1, 1, 1, 14900, 15000, 100, 40.50, 10000.00, 405000.00, 'Jakarta', 'SPBU Pertamina Jl. Sudirman', 'Isi BBM full tank'),
    (1, 1, 6, 1, 1, 15200, 15320, 120, 35.00, 10000.00, 350000.00, 'Jakarta', 'SPBU Shell Jl. Gatot Subroto', 'Isi BBM perjalanan'),
    (2, 2, 7, 1, 1, 28300, 28500, 200, 50.00, 10200.00, 510000.00, 'Bekasi', 'SPBU Pertamina Bekasi', 'Isi BBM luar kota');

INSERT INTO fuel_expenses (
    "driverId", "vehicleId", "bookingId", "fuelTypeId", "recordedById",
    quantity, "pricePerUnit", "totalCost",
    "batteryBefore", "batteryAfter",
    location, "stationName", note
) VALUES
    (1, 7, 9, 4, 1, 45.00, 2466.00, 110970.00, 20.00, 95.00, 'Jakarta', 'SPKLU PLN Kemayoran', 'Charge 75%');

-- ─── DRIVER RATINGS ───────────────────────────────────────────────────────────
INSERT INTO driver_ratings ("bookingId", "driverId", "ratedById", rating, review) VALUES
    (1, 1, 2, 5, 'Driver sangat profesional, tepat waktu dan ramah. Sangat direkomendasikan!');

-- ─── MAINTENANCE RECORDS ──────────────────────────────────────────────────────
INSERT INTO maintenance_types (name) VALUES ('Servis Berkala'), ('Ganti Ban'), ('Perbaikan AC');

INSERT INTO maintenance_records ("vehicleId", "maintenanceTypeId", description, type, status, "startDate", "endDate", "completedAt", "totalCost", "recordedById", location, odometer, "vendorName") VALUES
    (4, 1, 'Ganti oli mesin, filter oli, dan filter udara — servis berkala 70.000 km', 'RUTIN', 'ONGOING',
     NOW() - INTERVAL '2 days', NULL, NULL, 850000.00, 1, 'Bengkel Resmi Mitsubishi', 72000, 'Bengkel A'),
    (1, 2, 'Ganti ban depan 2 buah — ban aus', 'PENGGANTIAN', 'COMPLETED',
     NOW() - INTERVAL '20 days', NOW() - INTERVAL '20 days' + INTERVAL '4 hours', NOW() - INTERVAL '20 days' + INTERVAL '4 hours', 1200000.00, 1, 'Toko Ban Jakarta', 14500, 'Toko Ban B');
    -- Room 10 AC repair is removed because maintenance is now vehicle-only

-- ─── AUDIT LOGS ───────────────────────────────────────────────────────────────
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

-- Notifications
CREATE TABLE notifications (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    type VARCHAR(50) NOT NULL,
    related_entity_id INT,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
