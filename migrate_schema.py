import re

with open('sql/schema/000001_init.up.sql', 'r') as f:
    sql = f.read()

# 1. Enums
sql = sql.replace(
    "CREATE TYPE fuel_type AS ENUM ('BBM', 'LISTRIK');",
    "CREATE TYPE fuel_type AS ENUM ('BBM', 'LISTRIK');\nCREATE TYPE energy_type AS ENUM ('BBM', 'LISTRIK', 'HYBRID');\nCREATE TYPE fuel_unit AS ENUM ('LITER', 'KWH');"
)

# 2. Master Data: fuel_types table
fuel_types_sql = """
-- ─── FUEL TYPES ───────────────────────────────────────────────────────────────
CREATE TABLE fuel_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type fuel_type NOT NULL,
    unit fuel_unit NOT NULL,
    default_price NUMERIC(12,2) NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER set_updated_at_fuel_types
    BEFORE UPDATE ON fuel_types FOR EACH ROW EXECUTE FUNCTION trigger_set_updated_at();
"""
sql = sql.replace("-- ─── VEHICLE CATEGORIES ───────────────────────────────────────────────────────", fuel_types_sql + "\n-- ─── VEHICLE CATEGORIES ───────────────────────────────────────────────────────")

# 3. vehicles table
sql = sql.replace(
    """"categoryId"      INTEGER       NOT NULL REFERENCES vehicle_categories(id),
    capacity          SMALLINT      NOT NULL DEFAULT 4 CHECK (capacity > 0),
    "photoUrl"        VARCHAR(500)  NULL
);""",
    """"categoryId"      INTEGER       NOT NULL REFERENCES vehicle_categories(id),
    capacity          SMALLINT      NOT NULL DEFAULT 4 CHECK (capacity > 0),
    "photoUrl"        VARCHAR(500)  NULL,
    "energy_type"     energy_type   NOT NULL DEFAULT 'BBM'
);"""
)

# 4. fuel_expenses table
old_fuel_expenses = """CREATE TABLE fuel_expenses (
    id               SERIAL        PRIMARY KEY,
    "driverId"       INTEGER       NOT NULL REFERENCES drivers(id),
    "vehicleId"      INTEGER       NOT NULL REFERENCES vehicles(id),
    "bookingId"      INTEGER       REFERENCES bookings(id),
    "fuelType"       fuel_type     NOT NULL DEFAULT 'BBM',
    liter            NUMERIC(10,2) NULL CHECK (liter IS NULL OR liter > 0),
    "pricePerLiter"  NUMERIC(12,2) NULL CHECK ("pricePerLiter" IS NULL OR "pricePerLiter" > 0),
    "odometerBefore" INTEGER       NULL CHECK ("odometerBefore" IS NULL OR "odometerBefore" >= 0),
    "odometerAfter"  INTEGER       NULL,
    kwh              NUMERIC(10,2) NULL CHECK (kwh IS NULL OR kwh > 0),
    "pricePerKwh"    NUMERIC(12,2) NULL CHECK ("pricePerKwh" IS NULL OR "pricePerKwh" > 0),
    "batteryBefore"  NUMERIC(5,2)  NULL CHECK ("batteryBefore" IS NULL OR ("batteryBefore" >= 0 AND "batteryBefore" <= 100)),
    "batteryAfter"   NUMERIC(5,2)  NULL CHECK ("batteryAfter"  IS NULL OR ("batteryAfter"  >= 0 AND "batteryAfter"  <= 100)),
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
        "fuelType" <> 'LISTRIK' OR (kwh IS NOT NULL AND "pricePerKwh" IS NOT NULL)
    )
);

CREATE INDEX idx_fuel_expenses_driver_id  ON fuel_expenses("driverId");
CREATE INDEX idx_fuel_expenses_vehicle_id ON fuel_expenses("vehicleId");
CREATE INDEX idx_fuel_expenses_booking_id ON fuel_expenses("bookingId");
CREATE INDEX idx_fuel_expenses_fuel_type  ON fuel_expenses("fuelType");
CREATE INDEX idx_fuel_expenses_created_at ON fuel_expenses("createdAt");"""

