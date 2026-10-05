-- Pengaturan lembur supir. Jam akhir kerja = jadwal selesai booking; lembur
-- dicatat bila kendaraan kembali MELEBIHI ambang (jam) setelahnya — begitu
-- terlewati, dihitung penuh sejak jam akhir kerja. Nilai awal = perilaku
-- lama: ambang 0 jam, SPD tidak dihitung lembur.
-- Idempoten: dijalankan ulang tiap deploy (psql -f).
INSERT INTO master_settings (key, value, unit, description) VALUES
    ('overtime_threshold_hours_non_spd', 0, 'jam',   'Lembur Non-SPD dihitung bila kembali melebihi sekian jam dari jadwal selesai'),
    ('overtime_spd_enabled',             0, 'bool',  'Lembur juga dihitung untuk booking SPD (1 = ya, 0 = tidak)'),
    ('overtime_spd_same_as_non_spd',     1, 'bool',  'Lembur SPD memakai aturan yang sama dengan Non-SPD (1 = ya, 0 = pakai ambang SPD sendiri)'),
    ('overtime_threshold_hours_spd',     0, 'jam',   'Lembur SPD dihitung bila kembali melebihi sekian jam dari jadwal selesai (bila aturan SPD terpisah)')
ON CONFLICT (key) DO NOTHING;
