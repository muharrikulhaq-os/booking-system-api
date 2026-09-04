-- Deleted duplicate rows can't be un-deleted; this only reverses the
-- constraint so a rollback doesn't leave the schema in an inconsistent state.
ALTER TABLE fuel_types DROP CONSTRAINT IF EXISTS fuel_types_name_key;