new_fuel_expenses = """CREATE TABLE fuel_expenses (
    id               SERIAL        PRIMARY KEY,
    "vehicleId"      INTEGER       NOT NULL REFERENCES vehicles(id),
    "fuelTypeId"     INTEGER       NOT NULL REFERENCES fuel_types(id),
    "bookingId"      INTEGER       REFERENCES bookings(id),
    "driverId"       INTEGER       REFERENCES drivers(id),
    "recordedById"   INTEGER       NOT NULL REFERENCES users(id),
    odometer         INTEGER       NULL CHECK (odometer IS NULL OR odometer >= 0),
    quantity         NUMERIC(10,2) NOT NULL CHECK (quantity > 0),
    "pricePerUnit"   NUMERIC(12,2) NOT NULL CHECK ("pricePerUnit" > 0),
    "totalCost"      NUMERIC(14,2) NOT NULL CHECK ("totalCost" > 0),
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
CREATE INDEX idx_fuel_expenses_created_at ON fuel_expenses("createdAt");"""

sql = sql.replace(old_fuel_expenses, new_fuel_expenses)

# 5. maintenance_records table
old_maintenance = """CREATE TABLE maintenance_records (
    id            SERIAL        PRIMARY KEY,
    "resourceId"  INTEGER       NOT NULL REFERENCES resources(id),
    description   TEXT          NOT NULL,
    "startDate"   TIMESTAMPTZ   NOT NULL,
    "endDate"     TIMESTAMPTZ,
    cost          NUMERIC(12,2) CHECK (cost >= 0),
    "createdById" INTEGER       NOT NULL REFERENCES users(id),
    "createdAt"   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_maintenance_dates CHECK ("endDate" IS NULL OR "endDate" > "startDate")
);

CREATE INDEX idx_maintenance_resource_id   ON maintenance_records("resourceId");
CREATE INDEX idx_maintenance_created_by_id ON maintenance_records("createdById");
CREATE INDEX idx_maintenance_start_date    ON maintenance_records("startDate");"""

new_maintenance = """CREATE TABLE maintenance_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE maintenance_records (
    id                  SERIAL        PRIMARY KEY,
    "vehicleId"         INTEGER       NOT NULL REFERENCES vehicles(id),
    "maintenanceTypeId" INTEGER       NULL REFERENCES maintenance_types(id),
    description         TEXT          NOT NULL,
    odometer            INTEGER       NULL CHECK (odometer >= 0),
    "totalCost"         NUMERIC(12,2) CHECK ("totalCost" >= 0),
    "vendorName"        VARCHAR(255)  NULL,
    location            VARCHAR(255)  NOT NULL,
    "startDate"         TIMESTAMPTZ   NOT NULL,
    "endDate"           TIMESTAMPTZ   NULL,
    "recordedById"      INTEGER       NOT NULL REFERENCES users(id),
    "createdAt"         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_maintenance_dates CHECK ("endDate" IS NULL OR "endDate" > "startDate")
);

CREATE INDEX idx_maintenance_vehicle_id    ON maintenance_records("vehicleId");
CREATE INDEX idx_maintenance_recorded_by_id ON maintenance_records("recordedById");
CREATE INDEX idx_maintenance_start_date    ON maintenance_records("startDate");"""

sql = sql.replace(old_maintenance, new_maintenance)

# 6. attachments table
old_attachments = """CREATE TABLE attachments (
    id             SERIAL       PRIMARY KEY,
    "uploadedById" INTEGER      NOT NULL REFERENCES users(id),
    "vehicleId"    INTEGER      REFERENCES vehicles(id)  ON DELETE CASCADE,
    "roomId"       INTEGER      REFERENCES rooms(id)     ON DELETE CASCADE,
    "bookingId"    INTEGER      REFERENCES bookings(id)  ON DELETE CASCADE,
    "filePath"     VARCHAR(500) NOT NULL,
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
);"""

