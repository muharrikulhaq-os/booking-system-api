# Hasil Testing Alur — 2026-09-30

Pengujian otomatis skenario di [SKENARIO_TESTING.md](SKENARIO_TESTING.md) terhadap **backend
`develop` (commit `aed39c6`)** yang dijalankan LOKAL (PostgreSQL 18, database `kce_test` hasil
semua migrasi, SMTP & FCM dimatikan). Tidak ada data production yang disentuh.

| Hasil | Jumlah |
|---|---|
| ✅ Lulus | 146 |
| ❌ Gagal (bug terkonfirmasi) | 32 |
| ℹ️ Perlu keputusan bisnis | 10 |
| ⏭️ Dilewati (OTP email, laporan periode — dicakup unit test) | 3 |
| 💥 Error runner | 0 |

> **Setelah semua perbaikan B1–B21 (runner terakhir, 2026-09-30):** ✅ 178 lulus · ❌ 0 gagal · ℹ️ 9 perlu keputusan · ⏭️ 3 dilewati.

Ulangi pengujian (mis. setelah perbaikan): lihat [`tests/skenario/README.md`](../tests/skenario/README.md).

## Bug terkonfirmasi (urut prioritas)

> ✅ **Sudah diperbaiki (2026-09-30):** B1, B2, B3 (AU-16, AU-18, RL-07 lulus) · B4, B5, B6, B7 (TO-03, TO-03b, AS-01..04, SB-02, SB-03, SP-12 lulus) · B10, B11, B12 (CP-10, MT-04, MT-07, MT-08, MT-09, VH-02, BC-07 lulus; + MT-18, VH-11 baru) · B8, B9, B13–B21 (BC-11, MG-09, AP-11, AP-12, AP-15, DL-01, BC-04, BC-08, VH-01, TO-05, VH-07, RM-03, RL-08 lulus) — diverifikasi ulang dengan runner.

