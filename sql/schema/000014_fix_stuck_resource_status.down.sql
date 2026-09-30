-- Pembersihan data satu kali; status yang sudah dibetulkan tidak dikembalikan.
DELETE FROM data_fixes WHERE key = '000014_fix_stuck_resource_status';
