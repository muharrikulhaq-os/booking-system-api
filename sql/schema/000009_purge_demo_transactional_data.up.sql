-- The 000001 seed data for bookings/fuel_expenses/driver_ratings/
-- maintenance_records/audit_logs/driver_assignments/approval_logs used
-- relative dates (NOW() ± INTERVAL) and none of those tables had a unique
-- constraint, so every deploy this session re-inserted a fresh, slightly
-- different-looking copy of the same demo scenario (John Doe / Jane Smith /
-- Pak Supir Satu / etc.) - unlike fuel_types, these can't be deduped by
-- exact-match GROUP BY since the timestamps differ per run. This purges
-- every row that traces back to those known demo identities, root and all.
-- The seed INSERTs themselves were removed from 000001, so this data will
-- not come back on the next deploy.

-- Demo users are identified by their known seed emails - stable, since
-- users.email is UNIQUE and was never duplicated by this bug.
-- Demo vehicles are identified by their known seed plate numbers, same
-- reasoning (vehicles.plateNumber is UNIQUE).

-- fuel_expenses first - its FK to bookings has no ON DELETE CASCADE.
DELETE FROM fuel_expenses fe
WHERE fe."vehicleId" IN (
    SELECT id FROM vehicles WHERE "plateNumber" IN (
        'B 1234 XY', 'B 5678 AB', 'B 9999 CD', 'B 2222 EF',
        'B 3333 GH', 'B 4444 IJ', 'B 5555 EV'
    )
);

-- booking_merges - its FK to bookings has no ON DELETE CASCADE either.
-- Defensive: the demo bookings were never merged by the seed script itself,
-- but this covers it in case a merge was tested against one of them later.
DELETE FROM booking_merges bm
WHERE bm."primaryBookingId" IN (
    SELECT id FROM bookings WHERE "userId" IN (
        SELECT id FROM users WHERE email IN (
            'admin@company.com', 'john.doe@company.com', 'jane.smith@company.com',
            'dewi.lestari@company.com', 'andi.supriadi@company.com',
            'supir1@company.com', 'supir2@company.com'
        )
    )
) OR bm."mergedBookingId" IN (
    SELECT id FROM bookings WHERE "userId" IN (
        SELECT id FROM users WHERE email IN (
            'admin@company.com', 'john.doe@company.com', 'jane.smith@company.com',
            'dewi.lestari@company.com', 'andi.supriadi@company.com',
            'supir1@company.com', 'supir2@company.com'
        )
    )
);

-- bookings - approval_logs and driver_ratings cascade-delete automatically
-- (both declared ON DELETE CASCADE on bookingId).
DELETE FROM bookings b
WHERE b."userId" IN (
    SELECT id FROM users WHERE email IN (
        'admin@company.com', 'john.doe@company.com', 'jane.smith@company.com',
        'dewi.lestari@company.com', 'andi.supriadi@company.com',
        'supir1@company.com', 'supir2@company.com'
    )
);

-- maintenance_records - independent of bookings.
DELETE FROM maintenance_records mr
WHERE mr."vehicleId" IN (
    SELECT id FROM vehicles WHERE "plateNumber" IN (
        'B 1234 XY', 'B 5678 AB', 'B 9999 CD', 'B 2222 EF',
        'B 3333 GH', 'B 4444 IJ', 'B 5555 EV'
    )
);

-- audit_logs - no FK at all (entityId is a loose polymorphic reference),
-- so match by the acting userId instead.
DELETE FROM audit_logs al
WHERE al."userId" IN (
    SELECT id FROM users WHERE email IN (
        'admin@company.com', 'john.doe@company.com', 'jane.smith@company.com',
        'dewi.lestari@company.com', 'andi.supriadi@company.com',
        'supir1@company.com', 'supir2@company.com'
    )
);

-- driver_assignments - independent, match by the demo drivers.
DELETE FROM driver_assignments da
WHERE da."driverId" IN (
    SELECT d.id FROM drivers d
    JOIN users u ON u.id = d."userId"
    WHERE u.email IN ('supir1@company.com', 'supir2@company.com')
);
