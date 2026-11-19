-- Pengaturan "ubah penumpang setelah disetujui" DIBATALKAN (keputusan pemilik
-- produk 2026-11-19): penumpang selalu terkunci begitu booking disetujui.
-- Baris yang sempat dipasang versi sebelumnya dihapus (idempoten).
DELETE FROM master_settings WHERE key = 'booking_passenger_edit_after_approval';