new_attachments = """CREATE TABLE attachments (
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
);"""

sql = sql.replace(old_attachments, new_attachments)

# 7. Views
sql = sql.replace("""COALESCE(SUM(CASE WHEN fe."fuelType" = 'BBM' THEN fe.liter ELSE 0 END), 0) AS total_liter_bbm,
    COALESCE(SUM(CASE WHEN fe."fuelType" = 'BBM' THEN fe."totalAmount" ELSE 0 END), 0) AS total_cost_bbm,
    COALESCE(SUM(CASE WHEN fe."fuelType" = 'LISTRIK' THEN fe.kwh ELSE 0 END), 0) AS total_kwh_listrik,
    COALESCE(SUM(CASE WHEN fe."fuelType" = 'LISTRIK' THEN fe."totalAmount" ELSE 0 END), 0) AS total_cost_listrik,
    COALESCE(SUM(fe."totalAmount"), 0) AS total_fuel_cost""", """COALESCE(SUM(CASE WHEN ft.type = 'BBM' THEN fe.quantity ELSE 0 END), 0) AS total_liter_bbm,
    COALESCE(SUM(CASE WHEN ft.type = 'BBM' THEN fe."totalCost" ELSE 0 END), 0) AS total_cost_bbm,
    COALESCE(SUM(CASE WHEN ft.type = 'LISTRIK' THEN fe.quantity ELSE 0 END), 0) AS total_kwh_listrik,
    COALESCE(SUM(CASE WHEN ft.type = 'LISTRIK' THEN fe."totalCost" ELSE 0 END), 0) AS total_cost_listrik,
    COALESCE(SUM(fe."totalCost"), 0) AS total_fuel_cost""")
sql = sql.replace("""LEFT JOIN fuel_expenses fe ON fe."vehicleId" = v.id
GROUP BY v.id, r.name, v."plateNumber", vc.name, v.capacity, r.status, v."currentOdometer";""", """LEFT JOIN fuel_expenses fe ON fe."vehicleId" = v.id
LEFT JOIN fuel_types ft ON ft.id = fe."fuelTypeId"
GROUP BY v.id, r.name, v."plateNumber", vc.name, v.capacity, r.status, v."currentOdometer";""")

sql = sql.replace("""COUNT(CASE WHEN fe."fuelType" = 'BBM' THEN 1 END) AS bbm_entries,
    COALESCE(SUM(CASE WHEN fe."fuelType" = 'BBM' THEN fe.liter END), 0) AS total_liter,
    COALESCE(SUM(CASE WHEN fe."fuelType" = 'BBM' THEN fe."totalAmount" END), 0) AS total_cost_bbm,
    COUNT(CASE WHEN fe."fuelType" = 'LISTRIK' THEN 1 END) AS listrik_entries,
    COALESCE(SUM(CASE WHEN fe."fuelType" = 'LISTRIK' THEN fe.kwh END), 0) AS total_kwh,
    COALESCE(SUM(CASE WHEN fe."fuelType" = 'LISTRIK' THEN fe."totalAmount" END), 0) AS total_cost_listrik,
    COALESCE(SUM(fe."totalAmount"), 0) AS grand_total""", """COUNT(CASE WHEN ft.type = 'BBM' THEN 1 END) AS bbm_entries,
    COALESCE(SUM(CASE WHEN ft.type = 'BBM' THEN fe.quantity END), 0) AS total_liter,
    COALESCE(SUM(CASE WHEN ft.type = 'BBM' THEN fe."totalCost" END), 0) AS total_cost_bbm,
    COUNT(CASE WHEN ft.type = 'LISTRIK' THEN 1 END) AS listrik_entries,
    COALESCE(SUM(CASE WHEN ft.type = 'LISTRIK' THEN fe.quantity END), 0) AS total_kwh,
    COALESCE(SUM(CASE WHEN ft.type = 'LISTRIK' THEN fe."totalCost" END), 0) AS total_cost_listrik,
    COALESCE(SUM(fe."totalCost"), 0) AS grand_total""")
