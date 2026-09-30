-- Pembersihan data satu kali; penugasan yang sudah dilepas tidak dikembalikan.
-- Menghapus penanda saja (pembersihan akan berjalan lagi pada migrasi berikutnya).
DELETE FROM data_fixes WHERE key = '000013_release_stuck_driver_assignments';
