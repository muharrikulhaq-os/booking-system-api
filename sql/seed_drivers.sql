-- ═══════════════════════════════════════════════════════════════════════════════
-- SEED: 10 Drivers
-- Catatan:
--   • Semua password = bcrypt("Password123!")
--   • roleId = 3  (DRIVER)
--   • departmentId = 4  (Operations)
--   • licenseNumber format: SIM-B1-YYYY-NNN
--   • phoneNumber format: +628XXXXXXXXX
-- ═══════════════════════════════════════════════════════════════════════════════

-- ─── 1. INSERT USERS (role DRIVER) ───────────────────────────────────────────
INSERT INTO users ("employeeId", name, email, password, "isActive", "roleId", "departmentId") VALUES
    ('DRV003', 'Budi Santoso',       'budi.santoso@company.com',       '$2a$10$v8DgunH6sxHTB4GkapmmIOz97bOUKt1u/mfdrI55s6MiLrLeUjlNG', TRUE, 3, 4),
    ('DRV004', 'Agus Prasetyo',      'agus.prasetyo@company.com',      '$2a$10$v8DgunH6sxHTB4GkapmmIOz97bOUKt1u/mfdrI55s6MiLrLeUjlNG', TRUE, 3, 4),
    ('DRV005', 'Hendra Kurniawan',   'hendra.kurniawan@company.com',   '$2a$10$v8DgunH6sxHTB4GkapmmIOz97bOUKt1u/mfdrI55s6MiLrLeUjlNG', TRUE, 3, 4),
    ('DRV006', 'Rizky Firmansyah',   'rizky.firmansyah@company.com',   '$2a$10$v8DgunH6sxHTB4GkapmmIOz97bOUKt1u/mfdrI55s6MiLrLeUjlNG', TRUE, 3, 4),
    ('DRV007', 'Deni Wahyudi',       'deni.wahyudi@company.com',       '$2a$10$v8DgunH6sxHTB4GkapmmIOz97bOUKt1u/mfdrI55s6MiLrLeUjlNG', TRUE, 3, 4),
    ('DRV008', 'Fajar Nugroho',      'fajar.nugroho@company.com',      '$2a$10$v8DgunH6sxHTB4GkapmmIOz97bOUKt1u/mfdrI55s6MiLrLeUjlNG', TRUE, 3, 4),
    ('DRV009', 'Slamet Riyadi',      'slamet.riyadi@company.co  m',      '$2a$10$v8DgunH6sxHTB4GkapmmIOz97bOUKt1u/mfdrI55s6MiLrLeUjlNG', TRUE, 3, 4),
    ('DRV010', 'Wahyu Hidayat',      'wahyu.hidayat@company.com',      '$2a$10$v8DgunH6sxHTB4GkapmmIOz97bOUKt1u/mfdrI55s6MiLrLeUjlNG', TRUE, 3, 4),
    ('DRV011', 'Taufik Hermawan',    'taufik.hermawan@company.com',    '$2a$10$v8DgunH6sxHTB4GkapmmIOz97bOUKt1u/mfdrI55s6MiLrLeUjlNG', TRUE, 3, 4),
    ('DRV012', 'Yusuf Abdillah',     'yusuf.abdillah@company.com',     '$2a$10$v8DgunH6sxHTB4GkapmmIOz97bOUKt1u/mfdrI55s6MiLrLeUjlNG', TRUE, 3, 4);

-- ─── 2. INSERT DRIVERS ───────────────────────────────────────────────────────
-- userId diambil dari users yang baru saja di-insert.
-- Menggunakan subquery agar tidak bergantung pada urutan ID auto-increment.
INSERT INTO drivers ("userId", "licenseNumber", "phoneNumber", "isActive")
SELECT u.id,
       CASE u."employeeId"
           WHEN 'DRV003' THEN 'SIM-B1-2024-003'
           WHEN 'DRV004' THEN 'SIM-B1-2024-004'
           WHEN 'DRV005' THEN 'SIM-B1-2024-005'
           WHEN 'DRV006' THEN 'SIM-B1-2024-006'
           WHEN 'DRV007' THEN 'SIM-B1-2024-007'
           WHEN 'DRV008' THEN 'SIM-B2-2024-001'
           WHEN 'DRV009' THEN 'SIM-B2-2024-002'
           WHEN 'DRV010' THEN 'SIM-B2-2024-003'
           WHEN 'DRV011' THEN 'SIM-B2-2024-004'
           WHEN 'DRV012' THEN 'SIM-B2-2024-005'
       END,
       CASE u."employeeId"
           WHEN 'DRV003' THEN '+6281311110001'
           WHEN 'DRV004' THEN '+6281311110002'
           WHEN 'DRV005' THEN '+6281311110003'
           WHEN 'DRV006' THEN '+6281311110004'
           WHEN 'DRV007' THEN '+6281311110005'
           WHEN 'DRV008' THEN '+6281311110006'
           WHEN 'DRV009' THEN '+6281311110007'
           WHEN 'DRV010' THEN '+6281311110008'
           WHEN 'DRV011' THEN '+6281311110009'
           WHEN 'DRV012' THEN '+6281311110010'
       END,
       TRUE
FROM users u
WHERE u."employeeId" IN (
    'DRV003','DRV004','DRV005','DRV006','DRV007',
    'DRV008','DRV009','DRV010','DRV011','DRV012'
);

-- ─── VERIFIKASI ──────────────────────────────────────────────────────────────
-- Jalankan query berikut untuk memastikan data berhasil di-insert:
-- SELECT d.id, u."employeeId", u.name, u.email, d."licenseNumber", d."phoneNumber", d."isActive"
-- FROM drivers d
-- JOIN users u ON u.id = d."userId"
-- ORDER BY d.id;
