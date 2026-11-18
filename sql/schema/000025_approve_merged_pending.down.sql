-- Perbaikan data satu kali; booking yang sudah disetujui tidak dikembalikan.
DELETE FROM data_fixes WHERE key = '000025_approve_merged_pending';
