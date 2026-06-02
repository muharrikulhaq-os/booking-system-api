-- name: DashboardSummary :one
SELECT
    (SELECT COUNT(*) FROM bookings) AS total_bookings,
    (SELECT COUNT(*) FROM rooms) AS total_rooms,
    (SELECT COUNT(*) FROM drivers) AS total_drivers,
    (SELECT COUNT(*) FROM vehicles) AS total_vehicles;
