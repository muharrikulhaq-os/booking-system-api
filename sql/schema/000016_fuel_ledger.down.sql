ALTER TABLE IF EXISTS fuel_ledger DROP COLUMN IF EXISTS "voucherId";
ALTER TABLE IF EXISTS fuel_expenses DROP COLUMN IF EXISTS "voucherId";
DROP TABLE IF EXISTS fuel_vouchers;
DROP TYPE IF EXISTS fuel_voucher_status;
DELETE FROM master_settings WHERE key = 'fuel_voucher_validity_days';
DROP TABLE IF EXISTS fuel_ledger;
DROP TYPE IF EXISTS fuel_ledger_type;

ALTER TABLE fuel_expenses
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS "stationId",
    DROP COLUMN IF EXISTS reason,
    DROP COLUMN IF EXISTS "voidedAt",
    DROP COLUMN IF EXISTS "voidedById",
    DROP COLUMN IF EXISTS "voidReason",
    DROP COLUMN IF EXISTS "meterStartKwh",
    DROP COLUMN IF EXISTS "meterEndKwh",
    DROP COLUMN IF EXISTS "quantitySource";

DROP TABLE IF EXISTS fuel_stations;

DROP TRIGGER IF EXISTS set_vehicle_fuel_defaults ON vehicles;
DROP FUNCTION IF EXISTS trigger_vehicle_fuel_defaults();
DROP FUNCTION IF EXISTS default_km_per_liter(TEXT);

ALTER TABLE vehicles
    DROP COLUMN IF EXISTS "kmPerLiter",
    DROP COLUMN IF EXISTS "tankCapacityLiter",
    DROP COLUMN IF EXISTS "kmPerKwh",
    DROP COLUMN IF EXISTS "batteryCapacityKwh",
    DROP COLUMN IF EXISTS "fuelBaselineOdometer";

DELETE FROM data_fixes WHERE key = '000016_vehicle_fuel_profile';
-- View v_vehicle_summary / v_fuel_expense_summary dikembalikan oleh 000001.