sql = sql.replace("""LEFT JOIN fuel_expenses fe ON fe."vehicleId" = v.id
GROUP BY v.id, v."plateNumber", r.name, vc.name;""", """LEFT JOIN fuel_expenses fe ON fe."vehicleId" = v.id
LEFT JOIN fuel_types ft ON ft.id = fe."fuelTypeId"
GROUP BY v.id, v."plateNumber", r.name, vc.name;""")

# 8. Seed data replacements
sql = sql.replace(
    "INSERT INTO vehicle_categories (name) VALUES\n    ('MPV'), ('SUV'), ('Sedan'), ('Pickup'), ('Bus / Minibus'), ('Listrik / EV');",
    "INSERT INTO vehicle_categories (name) VALUES\n    ('MPV'), ('SUV'), ('Sedan'), ('Pickup'), ('Bus / Minibus'), ('Listrik / EV');\n\n"
    "INSERT INTO fuel_types (name, type, unit, default_price, is_active) VALUES\n"
    "    ('Pertalite', 'BBM', 'LITER', 10000.00, TRUE),\n"
    "    ('Pertamax', 'BBM', 'LITER', 13500.00, TRUE),\n"
    "    ('Dexlite', 'BBM', 'LITER', 14500.00, TRUE),\n"
    "    ('Listrik PLN', 'LISTRIK', 'KWH', 2466.00, TRUE);"
)

sql = sql.replace(
    """INSERT INTO vehicles ("resourceId", "plateNumber", brand, model, year, "currentOdometer", "categoryId", capacity) VALUES
    (1, 'B 1234 XY', 'Toyota',     'Avanza',   2022, 15000, 1, 7),
    (2, 'B 5678 AB', 'Honda',      'CR-V',     2021, 28500, 2, 5),
    (3, 'B 9999 CD', 'Toyota',     'Fortuner', 2023,  5200, 2, 7),
    (4, 'B 2222 EF', 'Mitsubishi', 'L300',     2020, 72000, 4, 8),
    (5, 'B 3333 GH', 'Daihatsu',   'Xenia',    2022, 18300, 1, 7),
    (6, 'B 4444 IJ', 'Toyota',     'HiAce',    2019, 95000, 5, 15),
    (7, 'B 5555 EV', 'Hyundai',    'Ioniq 5',  2024,  3200, 6, 5);""",
    """INSERT INTO vehicles ("resourceId", "plateNumber", brand, model, year, "currentOdometer", "categoryId", capacity, energy_type) VALUES
    (1, 'B 1234 XY', 'Toyota',     'Avanza',   2022, 15000, 1, 7, 'BBM'),
    (2, 'B 5678 AB', 'Honda',      'CR-V',     2021, 28500, 2, 5, 'BBM'),
    (3, 'B 9999 CD', 'Toyota',     'Fortuner', 2023,  5200, 2, 7, 'BBM'),
    (4, 'B 2222 EF', 'Mitsubishi', 'L300',     2020, 72000, 4, 8, 'BBM'),
    (5, 'B 3333 GH', 'Daihatsu',   'Xenia',    2022, 18300, 1, 7, 'BBM'),
    (6, 'B 4444 IJ', 'Toyota',     'HiAce',    2019, 95000, 5, 15, 'BBM'),
    (7, 'B 5555 EV', 'Hyundai',    'Ioniq 5',  2024,  3200, 6, 5, 'LISTRIK');"""
)

