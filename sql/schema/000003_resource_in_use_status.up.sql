-- Resource dianggap IN_USE selama booking-nya berstatus ONGOING (di-set otomatis
-- oleh booking_service.Start/Complete). Nilai ini sengaja TIDAK ditambahkan ke
-- validator manual PATCH /vehicles|rooms/:id/status (oneof=AVAILABLE MAINTENANCE
-- INACTIVE) — hanya sistem yang boleh mengubahnya, supaya tidak desync dari siklus
-- booking.
ALTER TYPE resource_status ADD VALUE 'IN_USE';