| # | Bug | Skenario | Letak kode | Dampak nyata |
|---|---|---|---|---|
| B1 | 🔒 `POST /auth/register` publik bisa membuat akun **ADMIN** (`roleId` bebas) | AU-16 | `internal/service/auth_service.go` `Register` | Siapa pun bisa jadi admin |
| B2 | 🔒 DRIVER / ROOM_KEEPER bisa **membatalkan booking PENDING milik orang lain** | RL-07 | `booking_service.go` `Cancel` (cek pemilik hanya untuk EMPLOYEE) | Penyalahgunaan lewat API |
| B3 | Dua login akun yang sama di detik yang sama → **500** (refresh token kembar melanggar unique) | AU-18 | refresh token deterministik per detik; tabel `refresh_tokens.token` unique | Klik ganda / web+HP login bersamaan gagal |
| B4 | Booking APPROVED yang **EXPIRED tidak melepas supir** → supir "sibuk" selamanya, kendaraannya menolak supir lain | TO-03, TO-03b | `sweepStaleBookings` + `MarkExpiredBookings` | Supir hilang dari pemilihan otomatis; approve gagal "dipegang supir lain" |
| B5 | **Tugaskan / pindah supir & kendaraan** tidak memperbarui catatan supir pemegang kendaraan | AS-01, AS-02b, AS-03, AS-04 | `booking_service.go` `AssignVehicle` | Supir lama tertahan setelah trip selesai; supir baru tidak tercatat |
| B6 | **Alihkan resource (substitute)** hanya mengganti resource; kendaraan ditugaskan & supir tetap di kendaraan lama; penanda "Dialihkan" kosong | SB-02, SB-03 | `SubstituteResource` + query `UpdateBookingResource` | Bentrok dicek di kendaraan yang salah |
| B7 | Supir yang sama bisa disetujui untuk **dua perjalanan bentrok** di kendaraan berbeda | SP-12 | `Approve` (tidak ada cek bentrok jadwal supir NON_SPD) | Satu supir dijadwalkan di dua tempat |
| B8 | **Dua rapat bentrok di ruangan yang sama** sama-sama disetujui | BC-11 | `Create` / `Approve` (tidak ada cek bentrok ruangan) | Ruangan double-book |
| B9 | **Gabung + ganti supir** → kendaraan booking utama jadi kosong (null) | MG-09 | `MergeBookings` (kendaraan diambil dari penugasan supir baru yang kosong) | Trip gabungan kehilangan kendaraan |
| B10 | Status kendaraan **tertimpa**: selesai booking → AVAILABLE walau maintenance otomatis terbuka; hapus maintenance saat trip → AVAILABLE; admin bisa set AVAILABLE saat IN_USE | CP-10, MT-09, VH-02 | `Complete`, `MaintenanceService.Delete`, `VehicleService.UpdateStatus` | Kendaraan rusak/dipakai tampak tersedia |
| B11 | Maintenance **terjadwal minggu depan langsung mengunci kendaraan hari ini** | MT-04, BC-07 | `MaintenanceService.Create` (status MAINTENANCE tanpa melihat tanggal) | Kendaraan tidak bisa dibooking tanpa alasan |
| B12 | Maintenance "selesai" lewat **edit** atau **dibuat langsung selesai** tetap mengunci kendaraan (dan tidak bisa diselesaikan lagi) | MT-07, MT-08 | `MaintenanceService.Update` / `Create` | Kendaraan terkunci permanen |
| B13 | **Log persetujuan tidak pernah tersimpan** untuk APPROVE (0 dari 185) | AP-15, AP-12 | `booking_service.go` `Approve`: `Action: "APPROVE"` vs enum `approval_action` (`APPROVED`) — error diabaikan | Riwayat/performa persetujuan kosong |
| B14 | Dashboard **"supir tersedia" beda** dengan picker & pemilihan otomatis (53 vs 49) | DL-01 | `sql/query/dashboard.sql` (abaikan akun nonaktif & penugasan) | Angka dashboard menyesatkan |
| B15 | Booking **tanggal lampau** diterima | BC-04 | `Create` | Data kotor, langsung hangus |
| B16 | Kendaraan **INACTIVE** bisa dibooking | BC-08, VH-01 | `Create` (hanya menolak MAINTENANCE) | Kendaraan nonaktif tetap dipakai |
| B17 | Transisi otomatis (IGNORED/EXPIRED/OVERDUE) **baru terjadi saat daftar booking dibuka** | TO-05 | `List` → `sweepStaleBookings` | Detail/dashboard/perangkat lain menampilkan status lama |
| B18 | Hapus kendaraan yang punya riwayat booking → **500** | VH-07 | `VehicleService.Delete` (FK tidak ditangani) | Pesan error tidak jelas |
| B19 | Mobile memberi penjaga ruangan tombol ubah status ruangan, backend menolak **403** | RM-03 | `room_handler.go` (`admin`) vs mobile `permissions.dart` (`roomManage`) | Tombol selalu gagal |
| B20 | EMPLOYEE lolos otorisasi pencatatan BBM (API) | RL-08 | route `POST /fuel-expenses` tanpa role | Data BBM dari pihak tak berwenang |
| B21 | Admin boleh **menyetujui** booking sendiri, tapi tidak boleh **menolaknya** | AP-11 | `Approve` (cek self-approval hanya di `Reject`) | Aturan tidak konsisten |

## Dugaan yang terbantah
- **VH-10** — kendaraan baru ternyata mendapat baseline servis = odometer saat didaftarkan (nilai 0 hanya ada di data seed). Bukan bug.

## Perlu keputusan bisnis (ℹ️)
| Skenario | Perilaku saat ini |
|---|---|
| CN-03 / CN-04 | Booking APPROVED tidak bisa dibatalkan siapa pun (supir tetap memegang kendaraan) |
| ST-07 | Admin bisa memulai booking kendaraan **tanpa supir** lewat API |
| ST-11 | Penjaga ruangan mana pun bisa memulai booking ruangan yang bukan jagaannya |
| SP-13 | Pemilihan otomatis bisa mengambil **supir tetap kendaraan lain** padahal ada supir bebas |
| DU-03 | Supir dinonaktifkan padahal punya booking APPROVED → booking tetap ke supir nonaktif, tanpa peringatan |
| AU-05b | Access token akun yang baru dinonaktifkan tetap berlaku sampai kedaluwarsa (±15 menit) |
| FL-04 | API menerima isi "Listrik" untuk kendaraan BBM (UI mengunci) |
| FL-06 | Hapus catatan BBM tidak memundurkan odometer kendaraan |
| MT-15 | Maintenance otomatis berstatus `ongoing`; enum mobile hanya `pending`/`completed` → cek tampilan di HP |

