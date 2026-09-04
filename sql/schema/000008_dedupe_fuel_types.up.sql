-- Root cause: the CI migrate step re-runs every sql/schema/*.up.sql file on
-- EVERY push (see .github/workflows/ci-cd.yml), and the seed INSERT INTO
-- fuel_types in 000001_init.up.sql has no unique constraint to lean on for
-- idempotency - so each deploy quietly appended 4 more duplicate rows,
-- which is what showed up as "looping" in Pengaturan > Jenis Bahan Bakar.

-- 1. Re-point any fuel_expenses referencing a duplicate row to the
--    canonical one (lowest id per name) before the duplicates are deleted.
UPDATE fuel_expenses fe
SET "fuelTypeId" = c.keep_id
FROM fuel_types ft
JOIN (SELECT name, MIN(id) AS keep_id FROM fuel_types GROUP BY name) c
  ON c.name = ft.name
WHERE fe."fuelTypeId" = ft.id AND ft.id != c.keep_id;

-- 2. Delete the duplicate rows, keeping one (lowest id) per name.
DELETE FROM fuel_types ft
USING (SELECT name, MIN(id) AS keep_id FROM fuel_types GROUP BY name) c
WHERE ft.name = c.name AND ft.id != c.keep_id;

-- 3. Prevent this from ever happening again.
ALTER TABLE fuel_types ADD CONSTRAINT fuel_types_name_key UNIQUE (name);

-- Same root cause hit `resources` too (its seed INSERT has no unique
-- constraint either), but it never showed up in the Vehicles/Rooms UI
-- because those pages join through `vehicles`/`rooms`, which DO have their
-- own unique constraints (plateNumber, resourceId) that silently blocked
-- the downstream duplicate vehicles/rooms rows - so the orphaned duplicate
-- resources rows just piled up unused. Safe to delete: a resource this
-- seed script created always has a matching vehicles/rooms row when it's
-- actually in use (VehicleService.Delete/RoomService.Delete always cascade
-- through the resource, never leaving a real one orphaned).
DELETE FROM resources r
WHERE r.type = 'VEHICLE' AND NOT EXISTS (SELECT 1 FROM vehicles v WHERE v."resourceId" = r.id);

DELETE FROM resources r
WHERE r.type = 'ROOM' AND NOT EXISTS (SELECT 1 FROM rooms rm WHERE rm."resourceId" = r.id);
