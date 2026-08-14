CREATE OR REPLACE VIEW v_driver_ratings_summary AS
SELECT 
    d.id AS driver_id, u.name AS driver_name, u."employeeId", d."isActive",
    COUNT(dr.id) AS total_ratings,
    COALESCE(ROUND(AVG(dr.rating)::NUMERIC, 2), 0) AS average_rating,
    SUM(CASE WHEN dr.rating = 5 THEN 1 ELSE 0 END) AS bintang_5,
    SUM(CASE WHEN dr.rating = 4 THEN 1 ELSE 0 END) AS bintang_4,
    SUM(CASE WHEN dr.rating = 3 THEN 1 ELSE 0 END) AS bintang_3,
    SUM(CASE WHEN dr.rating = 2 THEN 1 ELSE 0 END) AS bintang_2,
    SUM(CASE WHEN dr.rating = 1 THEN 1 ELSE 0 END) AS bintang_1
FROM drivers d
JOIN users u ON u.id = d."userId"
LEFT JOIN driver_ratings dr ON dr."driverId" = d.id
GROUP BY d.id, u.name, u."employeeId", d."isActive";