## Belum diuji di putaran ini
- **Tampilan & sinkronisasi di aplikasi** (SY-01..13 selain header, TZ tampilan, NT push/FCM, semua 📱): butuh web/mobile yang berjalan — putaran berikutnya lewat browser (web lokal) dan HP.
- **OTP lupa password** (AU-10/11): SMTP sengaja dimatikan di lingkungan test.

## Hasil lengkap

| ID | Hasil | Catatan |
|---|---|---|
| AU-01 | ✅ PASS | Login semua role OK (ADM=ADMIN, EMPA=EMPLOYEE, RK1=ROOM_KEEPER, DRV=DRIVER) |
| AU-02 | ✅ PASS | Password salah → 401 "incorrect password" |
| AU-03 | ✅ PASS | Email tidak terdaftar → 401 "email didnt match any accounts" |
| AU-04 | ✅ PASS | Login akun nonaktif → 403 "account is inactive" |
| AU-05 | ✅ PASS | Refresh token akun yang baru dinonaktifkan → 403 "account is inactive" |
| AU-05b | ℹ️ INFO | Access token lama akun nonaktif masih dipakai /auth/me → 200 (berlaku sampai kedaluwarsa) |
| AU-08 | ✅ PASS | Refresh setelah logout → 401 |
| AU-10 | ⏭️ SKIP | OTP lewat email — SMTP sengaja dimatikan di lingkungan test |
| AU-11 | ⏭️ SKIP | OTP lewat email — SMTP sengaja dimatikan di lingkungan test |
| AU-12 | ✅ PASS | Ganti password dengan password lama salah → 401 "unauthorized" |
| AU-16 | ❌ FAIL | Register publik BERHASIL membuat akun ADMIN (role=ADMIN) |
| AU-18 | ❌ FAIL | Dua login di detik yang sama → 200, 500: refresh token kembar melanggar unique constraint |
| RL-01 | ✅ PASS | DRIVER buat booking → 403, ROOM_KEEPER → 403 |
| RL-02 | ✅ PASS | EMPB buka booking EMPA → 403 |
| RL-03 | ✅ PASS | Supir lain buka booking yang bukan tugasnya → 403 |
| RL-04 | ✅ PASS | Daftar EMPA: 1 booking, milik orang lain: 0 |
| RL-05 | ✅ PASS | Daftar supir: 1 tugas, bukan miliknya: 0 |
| RL-06 | ✅ PASS | Kalender resource dibuka EMPB memuat booking EMPA: true |
| RL-07 | ❌ FAIL | Supir (200) & penjaga (200) BISA membatalkan booking PENDING milik EMPA |
| RL-08 | ❌ FAIL | EMPLOYEE lolos otorisasi catat BBM (400 "proofPhoto is required") |
| RL-09 | ✅ PASS | Laporan 403, maintenance 403, pengguna(oleh supir) 403 |
| BC-01 | ✅ PASS | PENDING, notif admin=true, X-Data-Changed=booking |
| BC-02 | ✅ PASS | Booking ruangan → 201 PENDING |
| BC-03 | ✅ PASS | Jam selesai < mulai → 400 "end date must be after start date" |
| BC-04 | ❌ FAIL | Booking KEMARIN diterima (201 PENDING) |
| BC-05 | ✅ PASS | Kendaraan MAINTENANCE → 409 "resource is currently under maintenance" |
| BC-06 | ✅ PASS | Booking di tanggal maintenance → 409 "kendaraan ini sedang/akan menjalani maintenance pada tanggal tersebut" |
| BC-07 | ❌ FAIL | Booking H+5 DITOLAK 409 "resource is currently under maintenance" — kendaraan sudah berstatus MAINTENANCE sejak maintenance terjadwal dibuat |
| BC-08 | ❌ FAIL | Kendaraan INACTIVE BISA dibooking (201 PENDING) |
| BC-09 | ✅ PASS | Kendaraan IN_USE sekarang, booking besok → 201 |
| BC-10 | ✅ PASS | Dua booking PENDING kendaraan & jam sama → 201, 201 |
| BC-11 | ❌ FAIL | DUA rapat bentrok di ruangan yang sama sama-sama DISETUJUI (200, 200) |
| BC-12 | ✅ PASS | 10 penumpang di kapasitas 6: dibuat, approve + peringatan "Warning: Vehicle capacity overload! (Remaining capacity is negative). Please substitute vehicle if needed." |
| BC-13 | ✅ PASS | Pilih supir nonaktif → 400 "supir yang dipilih tidak aktif" |
| BC-14 | ✅ PASS | Kendaraan SPD pagi, booking sore hari yang sama → 409 "kendaraan ini sedang bertugas SPD pada tanggal tersebut" |
| BC-15 | ✅ PASS | SPD baru di hari yang sudah terisi NON_SPD → 409 "kendaraan ini sedang bertugas SPD pada tanggal tersebut" |
| BC-16 | ✅ PASS | SPD lintas malam memblokir hari berikutnya → 409 "kendaraan ini sedang bertugas SPD pada tanggal tersebut" |
| BC-17 | ✅ PASS | Pilih supir yang memegang kendaraan lain → kendaraan booking = kendaraan supir (true) |
| SP-01 | ✅ PASS | Tanpa pilih supir → supir tetap (DRVSPF o21v3) |
| SP-02 | ✅ PASS | Pilih supir lain → tetap supir tetap (DRVSPF o21v3) |
| SP-03 | ✅ PASS | Supir kosong otomatis → DRVFREE o21v3 |
| SP-04 | ✅ PASS | Tidak ada supir kosong → PENDING tanpa supir (201, supir=kosong) |
| SP-05 | ✅ PASS | Supir nonaktif (menu Driver) tidak dipilih otomatis (supir=kosong) |
| SP-06 | ✅ PASS | Akun supir nonaktif (menu Pengguna): tidak otomatis & tidak di picker |
| SP-07 | ✅ PASS | Supir tetap sedang SPD di kendaraan lain → 409 "kendaraan ini sedang bertugas SPD pada tanggal tersebut" |
| SP-08 | ✅ PASS | Pindah supir tetap dari menu Kendaraan tersinkron di menu Driver |
| SP-09 | ✅ PASS | Ubah kendaraan tetap dari menu Driver: kendaraan lama kosong, baru terisi |
| SP-10 | ✅ PASS | Hapus supir tetap → kosong |
| SP-11 | ✅ PASS | Picker: plat=T o21v3119 KCE, sisa kursi=2, tujuan bentrok="Uji skenario" |
| SP-12 | ❌ FAIL | Supir yang SAMA disetujui untuk 2 perjalanan bentrok di kendaraan berbeda (200) |
| SP-13 | ℹ️ INFO | Pemilihan otomatis mengambil supir tetap kendaraan LAIN (DRVSP13F o21v3) padahal ada supir bebas |
| AP-01 | ✅ PASS | APPROVED, supir memegang kendaraan=true, notif pemohon=true, notif supir=true, X-Data-Changed=booking |
| AP-02 | ✅ PASS | Approve ulang → 400 "booking is not in PENDING status" |
| AP-03 | ✅ PASS | Kendaraan sudah APPROVED di jam bentrok → 409 "kendaraan ini sudah dipakai booking lain yang bentrok jadwalnya - gabungkan (merge) atau alihkan ke kendaraan lain sebelum menyetujui" |
| AP-04 | ✅ PASS | Kendaraan dipegang supir lain (booking hari lain) → 409 "kendaraan ini sedang dipegang supir lain - gabungkan (merge) atau alihkan ke kendaraan lain sebelum menyetujui" |
| AP-05 | ✅ PASS | Maintenance dijadwalkan setelah booking PENDING → approve 409 "kendaraan ini sedang/akan menjalani maintenance pada tanggal tersebut" |
| AP-07 | ✅ PASS | Penjaga ruangan dapat notifikasi "Ruangan dipesan" |
| AP-08 | ✅ PASS | Tolak tanpa catatan → 422 |
| AP-09 | ✅ PASS | REJECTED, notif pemohon=true, timeline REJECT=true |
| AP-10 | ✅ PASS | Admin menolak booking sendiri → 403 "you cannot approve your own booking" |
| AP-11 | ❌ FAIL | Admin BISA menyetujui booking sendiri (200) padahal menolaknya dilarang |
| AP-12 | ❌ FAIL | Dua admin approve bersamaan → 200, 400; log persetujuan tercatat 0x |
| AP-14 | ✅ PASS | Booking tanpa supir bisa disetujui (menunggu penugasan) |
| AP-15 | ❌ FAIL | 185 booking disetujui, tapi log persetujuan APPROVE = 0 (insert "APPROVE" ditolak enum approval_action) |
| SB-01 | ✅ PASS | Dialihkan & disetujui di kendaraan baru, notif pemohon=true |
| SB-02 | ❌ FAIL | Resource = kendaraan BARU, tapi kendaraan ditugaskan = LAMA & supir memegang kendaraan LAMA |
| SB-03 | ❌ FAIL | Penanda dialihkan tidak ada (isReassigned=false, originalResource=null) |
| SB-04 | ✅ PASS | Alihkan ke resource yang sama → 400 |
| SB-05 | ✅ PASS | Alihkan kendaraan → ruangan → 400 "resource type mismatch: cannot substitute with a different resource type" |
| SB-06 | ✅ PASS | Alihkan ke kendaraan MAINTENANCE → 409 |
| SB-07 | ✅ PASS | Alihkan ke kendaraan yang punya booking bentrok → 409 |
| SB-08 | ✅ PASS | Alihkan booking APPROVED → 409 |
| SB-09 | ✅ PASS | Alihkan ruangan → ruangan → 200 |
| AS-01 | ❌ FAIL | Supir ditugaskan=true, notif=true, tapi TIDAK tercatat memegang kendaraan (held=null) |
| AS-02 | ✅ PASS | Pindah kendaraan: resource baru + "Dialihkan dari" kendaraan lama |
| AS-02b | ❌ FAIL | Supir kini memegang kendaraan LAMA (tidak ikut pindah) |
| AS-03 | ❌ FAIL | Setelah pindah supir: supir LAMA memegang=226, supir BARU memegang=null |
| AS-04 | ❌ FAIL | Setelah booking selesai, supir LAMA MASIH memegang kendaraan 226 (tertahan) |
| AS-05 | ✅ PASS | Tugaskan pada booking PENDING → 409 |
| AS-06 | ✅ PASS | Tugaskan pada booking ruangan → 400 |
| AS-07 | ✅ PASS | Tugaskan supir nonaktif → 404 "supir aktif tidak ditemukan" |
| AS-08 | ✅ PASS | Tugaskan kendaraan yang bentrok → 409 |
| AS-09 | ✅ PASS | Tugaskan kendaraan yang SPD di hari itu → 409 |
| AS-10 | ✅ PASS | Tugaskan kendaraan yang dijadwalkan maintenance → 409 |
| AS-11 | ✅ PASS | Mulai setelah pindah: baru=IN_USE, lama=AVAILABLE |
| CN-01 | ✅ PASS | CANCELLED, notif admin=true, notif supir=true |
| CN-02 | ✅ PASS | Setelah batal, kendaraan & supir yang sama langsung dipakai & disetujui untuk booking lain |
| CN-03 | ℹ️ INFO | Karyawan membatalkan booking APPROVED → 400 "booking is not in PENDING status" (perlu keputusan) |
| CN-04 | ℹ️ INFO | Admin membatalkan booking APPROVED → 400 "booking is not in PENDING status" (perlu keputusan); supir tetap memegang kendaraan=true |
| CN-05 | ✅ PASS | Batalkan booking ONGOING → 400 |
| CN-06 | ✅ PASS | Karyawan lain membatalkan → 403 |
| CN-07 | ✅ PASS | Admin membatalkan; timeline mencatat pelaku "Admin Uji o21v3" |
| ST-01 | ✅ PASS | ONGOING, kendaraan IN_USE, odometer awal 10050, notif pemohon=true |
| ST-02 | ✅ PASS | Mulai 60 menit sebelum jadwal → 400 "jadwal booking belum dapat dimulai (maksimal 30 menit sebelum jadwal)" |
| ST-03 | ✅ PASS | Mulai setelah jam selesai → 400 "booking period has already ended" |
| ST-04 | ✅ PASS | Odometer awal < tercatat → 400 "odometer tidak boleh kurang dari catatan kendaraan saat ini (10000 km)" |
| ST-05 | ✅ PASS | Supir lain memulai → 403 |
| ST-06 | ✅ PASS | Supir memulai booking ruangan → 400 |
| ST-07 | ℹ️ INFO | Admin memulai booking kendaraan TANPA supir → 200 (UI menyembunyikan tombol; perlu keputusan apakah API juga menolak) |
| ST-08 | ✅ PASS | Karyawan memulai ruangannya sendiri → 200, ruangan IN_USE |
| ST-09 | ✅ PASS | Karyawan memulai booking kendaraannya → 400 |
| ST-10 | ✅ PASS | Penjaga ruangan memulai ruangannya → 200 |
| ST-11 | ℹ️ INFO | Penjaga ruangan LAIN (bukan penjaga ruangan itu) memulai → 200 |
| ST-12 | ✅ PASS | Maintenance dibuat setelah disetujui → mulai 409 "kendaraan ini sedang/akan menjalani maintenance pada tanggal tersebut" |
| CP-01 | ✅ PASS | COMPLETED, kendaraan AVAILABLE, supir dilepas=true, notif selesai=true, ajakan rating=true, overtime=null |
| CP-02 | ✅ PASS | Selesai 90 menit terlambat → overtime 90 menit, notif supir=true |
| CP-03 | ✅ PASS | SPD terlambat → tanpa overtime (null) |
| CP-04 | ✅ PASS | Selesaikan OVERDUE → COMPLETED, overtime 30 menit |
| CP-05 | ✅ PASS | Selesaikan booking APPROVED → 409 |
| CP-06 | ✅ PASS | Supir menyelesaikan → 403 |
| CP-07 | ✅ PASS | Karyawan menyelesaikan ruangannya → 200, ruangan AVAILABLE |
| CP-08 | ✅ PASS | Supir masih punya booking aktif lain → tetap memegang kendaraan |
| CP-10 | ❌ FAIL | Maintenance otomatis masih terbuka, tapi setelah booking selesai status kendaraan jadi AVAILABLE |
| TO-01 | ✅ PASS | Setelah daftar dibuka → IGNORED, timeline aksi sistem=true |
| TO-02 | ✅ PASS | APPROVED tak pernah dimulai → EXPIRED |
| TO-03 | ❌ FAIL | Setelah EXPIRED supir MASIH memegang kendaraan (sebelum=254, sesudah=254) |
| TO-03b | ❌ FAIL | Kendaraan bekas booking EXPIRED dipakai supir lain → approve 409 "kendaraan ini sedang dipegang supir lain - gabungkan (merge) atau alihkan ke kendaraan lain sebelum menyetujui" |
| TO-04 | ✅ PASS | Lewat jam selesai → OVERDUE, kendaraan IN_USE |
| TO-05 | ❌ FAIL | Jam selesai sudah lewat, tapi detail booking masih PENDING sampai ada yang membuka DAFTAR booking (dashboard 200) |
| TO-06 | ✅ PASS | Booking EXPIRED: approve 400, mulai 409, batal 400 |
| MG-01 | ✅ PASS | Gabung: ikut kendaraan & supir utama, jendela 09–13 (true), kendaraan lama bebas=true, notif=true |
| MG-02 | ✅ PASS | Jendela gabungan kustom 08–14 diterapkan |
| MG-03 | ✅ PASS | Gabung dengan diri sendiri → 400 |
| MG-04 | ✅ PASS | Gabung ulang pasangan yang sama → 409 |
| MG-05 | ✅ PASS | Gabung booking ruangan → 400 |
| MG-06 | ✅ PASS | Gabung ke booking ONGOING → 409 |
| MG-07 | ✅ PASS | Jendela gabungan menabrak booking lain di kendaraan yang sama → 409 |
| MG-08 | ✅ PASS | Total penumpang 6 di kapasitas 4 → peringatan "Warning: Vehicle capacity overload! (Remaining capacity is negative). Please substitute vehicle if needed." |
| MG-09 | ❌ FAIL | Gabung + ganti supir: supir utama=DRVMG9N o21v3, kendaraan utama=HILANG (null), supir sekunder=DRVMG9N o21v3 |
| MG-10 | ✅ PASS | Mulai utama → sekunder ONGOING; selesai utama → sekunder COMPLETED; supir dilepas=true |
| MG-11 | ✅ PASS | Rating dari booking sekunder → 400 "booking ini digabung (merge) - beri rating dari booking utama #273" |
| RR-01 | ✅ PASS | Laporan terkirim, odometer kendaraan 10100, notif admin=true |
| RR-02 | ✅ PASS | Laporan saat OVERDUE → 201 |
| RR-03 | ✅ PASS | Laporan kedua → 409 |
| RR-04 | ✅ PASS | Odometer akhir < awal trip → 400 "odometer tidak boleh kurang dari odometer awal trip ini (10050 km)" |
| RR-05 | ✅ PASS | Odometer melewati batas servis → maintenance otomatis 1, status MAINTENANCE |
| RR-06 | ✅ PASS | Admin mengirim laporan → 403 |
| RR-07 | ✅ PASS | Laporan untuk booking ruangan → 400 |
| RR-08 | ✅ PASS | Pemohon melihat laporan: 200, foto 1 |
| RT-01 | ✅ PASS | Rating tersimpan, notif supir=true, rata-rata=5 |
| RT-02 | ✅ PASS | Rating kedua → 409 |
| RT-03 | ✅ PASS | Bukan pemilik → 403; belum selesai → 400 |
| RT-04 | ✅ PASS | Rating supir pada booking tanpa supir → 400 |
| RT-05 | ✅ PASS | Rating ruangan tanpa penjaga → 201 |
| RT-06 | ✅ PASS | Rating ruangan masuk ringkasan penjaga (true) |
| MT-01 | ✅ PASS | Maintenance hari ini → kendaraan MAINTENANCE, booking 409, X-Data-Changed=maintenance |
| MT-02 | ✅ PASS | Maintenance kedua saat masih terbuka → 409 |
| MT-03 | ✅ PASS | Maintenance bentrok dengan booking disetujui → dibuat + peringatan "Warning: There are active bookings that overlap with this maintenance schedule." |
| MT-04 | ❌ FAIL | Maintenance untuk MINGGU DEPAN langsung membuat kendaraan MAINTENANCE; booking BESOK ditolak (409) |
| MT-05 | ✅ PASS | Selesai → AVAILABLE, baseline servis 31000 |
| MT-06 | ✅ PASS | Selesaikan ulang → 400 |
| MT-07 | ❌ FAIL | Edit maintenance jadi "completed" (200) tapi kendaraan tetap MAINTENANCE |
| MT-08 | ❌ FAIL | Maintenance dibuat berstatus "completed" (201) tapi kendaraan MAINTENANCE; diselesaikan lagi → 400 "Maintenance is already completed" (terkunci) |
| MT-09 | ❌ FAIL | Trip berjalan: buat maintenance → MAINTENANCE, hapus maintenance → AVAILABLE (seharusnya IN_USE) |
| MT-10 | ✅ PASS | Hapus maintenance terbuka → AVAILABLE & bisa dibooking |
| MT-11 | ✅ PASS | Isi BBM melewati interval → maintenance otomatis 1, kendaraan MAINTENANCE |
| MT-12 | ✅ PASS | Batas terlewati lagi saat masih terbuka → tetap 1 maintenance |
| MT-13 | ✅ PASS | Maintenance tanpa tanggal selesai memblokir tanggal jauh (409) |
| MT-14 | ✅ PASS | Setelah maintenance selesai, tanggal yang tadinya terblokir bisa dibooking (201) |
| MT-15 | ℹ️ INFO | Maintenance otomatis berstatus "ongoing" — pastikan mobile menampilkannya "Berlangsung" (enum mobile hanya pending/completed) |
| MT-17 | ✅ PASS | Sisa km sampai servis = 8000 |
| VH-01 | ❌ FAIL | Lihat BC-08: kendaraan INACTIVE masih bisa dibooking |
| VH-02 | ❌ FAIL | Kendaraan sedang dipakai trip, admin ubah manual ke AVAILABLE → 200, status kini AVAILABLE |
| VH-04 | ✅ PASS | Odometer diturunkan → 400 "odometer tidak boleh kurang dari catatan saat ini (10000 km)" |
| VH-05 | ✅ PASS | Odometer dinaikkan melewati interval → maintenance otomatis 1 |
| VH-06 | ✅ PASS | Plat duplikat → 409 |
| VH-07 | ❌ FAIL | Hapus kendaraan berriwayat booking → 500 "internal server error" (error server, bukan pesan jelas) |
| VH-10 | ✅ PASS | Kendaraan baru (odometer 15.000) tidak langsung jatuh tempo servis |
| RM-01 | ✅ PASS | Penjaga tampil di ruangan & ruangan tampil di penjaga (true) |
| RM-02 | ✅ PASS | Penjaga nonaktif memulai booking ruangan → 403 |
| RM-03 | ❌ FAIL | Penjaga ruangan mengubah status ruangan → 403 (mobile menampilkan tombolnya untuk ROOM_KEEPER) |
| DU-01 | ✅ PASS | Buat supir tanpa SIM/telepon → 400 "licenseNumber and phoneNumber are required for DRIVER role" |
| DU-02 | ✅ PASS | Supir baru muncul di picker |
| DU-03 | ℹ️ INFO | Supir dinonaktifkan padahal punya booking APPROVED → booking tetap ditugaskan ke supir nonaktif (DRVDU3 o21v3), tanpa peringatan |
| DU-04 | ✅ PASS | Akun supir dinonaktifkan: di picker=false, login=403 |
| DU-05 | ✅ PASS | Diaktifkan kembali → muncul lagi di picker |
| DU-06 | ✅ PASS | Hapus pengguna berriwayat booking → 409 "user ini masih punya riwayat terkait (booking, persetujuan, driver/room keeper, dll) dan tidak bisa dihapus - nonaktifkan saja user ini" |
| DU-07 | ✅ PASS | Ubah ke DRIVER tanpa SIM → 400; dengan SIM → 200, data supir dibuat=true |
| DU-08 | ✅ PASS | Tugaskan manual → memegang 282; lepas → null |
| FL-01 | ✅ PASS | Tercatat, odometer 10100, biaya 100000 (10 L × harga master), X-Data-Changed=fuel |
| FL-02 | ✅ PASS | Odometer sebelum < tercatat → 400 |
| FL-03 | ✅ PASS | Odometer sesudah ≤ sebelum → 400 |
| FL-04 | ℹ️ INFO | Isi "Listrik PLN" untuk kendaraan BBM lewat API → 201 (UI mengunci; API tidak memvalidasi) |
| FL-05 | ✅ PASS | Sama dengan MT-11 |
| FL-06 | ℹ️ INFO | Catatan BBM dihapus → odometer kendaraan tetap 10200 (tidak mundur) |
| SY-HDR | ✅ PASS | Header X-Data-Changed tepat untuk 6 jenis request |
| TZ-03 | ✅ PASS | Booking 00:30 WIB tersimpan 2026-11-21 00:30 WIB, dikembalikan sebagai 2026-11-20T17:30:00Z |
| TZ-04 | ✅ PASS | Waktu notifikasi terbaru selisih 0.0 menit dari sekarang (tidak bergeser 7 jam) |
| TZ-05..07 | ⏭️ SKIP | Batas periode laporan WIB dicakup unit test backend (TestPeriodBoundsUseWIBCalendar); tampilan diuji di browser |
| DL-01 | ❌ FAIL | Beda: available_drivers: dashboard 53 vs nyata 49 |
| DL-05 | ✅ PASS | Audit log aksi manusia: 1669 baris, tanpa IP: 0 |
