-- Ubah penumpang (karyawan/tamu yang ikut) setelah booking disetujui.
-- 0 = terkunci begitu disetujui (bawaan); 1 = boleh sampai perjalanan dimulai.
-- Diatur admin di menu Pengaturan web (kartu "Penumpang Booking").
INSERT INTO master_settings (key, value, unit, description) VALUES
    ('booking_passenger_edit_after_approval', 0, 'bool',
     'Penumpang booking kendaraan boleh diubah setelah disetujui (1 = boleh sampai perjalanan dimulai, 0 = terkunci)')
ON CONFLICT (key) DO NOTHING;
