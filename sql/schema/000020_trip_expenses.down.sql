DELETE FROM master_settings WHERE key = 'trip_expense_full_mode';
ALTER TABLE bookings DROP COLUMN IF EXISTS "tollBudget";
DROP TABLE IF EXISTS trip_expenses;
DROP TABLE IF EXISTS trip_reimbursements;
