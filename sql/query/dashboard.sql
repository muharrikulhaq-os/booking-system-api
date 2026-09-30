-- name: DashboardSummary :one
SELECT
    (SELECT COUNT(*) FROM bookings) AS total_bookings,
    (SELECT COUNT(*) FROM rooms) AS total_rooms,
    (SELECT COUNT(*) FROM rooms ro JOIN resources r ON r.id = ro."resourceId" WHERE r.status = 'AVAILABLE') AS available_rooms,
    (SELECT COUNT(*) FROM drivers d JOIN users u ON u.id = d."userId" WHERE d."isActive" = TRUE AND u."isActive" = TRUE) AS total_drivers,
    -- Sama dengan picker & pemilihan otomatis (GetFreeDriver): akun aktif &
    -- tidak sedang memegang kendaraan (driver_assignments terbuka).
    (SELECT COUNT(*) FROM drivers d JOIN users u ON u.id = d."userId" WHERE d."isActive" = TRUE AND u."isActive" = TRUE AND NOT EXISTS (
        SELECT 1 FROM driver_assignments da WHERE da."driverId" = d.id AND da."releasedAt" IS NULL
    )) AS available_drivers,
    (SELECT COUNT(*) FROM vehicles) AS total_vehicles,
    (SELECT COUNT(*) FROM vehicles v JOIN resources r ON r.id = v."resourceId" WHERE r.status = 'AVAILABLE') AS available_vehicles;
