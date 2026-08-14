-- name: DashboardSummary :one
SELECT
    (SELECT COUNT(*) FROM bookings) AS total_bookings,
    (SELECT COUNT(*) FROM rooms) AS total_rooms,
    (SELECT COUNT(*) FROM rooms ro JOIN resources r ON r.id = ro."resourceId" WHERE r.status = 'AVAILABLE') AS available_rooms,
    (SELECT COUNT(*) FROM drivers WHERE "isActive" = TRUE) AS total_drivers,
    (SELECT COUNT(*) FROM drivers d WHERE d."isActive" = TRUE AND NOT EXISTS (
        SELECT 1 FROM bookings b WHERE b."assignedDriverId" = d.id AND b.status IN ('APPROVED', 'ONGOING')
    )) AS available_drivers,
    (SELECT COUNT(*) FROM vehicles) AS total_vehicles,
    (SELECT COUNT(*) FROM vehicles v JOIN resources r ON r.id = v."resourceId" WHERE r.status = 'AVAILABLE') AS available_vehicles;