sql = sql.replace(
    """INSERT INTO fuel_expenses (
    "driverId", "vehicleId", "bookingId", "fuelType",
    liter, "pricePerLiter", "odometerBefore", "odometerAfter",
    "totalAmount", note
) VALUES
    (1, 1, 1, 'BBM', 40.50, 10000.00, 14600, 15000, 405000.00, 'SPBU Pertamina Jl. Sudirman'),
    (1, 1, 6, 'BBM', 35.00, 10000.00, 15000, 15320, 350000.00, 'SPBU Shell Jl. Gatot Subroto'),
    (2, 2, 7, 'BBM', 50.00, 10200.00, 28000, 28500, 510000.00, 'SPBU Pertamina Bekasi');

INSERT INTO fuel_expenses (
    "driverId", "vehicleId", "bookingId", "fuelType",
    kwh, "pricePerKwh", "batteryBefore", "batteryAfter",
    "totalAmount", note
) VALUES
    (1, 7, 9, 'LISTRIK', 45.00, 2466.00, 20.00, 95.00, 110970.00, 'SPKLU PLN Kemayoran — charge 75%');""",
    """INSERT INTO fuel_expenses (
    "driverId", "vehicleId", "bookingId", "fuelTypeId", "recordedById",
    odometer, quantity, "pricePerUnit", "totalCost",
    location, "stationName", note
) VALUES
    (1, 1, 1, 1, 1, 15000, 40.50, 10000.00, 405000.00, 'Jakarta', 'SPBU Pertamina Jl. Sudirman', 'Isi BBM full tank'),
    (1, 1, 6, 1, 1, 15320, 35.00, 10000.00, 350000.00, 'Jakarta', 'SPBU Shell Jl. Gatot Subroto', 'Isi BBM perjalanan'),
    (2, 2, 7, 1, 1, 28500, 50.00, 10200.00, 510000.00, 'Bekasi', 'SPBU Pertamina Bekasi', 'Isi BBM luar kota');

INSERT INTO fuel_expenses (
    "driverId", "vehicleId", "bookingId", "fuelTypeId", "recordedById",
    quantity, "pricePerUnit", "totalCost",
    "batteryBefore", "batteryAfter",
    location, "stationName", note
) VALUES
    (1, 7, 9, 4, 1, 45.00, 2466.00, 110970.00, 20.00, 95.00, 'Jakarta', 'SPKLU PLN Kemayoran', 'Charge 75%');"""
)

sql = sql.replace(
    """INSERT INTO maintenance_records ("resourceId", description, "startDate", "endDate", cost, "createdById") VALUES
    (4,  'Ganti oli mesin, filter oli, dan filter udara — servis berkala 70.000 km',
     NOW() - INTERVAL '2 days', NULL, 850000.00, 1),
    (1,  'Ganti ban depan 2 buah — ban aus',
     NOW() - INTERVAL '20 days', NOW() - INTERVAL '20 days' + INTERVAL '4 hours', 1200000.00, 1),
    (10, 'Perbaikan AC Board Room — kompresor bermasalah',
     NOW() - INTERVAL '7 days', NOW() - INTERVAL '5 days', 2500000.00, 1);""",
    """INSERT INTO maintenance_types (name) VALUES ('Servis Berkala'), ('Ganti Ban'), ('Perbaikan AC');

INSERT INTO maintenance_records ("vehicleId", "maintenanceTypeId", description, "startDate", "endDate", "totalCost", "recordedById", location, odometer, "vendorName") VALUES
    (4, 1, 'Ganti oli mesin, filter oli, dan filter udara — servis berkala 70.000 km',
     NOW() - INTERVAL '2 days', NULL, 850000.00, 1, 'Bengkel Resmi Mitsubishi', 72000, 'Bengkel A'),
    (1, 2, 'Ganti ban depan 2 buah — ban aus',
     NOW() - INTERVAL '20 days', NOW() - INTERVAL '20 days' + INTERVAL '4 hours', 1200000.00, 1, 'Toko Ban Jakarta', 14500, 'Toko Ban B');
    -- Room 10 AC repair is removed because maintenance is now vehicle-only"""
)

with open('sql/schema/000001_init.up.sql', 'w') as f:
    f.write(sql)
print("Updated sql/schema/000001_init.up.sql successfully.")
