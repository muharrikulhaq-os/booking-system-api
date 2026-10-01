-- Membatalkan struktur maintenance vendor. Status lama tidak dikembalikan
-- persis (DRAFT/SUBMITTED/SCHEDULED/IN_PROGRESS → pending, COMPLETED → completed).
ALTER TABLE maintenance_records DROP CONSTRAINT IF EXISTS chk_maintenance_status;
UPDATE maintenance_records SET status = CASE WHEN status = 'COMPLETED' THEN 'completed' ELSE 'pending' END;
ALTER TABLE maintenance_records ALTER COLUMN status SET DEFAULT 'ONGOING';

DROP TABLE IF EXISTS maintenance_documents;
ALTER TABLE vehicle_issue_reports DROP CONSTRAINT IF EXISTS fk_issue_reports_maintenance;
ALTER TABLE maintenance_records
    DROP COLUMN IF EXISTS "requestNo", DROP COLUMN IF EXISTS "vendorId", DROP COLUMN IF EXISTS category,
    DROP COLUMN IF EXISTS complaint, DROP COLUMN IF EXISTS "plannedDate", DROP COLUMN IF EXISTS "estimatedDays",
    DROP COLUMN IF EXISTS "scheduledDate", DROP COLUMN IF EXISTS "pickupMethod", DROP COLUMN IF EXISTS "estimatedCost",
    DROP COLUMN IF EXISTS "costBearer", DROP COLUMN IF EXISTS "sourceIssueId", DROP COLUMN IF EXISTS "submittedAt",
    DROP COLUMN IF EXISTS "scheduleNote", DROP COLUMN IF EXISTS "handoverAt", DROP COLUMN IF EXISTS "handoverOdometer",
    DROP COLUMN IF EXISTS "handoverFuelLevel", DROP COLUMN IF EXISTS "handoverReceiverName",
    DROP COLUMN IF EXISTS "handoverChecklist", DROP COLUMN IF EXISTS "handoverNote", DROP COLUMN IF EXISTS "returnedAt",
    DROP COLUMN IF EXISTS "returnOdometer", DROP COLUMN IF EXISTS "returnFuelLevel", DROP COLUMN IF EXISTS "returnHandlerName",
    DROP COLUMN IF EXISTS "returnChecklist", DROP COLUMN IF EXISTS "workDone", DROP COLUMN IF EXISTS "partsReplaced",
    DROP COLUMN IF EXISTS "returnNote", DROP COLUMN IF EXISTS "cancelledAt", DROP COLUMN IF EXISTS "cancelReason",
    DROP COLUMN IF EXISTS "updatedAt";
DROP TABLE IF EXISTS vehicle_issue_reports;
DROP TABLE IF EXISTS document_counters;
DROP TABLE IF EXISTS document_settings;
ALTER TABLE vehicles DROP CONSTRAINT IF EXISTS chk_vehicles_ownership;
ALTER TABLE vehicles DROP COLUMN IF EXISTS ownership, DROP COLUMN IF EXISTS "ownerVendorId",
    DROP COLUMN IF EXISTS "rentalContractNo";
DROP TABLE IF EXISTS vendors;
DELETE FROM data_fixes WHERE key = '000016_maintenance_vendor';
