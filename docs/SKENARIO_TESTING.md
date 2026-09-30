# Skenario Testing — Reservasi KCE (Web · Mobile · Backend)

> Daftar skenario **alur nyata** dari login sampai maintenance, disusun per status dan
> korelasinya dengan menu lain. Hasil yang diharapkan ditulis menurut **aturan bisnis**;
> bila hasil membaca kode backend (2026-09-30) menunjukkan perilaku saat ini **berbeda**,
> skenario ditandai 🔴 dan dijelaskan di [§ Temuan](#temuan-dari-membaca-kode).
>
> Guest booking **tidak diuji** — fitur sudah tidak dipakai (2026-09-30).
>
> **Maintenance hanya diajukan admin** — tidak ada maintenance otomatis / pengingat km (keputusan 2026-09-30).
>
> Menambah skenario: lanjutkan nomor terakhir di bagiannya (mis. `BC-25`), jangan
> mengubah nomor yang sudah ada — nomor dipakai sebagai rujukan laporan bug.

## Daftar isi
- [Cara pakai & legenda](#cara-pakai--legenda)
- [Data uji](#data-uji)
- [Peta status](#peta-status)
- [Kode menu (kolom "Menu lain ikut berubah")](#kode-menu)
1. [Autentikasi & sesi — AU](#1-autentikasi--sesi--au)
2. [Hak akses per role — RL](#2-hak-akses-per-role--rl)
3. [Buat booking — BC](#3-buat-booking--bc)
4. [Pemilihan supir otomatis & supir tetap — SP](#4-pemilihan-supir-otomatis--supir-tetap--sp)
5. [Persetujuan & penolakan — AP](#5-persetujuan--penolakan--ap)
6. [Alihkan resource (substitute) — SB](#6-alihkan-resource-substitute--sb)
7. [Tugaskan / pindah kendaraan & supir — AS](#7-tugaskan--pindah-kendaraan--supir--as)
8. [Pembatalan — CN](#8-pembatalan--cn)
9. [Mulai perjalanan / pemakaian — ST](#9-mulai-perjalanan--pemakaian--st)
10. [Selesaikan — CP](#10-selesaikan--cp)
11. [Transisi otomatis berbasis waktu — TO](#11-transisi-otomatis-berbasis-waktu--to)
12. [Gabung booking (merge) — MG](#12-gabung-booking-merge--mg)
13. [Laporan pengembalian — RR](#13-laporan-pengembalian--rr)
14. [Rating — RT](#14-rating--rt)
15. [Maintenance — MT](#15-maintenance--mt)
16. [Kendaraan — VH](#16-kendaraan--vh)
17. [Ruangan & penjaga ruangan — RM](#17-ruangan--penjaga-ruangan--rm)
18. [Driver & pengguna — DU](#18-driver--pengguna--du)
19. [BBM / listrik — FL](#19-bbm--listrik--fl)
20. [Sinkronisasi antar menu & perangkat — SY](#20-sinkronisasi-antar-menu--perangkat--sy)
21. [Waktu & zona WIB — TZ](#21-waktu--zona-wib--tz)
22. [Notifikasi — NT](#22-notifikasi--nt)
23. [Dashboard & laporan — DL](#23-dashboard--laporan--dl)
25. [Alur ujung-ke-ujung (E2E) — E2E](#25-alur-ujung-ke-ujung-e2e--e2e)
- [Temuan dari membaca kode](#temuan-dari-membaca-kode)

---

## Cara pakai & legenda

| Tanda | Arti |
|---|---|
| 🔴 | **Dugaan bug** — dari membaca kode, perilaku saat ini kemungkinan BERBEDA dari "Diharapkan". Wajib diuji & dilaporkan. Detail: [Temuan](#temuan-dari-membaca-kode) |
| ❓ | **Perlu keputusan bisnis** — aturan belum jelas; "Diharapkan" berisi usulan |
| 🌐 / 📱 | Khusus web / khusus mobile (tanpa tanda = keduanya) |
| API | Perlu diuji lewat API langsung (Postman), karena UI menyembunyikan tombolnya |

Kolom **Menu lain ikut berubah** = menu yang HARUS menampilkan data terbaru tanpa reload
(di tab/perangkat yang sama maupun lain) — lihat aturan sinkronisasi di [§20](#20-sinkronisasi-antar-menu--perangkat--sy).

Waktu ditulis dalam **WIB**. `H` = hari ini, `H+1` = besok, `J` = jam sekarang.

## Data uji

Siapkan sekali sebelum testing (nama boleh diganti, peran & sifatnya jangan):

| Kode | Data | Sifat penting |
|---|---|---|
| ADM, ADM2 | Admin | ADM2 untuk menguji "admin tidak boleh menolak booking sendiri" |
| EMP-A, EMP-B | Karyawan | Pemohon booking; EMP-B dipakai untuk bentrok/merge |
| DRV-1 | Supir aktif | **Supir tetap K1** |
| DRV-2 | Supir aktif | Tanpa kendaraan tetap |
| DRV-3 | Supir aktif | Tanpa kendaraan tetap, dipakai untuk "pindah supir" |
| DRV-X | Supir **nonaktif** | Untuk uji supir nonaktif |
| RK-1 | Penjaga ruangan aktif | Menjaga R1 |
| RK-2 | Penjaga ruangan aktif | Tidak menjaga ruangan mana pun |
| K1 | Avanza, kapasitas 6, BBM | Supir tetap DRV-1 |
| K2 | Innova, kapasitas 7, BBM | Tanpa supir tetap |
| K3 | Ioniq, kapasitas 4, LISTRIK | Tanpa supir tetap |
| K4 | Kendaraan status **INACTIVE** | |
| R1 | Ruang Melati | Penjaga RK-1 |
| R2 | Ruang Anggrek | **Tanpa** penjaga |

## Peta status

**Booking**
```
PENDING ──approve──► APPROVED ──start──► ONGOING ──complete──► COMPLETED ──► (rating)
   │ ├─reject──► REJECTED            │                 │
   │ ├─cancel──► CANCELLED           │ lewat endDate   │ lewat endDate
   │ └─lewat endDate─► IGNORED       └─► EXPIRED       └─► OVERDUE ──complete──► COMPLETED
   └─substitute / merge tetap di jalur PENDING→APPROVED
```
- Transisi otomatis (IGNORED/EXPIRED/OVERDUE) **baru terjadi saat ada yang membuka daftar booking** (bukan terjadwal) — lihat [TO](#11-transisi-otomatis-berbasis-waktu--to).

**Resource (kendaraan/ruangan)**: `AVAILABLE` · `IN_USE` (booking ONGOING) · `MAINTENANCE` · `INACTIVE`

**Supir**: aktif/nonaktif (menu Driver) + akun aktif/nonaktif (menu Pengguna) + **memegang kendaraan**
(mulai saat booking-nya di-approve, dilepas saat booking selesai). Supir yang memegang kendaraan
tidak dipilih otomatis untuk booking lain.

## Kode menu

| Kode | Menu | Kode | Menu |
|---|---|---|---|
| LB | Daftar booking | KD | Kendaraan (daftar & detail) |
| DB | Detail booking + timeline | RG | Ruangan (daftar & detail) |
| AQ | Antrian approval (admin) | DR | Driver (daftar & detail) |
| DS | Dashboard | PG | Pengguna |
| KL | Kalender ketersediaan | RK | Penjaga ruangan |
| PK | Picker resource & supir di form booking | MT | Maintenance |
| NT | Notifikasi (lonceng + push) | BBM | Catatan BBM/listrik |
| LP | Laporan (semua tab + audit log) | HD | Header (nama/role pengguna) |

---

## 1. Autentikasi & sesi — AU

| ID | Skenario | Diharapkan | Menu lain ikut berubah |
|---|---|---|---|
| AU-01 | Login email & password benar (tiap role) | Masuk ke dashboard sesuai role; menu/tab sesuai role | — |
| AU-02 | Login password salah | Pesan "password salah", tetap di halaman login, pesan tidak hilang karena reload | — |
| AU-03 | Login email tidak terdaftar | Pesan email tidak ditemukan | — |
| AU-04 | Login akun yang dinonaktifkan admin | Ditolak "akun tidak aktif" | — |
| AU-05 | Akun dinonaktifkan admin **saat sedang login** | Paling lambat saat token di-refresh (±15 menit) sesi berakhir & kembali ke login | PG |
| AU-06 | Biarkan aplikasi diam > masa access token lalu klik menu | Token diperbarui diam-diam, request berhasil, tidak ter-logout | — |
| AU-07 | Refresh token kedaluwarsa / dicabut | Ke halaman login, tanpa layar error | — |
| AU-08 | Logout | Token dicabut; tombol back tidak membuka halaman terproteksi | — |
| AU-09 | 📱 Tutup app lalu buka lagi (masih login) | Langsung masuk (auto-login), data termuat | — |
| AU-10 | Lupa password → OTP benar → reset | Password baru bisa dipakai login, yang lama tidak | — |
| AU-11 | OTP salah / OTP kedaluwarsa | Ditolak dengan pesan jelas; tidak bisa reset | — |
| AU-12 | Ganti password dengan password lama salah | Ditolak | — |
| AU-13 | 🌐 Buka URL halaman admin (mis. `/users`) sebagai EMPLOYEE | Dialihkan ke `/unauthorized` | — |
| AU-14 | 🌐 Buka URL terproteksi tanpa login | Ke `/login?redirect=...`, setelah login kembali ke halaman tadi | — |
| AU-15 | Admin mengubah **nama/departemen/role** akun yang sedang login di perangkat lain | Nama/role di header ikut berubah tanpa reload; bila role turun dari ADMIN, menu admin hilang (🌐 sudah; 🔴 📱 profil baru diperbarui saat app dibuka ulang) | HD, PG |
| AU-16 | API: `POST /auth/register` dengan `roleId` admin | **Ditolak** (pendaftaran publik tidak boleh memilih role admin) (✅ B1) | PG |
| AU-17 | Login di 2 perangkat bersamaan | Keduanya jalan; logout di satu tidak mengeluarkan yang lain | — |
| AU-18 | Login akun yang sama dua kali di detik yang sama (klik ganda, web + HP) | Keduanya berhasil (✅ B3) | — |

## 2. Hak akses per role — RL

| ID | Skenario | Diharapkan |
|---|---|---|
| RL-01 | DRIVER / ROOM_KEEPER mencoba membuat booking (UI & API) | Tidak ada tombol; API 403 |
| RL-02 | EMPLOYEE membuka detail booking milik orang lain (URL/ID) | 403 / "tidak ditemukan" |
| RL-03 | DRIVER membuka booking yang tidak ditugaskan kepadanya | 403 |
| RL-04 | EMPLOYEE melihat daftar booking | Hanya booking miliknya |
| RL-05 | DRIVER melihat daftar ("Tugas Saya") | Hanya booking yang ditugaskan kepadanya |
| RL-06 | Kalender ketersediaan resource dibuka EMPLOYEE | Menampilkan jadwal SEMUA pengguna di resource itu (bukan hanya miliknya) |
| RL-07 | API: DRIVER / ROOM_KEEPER membatalkan booking PENDING milik orang lain | **403** — hanya pemilik atau admin (✅ B2) |
| RL-08 | API: EMPLOYEE mencatat BBM | 403 (hanya ADMIN & DRIVER) (✅ B20) |
| RL-09 | EMPLOYEE / DRIVER membuka menu Laporan, Pengguna, Maintenance | Tidak ada menunya; API 403 |
| RL-10 | 📱 Mobile: fitur yang sengaja hanya di web (CRUD kendaraan/ruangan/pengguna, master jenis BBM) | Tidak tersedia di mobile (bukan sekadar disembunyikan) |

## 3. Buat booking — BC

| ID | Skenario | Diharapkan | Menu lain ikut berubah |
|---|---|---|---|
| BC-01 | EMP-A booking K2 H+1 09:00–12:00, NON_SPD, 3 penumpang | PENDING; admin dapat notifikasi "Booking baru" | LB, AQ, DS (pending +1), KL, NT admin |
| BC-02 | EMP-A booking R1 H+1 09:00–10:00 | PENDING | LB, AQ, DS, KL |
| BC-03 | Jam selesai ≤ jam mulai | Ditolak "rentang tanggal tidak valid" | — |
| BC-04 | Tanggal mulai di masa lalu (mis. kemarin) | **Ditolak** (UI & API; toleransi 30 menit untuk jeda mengisi form) (✅ B15) | — |
| BC-05 | Booking kendaraan berstatus MAINTENANCE | Ditolak 409 "sedang maintenance" | — |
| BC-06 | K2 punya jadwal maintenance H+3–H+4; booking K2 H+3 | Ditolak "sedang/akan menjalani maintenance pada tanggal tersebut" | — |
| BC-07 | K2 maintenance H+3–H+4; booking K2 H+5 | Diterima (tidak bentrok tanggal) | KL |
| BC-08 | Booking K4 (INACTIVE) / ruangan INACTIVE | **Ditolak** (juga saat approve); resource INACTIVE tidak muncul di picker (✅ B16) | PK |
| BC-09 | Resource sedang IN_USE sekarang, booking untuk besok | Diterima (IN_USE hanya status saat ini) | — |
| BC-10 | EMP-A & EMP-B booking K2 di jam yang sama (dua-duanya PENDING) | Keduanya diterima PENDING (bentrok diselesaikan admin saat approve: merge/alihkan) | AQ (tanda kandidat merge) |
| BC-11 | EMP-A & EMP-B booking **R1** di jam yang sama | Yang kedua **ditolak** saat approve (ruangan tidak bisa dipakai dua rapat) (✅ B8) | KL |
| BC-12 | Jumlah penumpang > kapasitas kendaraan | Booking masuk; saat approve muncul **peringatan** kapasitas | — |
| BC-13 | Pilih supir yang nonaktif (API) | Ditolak "supir yang dipilih tidak aktif" | — |
| BC-14 | Booking SPD K2 H+2 08:00–10:00 lalu booking lain K2 H+2 15:00–17:00 (K2 sudah APPROVED SPD) | Yang kedua ditolak "kendaraan sedang bertugas SPD pada tanggal tersebut" (SPD memblokir **seharian**) | — |
| BC-15 | K2 punya booking NON_SPD APPROVED H+2 08–10; booking baru **SPD** K2 H+2 15–17 | Ditolak (SPD butuh hari kosong) | — |
| BC-16 | Booking SPD lintas hari H+2 20:00 – H+3 06:00 | Memblokir tanggal H+2 **dan** H+3 penuh (WIB) | KL |
| BC-17 | Pilih supir DRV-2 yang sedang memegang K3 (booking lain APPROVED) untuk booking K2 | Booking memakai kendaraan yang dipegang DRV-2 (K3) — jalur gabung; dicek bentrok saat approve | PK |
| BC-18 | Booking dengan field wajib kosong (tujuan, penumpang) | Validasi form; tidak terkirim | — |
| BC-19 | 📱 Buka form booking dari dashboard, simpan | Kembali ke tab Booking (bukan dashboard), booking baru ada di atas | LB, DS |
| BC-20 | Klik simpan 2x cepat | Hanya 1 booking terbentuk | LB |
| BC-21 | Karyawan booking atas nama sendiri lalu admin melihat | Admin melihat nama & departemen pemohon benar | LB, AQ |

## 4. Pemilihan supir otomatis & supir tetap — SP

| ID | Skenario | Diharapkan | Menu lain ikut berubah |
|---|---|---|---|
| SP-01 | Booking **K1** (supir tetap DRV-1) tanpa memilih supir | Supir otomatis DRV-1 | DB, DR |
| SP-02 | Booking K1 sambil memilih DRV-2 | Tetap DRV-1 (supir tetap menang) — UI sebaiknya mengunci pilihan | DB |
| SP-03 | Booking K2 (tanpa supir tetap) tanpa memilih supir, ada supir kosong | Supir kosong pertama (aktif & tidak memegang kendaraan) otomatis ditempel | DB, DR |
| SP-04 | Booking K2 tanpa memilih supir, **semua supir sedang memegang kendaraan** | Booking tetap PENDING tanpa supir; admin bisa menugaskan setelah approve | AQ |
| SP-05 | Supir kosong satu-satunya dinonaktifkan (menu Driver) lalu booking K2 | Supir nonaktif tidak dipilih otomatis | PK |
| SP-06 | Akun supir dinonaktifkan dari menu **Pengguna** (driver masih aktif) | Tidak dipilih otomatis & tidak muncul di picker | PK, DR |
| SP-07 | DRV-1 (supir tetap K1) sedang SPD H+2; booking K1 H+2 | Ditolak "supir ini sedang bertugas SPD" | — |
| SP-08 | Ubah supir tetap K1 dari DRV-1 ke DRV-2 (dari menu **Kendaraan**) | Menu Driver: DRV-1 tidak lagi punya kendaraan tetap, DRV-2 = K1 | KD, DR |
| SP-09 | Ubah kendaraan tetap DRV-2 ke K3 (dari menu **Driver**) | K1 kehilangan supir tetap, K3 = DRV-2 | KD, DR |
| SP-10 | Hapus supir tetap K1 | Booking K1 berikutnya memakai supir kosong otomatis | KD, DR |
| SP-11 | Picker supir di form booking | Menampilkan supir aktif, sisa kursi kendaraan yang dipegang, tujuan trip yang bentrok, badge "Digunakan SPD" | PK |
| SP-12 | Booking K2 tanpa pilih supir → DRV-2 ditempel; sebelum approve DRV-2 di-approve di booking lain yang jamnya bentrok | Saat approve: ditolak & admin menugaskan supir lain (✅ B7) | AQ |
| SP-13 | ❓ Ada supir bebas & supir tetap kendaraan lain; booking kendaraan tanpa supir tetap | Usulan: supir bebas diutamakan; saat ini bisa mengambil supir tetap kendaraan lain | PK |

## 5. Persetujuan & penolakan — AP

| ID | Skenario | Diharapkan | Menu lain ikut berubah |
|---|---|---|---|
| AP-01 | Admin approve booking PENDING K2 (supir DRV-2) | APPROVED; DRV-2 **memegang** K2; pemohon & supir dapat notifikasi | LB, DB, AQ, DS, DR (DRV-2 sibuk), KL, NT, LP |
| AP-02 | Approve booking yang sudah APPROVED/REJECTED/CANCELLED | Ditolak "booking tidak PENDING" | — |
| AP-03 | Approve padahal K2 sudah dipakai booking APPROVED lain yang bentrok jam | 409 "gabungkan (merge) atau alihkan" | — |
| AP-04 | Approve booking K2 (supir DRV-3) padahal K2 sedang dipegang DRV-2 | 409 "kendaraan sedang dipegang supir lain" | — |
| AP-05 | Maintenance K2 dibuat SETELAH booking masuk PENDING, lalu approve | 409 "akan menjalani maintenance" | — |
| AP-06 | Approve dengan penumpang total > kapasitas | Berhasil + **peringatan** kapasitas tampil (tidak hilang diam-diam) | — |
| AP-07 | Approve booking ruangan R1 | APPROVED; penjaga ruangan dapat notifikasi "Ruangan dipesan" | NT RK-1 |
| AP-08 | Reject tanpa catatan | Ditolak validasi (catatan wajib) | — |
| AP-09 | Reject dengan catatan | REJECTED; pemohon lihat alasan di detail; notifikasi + email | LB, DB, AQ, DS, KL, NT |
| AP-10 | ADM membuat booking sendiri lalu **menolaknya** sendiri | Ditolak "tidak boleh menyetujui/menolak booking sendiri" | — |
| AP-11 | ADM membuat booking sendiri lalu **menyetujuinya** sendiri | Konsisten dengan AP-10 → **ditolak** (403); web & mobile menyembunyikan tombolnya dan menampilkan "diputuskan admin lain" (✅ B21) | — |
| AP-12 | Dua admin approve booking yang sama hampir bersamaan | Yang kedua ditolak "tidak PENDING"; tidak ada data ganda | AQ |
| AP-13 | Admin approve dari **mobile** saat admin lain membuka AQ di **web** | Baris hilang dari AQ web tanpa reload | AQ, DS |
| AP-14 | Booking PENDING tanpa supir (SP-04) di-approve | APPROVED tanpa supir; tombol "Mulai" tidak muncul sampai ditugaskan supir+kendaraan | DB |
| AP-15 | Riwayat persetujuan setelah approve | Tercatat (siapa, kapan, catatan); riwayat lama diisi ulang migrasi 000015 (✅ B13) | DB, LP |

## 6. Alihkan resource (substitute) — SB

| ID | Skenario | Diharapkan | Menu lain ikut berubah |
|---|---|---|---|
| SB-01 | Alihkan booking PENDING K2 → K3, lalu approve (2 langkah otomatis) | APPROVED di **K3**; detail menampilkan "Dialihkan dari K2"; pemohon dapat notifikasi | LB, DB, KL (K2 kosong, K3 terisi), NT |
| SB-02 | Setelah SB-01, cek kendaraan yang ditugaskan & kendaraan yang dipegang supir | Keduanya **K3** (✅ B6) | DR, KD |
| SB-03 | Setelah SB-01, tanda "Dialihkan" di daftar & detail | Tampil (✅ B6) | LB, DB |
| SB-04 | Alihkan ke resource yang sama | Ditolak | — |
| SB-05 | Alihkan kendaraan → ruangan | Ditolak "tipe resource berbeda" | — |
| SB-06 | Alihkan ke kendaraan MAINTENANCE | Ditolak | — |
| SB-07 | Alihkan ke kendaraan yang punya booking PENDING/APPROVED/ONGOING bentrok | Ditolak "bentrok jadwal" | — |
| SB-08 | Alihkan booking yang sudah APPROVED | Ditolak (hanya PENDING); gunakan "tugaskan kendaraan" (AS) | — |
| SB-09 | Alihkan ruangan R1 → R2 | Berhasil; R1 kosong di kalender | KL, RG |

## 7. Tugaskan / pindah kendaraan & supir — AS

| ID | Skenario | Diharapkan | Menu lain ikut berubah |
|---|---|---|---|
| AS-01 | Booking APPROVED tanpa supir → tugaskan DRV-3 + K2 | Supir & kendaraan tercatat; DRV-3 memegang K2; supir dapat notifikasi | DB, DR, KD, NT |
| AS-02 | Booking APPROVED K2 (DRV-2) → **pindah kendaraan** ke K3 | Resource jadi K3, tanda "Dialihkan dari K2"; K2 bebas di kalender | LB, DB, KL, KD |
| AS-03 | Booking APPROVED K2 (DRV-2) → **pindah supir** ke DRV-3 | DRV-3 memegang K2 & **DRV-2 kembali kosong** (✅ B5) | DR, PK |
| AS-04 | Lanjutan AS-03: booking diselesaikan | DRV-3 dilepas; DRV-2 **tidak** tertahan (✅ B5) | DR, PK |
| AS-05 | Tugaskan ke booking PENDING / ONGOING | Ditolak "hanya booking APPROVED" | — |
| AS-06 | Tugaskan pada booking ruangan | Ditolak | — |
| AS-07 | Tugaskan supir nonaktif | Ditolak "supir aktif tidak ditemukan" | — |
| AS-08 | Tugaskan kendaraan yang sudah APPROVED/ONGOING di jam bentrok | Ditolak 409 | — |
| AS-09 | Tugaskan kendaraan / supir yang SPD di hari itu | Ditolak 409 | — |
| AS-10 | Tugaskan kendaraan yang dijadwalkan maintenance di rentang booking | Ditolak 409 | — |
| AS-11 | Setelah pindah kendaraan, supir memulai perjalanan | Status IN_USE di kendaraan **baru**; kendaraan lama tetap AVAILABLE | KD, DS |

## 8. Pembatalan — CN

| ID | Skenario | Diharapkan | Menu lain ikut berubah |
|---|---|---|---|
| CN-01 | EMP-A membatalkan booking PENDING miliknya | CANCELLED; admin & supir (bila ada) dapat notifikasi; jadwal bebas di kalender | LB, DB, AQ, DS, KL, NT |
| CN-02 | Booking PENDING K1 dengan supir DRV-1 dibatalkan → cek K1 & DRV-1 | K1 tetap AVAILABLE & DRV-1 tetap kosong (PENDING belum memegang apa pun); keduanya bisa langsung dipakai booking lain | KD, DR, PK |
| CN-03 | ❓ EMP-A membatalkan booking **APPROVED** (rencana berubah) | Usulan: **boleh** sebelum mulai, dan supir + kendaraan dilepas kembali. Saat ini ditolak "booking tidak PENDING" untuk semua role | — |
| CN-04 | ❓ Admin membatalkan booking APPROVED (mis. kendaraan rusak) | Usulan: boleh, dengan alasan; supir & kendaraan dilepas. Saat ini tidak ada jalannya | — |
| CN-05 | Batalkan booking ONGOING / COMPLETED | Ditolak | — |
| CN-06 | EMP-B membatalkan booking EMP-A (API) | 403 | — |
| CN-07 | Admin membatalkan booking PENDING milik karyawan | CANCELLED; tercatat pelakunya admin di timeline | DB |
| CN-08 | Booking dialihkan admin (SB-01) sebelum approve, karyawan tidak setuju → batalkan dari banner "Resource dialihkan" | CANCELLED | LB, AQ |
| CN-09 | Batalkan booking yang sedang digabung (merge) sebagai sekunder | ❓ Usulan: hanya booking itu yang batal, booking utama tetap; penumpang berkurang | DB utama |

## 9. Mulai perjalanan / pemakaian — ST

| ID | Skenario | Diharapkan | Menu lain ikut berubah |
|---|---|---|---|
| ST-01 | DRV-2 memulai booking APPROVED K2 miliknya, 10 menit sebelum jadwal, isi odometer ≥ catatan + foto + lokasi | ONGOING; K2 → **IN_USE**; pemohon dapat notifikasi | LB, DB, DS, KD, DR, KL, NT |
| ST-02 | Mulai > 30 menit sebelum jadwal | Ditolak "belum dapat dimulai (maks 30 menit sebelum)" | — |
| ST-03 | Mulai setelah jam selesai lewat | Ditolak "periode sudah berakhir" | — |
| ST-04 | Odometer awal < odometer kendaraan tercatat | Ditolak, pesan menyebut angka tercatat | — |
| ST-05 | Supir yang tidak ditugaskan mencoba mulai (API) | 403 | — |
| ST-06 | Supir mencoba mulai booking ruangan | Ditolak | — |
| ST-07 | Booking kendaraan APPROVED tanpa supir → tombol mulai | Tidak muncul di UI; ❓ API admin saat ini tetap bisa memulai | DB |
| ST-08 | EMP-A memulai booking **ruangan** miliknya (self-service) | ONGOING; R1 → IN_USE | RG, DS |
| ST-09 | EMP-A memulai booking kendaraan miliknya | Ditolak (kendaraan dimulai supir/admin) | — |
| ST-10 | RK-1 memulai booking R1 | Berhasil | RG |
| ST-11 | ❓ RK-2 (tidak menjaga R1) memulai booking R1 | Usulan: ditolak (bukan ruangannya). Saat ini semua penjaga aktif boleh | — |
| ST-12 | Maintenance K2 dijadwalkan setelah booking di-approve, lalu mulai | Ditolak 409 → admin harus pindah kendaraan (AS-02) | — |
| ST-13 | Mulai booking utama yang punya booking gabungan | Booking gabungan ikut ONGOING otomatis, pemiliknya dapat notifikasi | LB, DB sekunder |
| ST-14 | 📱 Mulai perjalanan tanpa izin lokasi / kamera | Pesan jelas; tidak crash | — |

## 10. Selesaikan — CP

| ID | Skenario | Diharapkan | Menu lain ikut berubah |
|---|---|---|---|
| CP-01 | Admin menyelesaikan booking ONGOING K2 (NON_SPD) tepat waktu | COMPLETED; K2 → AVAILABLE; DRV-2 dilepas (kosong lagi); pemohon dapat notifikasi + ajakan rating | LB, DB, DS, KD, DR, PK, NT, LP |
| CP-02 | Selesaikan NON_SPD 1 jam 30 menit setelah jadwal selesai | Overtime supir 90 menit tercatat; supir & admin dapat notifikasi | DB (overtime), LP (trip driver) |
| CP-03 | Selesaikan SPD terlambat | Tidak ada overtime | LP |
| CP-04 | Selesaikan booking OVERDUE | COMPLETED + overtime (bila NON_SPD) | DS (overdue −1) |
| CP-05 | Selesaikan booking PENDING / APPROVED | Ditolak | — |
| CP-06 | DRIVER mencoba menyelesaikan (UI & API) | Tidak ada tombol; API 403 (supir mengirim laporan pengembalian, admin yang menutup) | — |
| CP-07 | EMP-A menyelesaikan booking ruangan miliknya | COMPLETED; R1 AVAILABLE; langsung muncul form rating ruangan | RG, DS |
| CP-08 | Supir DRV-2 punya 2 booking APPROVED/ONGOING, satu diselesaikan | DRV-2 **tetap** memegang kendaraan (masih ada booking aktif) | DR |
| CP-09 | Selesaikan booking utama hasil merge | Booking gabungan ikut COMPLETED; supir baru dilepas setelah semuanya selesai | LB, DB, DR |
| CP-10 | Admin membuat maintenance K2 saat trip K2 sedang berjalan, lalu booking diselesaikan | K2 tetap IN_USE selama trip, lalu **MAINTENANCE** setelah selesai (✅ B10) | KD, DS, MT |

## 11. Transisi otomatis berbasis waktu — TO

| ID | Skenario | Diharapkan | Menu lain ikut berubah |
|---|---|---|---|
| TO-01 | Booking PENDING dibiarkan sampai jam selesai lewat | IGNORED + tercatat di timeline sebagai aksi sistem | LB, DB, AQ, DS |
| TO-02 | Booking APPROVED tidak pernah dimulai sampai jam selesai lewat | EXPIRED | LB, DB, DS, KL |
| TO-03 | Lanjutan TO-02: cek supir & kendaraan | Supir **dilepas** (kosong lagi), kendaraan bisa dipegang supir lain (✅ B4) | DR, PK |
| TO-04 | Booking ONGOING melewati jam selesai | OVERDUE; kendaraan tetap IN_USE; masih bisa diselesaikan & laporan pengembalian | LB, DS (overdue) |
| TO-05 | Tidak ada yang membuka daftar booking (mis. tengah malam), buka **dashboard** / **kalender** / **kendaraan** | Status sudah berubah: transisi dijalankan server tiap menit & saat detail booking / dashboard dibuka, lalu disiarkan (DATA_CHANGED) ke semua perangkat (✅ B17) | DS, KL, KD |
| TO-06 | Booking EXPIRED/IGNORED | Tidak bisa di-approve, dimulai, atau dibatalkan | — |

## 12. Gabung booking (merge) — MG

| ID | Skenario | Diharapkan | Menu lain ikut berubah |
|---|---|---|---|
| MG-01 | EMP-A (APPROVED, K2, DRV-2, 09–12, 3 org) & EMP-B (PENDING, K3, 10–13, 2 org) searah → admin gabung EMP-B ke EMP-A | EMP-B ikut K2 & DRV-2, jendela 09–13 untuk keduanya, EMP-B otomatis APPROVED ("telah dilakukan merge"); K3 bebas | LB, DB (kartu "Digabungkan"), AQ, KL, DR, NT (EMP-B & DRV-2) |
| MG-02 | Gabung dengan jendela waktu kustom | Kedua booking memakai jendela kustom | KL |
| MG-03 | Gabung booking dengan dirinya sendiri | Ditolak | — |
| MG-04 | Gabung pasangan yang sudah digabung | Ditolak "sudah digabung" | — |
| MG-05 | Gabung booking ruangan | Ditolak (hanya kendaraan) | — |
| MG-06 | Gabung booking ONGOING/COMPLETED | Ditolak | — |
| MG-07 | Jendela gabungan bentrok dengan booking lain di kendaraan yang sama | Ditolak 409 | — |
| MG-08 | Total penumpang > kapasitas setelah gabung | Peringatan kapasitas | — |
| MG-09 | Gabung sambil memilih supir lain | Supir berganti, kendaraan tetap, sekunder mewarisi; supir baru ditolak bila bentrok jadwal / nonaktif (✅ B9) | DB, DR |
| MG-10 | Mulai / selesaikan booking utama | Sekunder ikut ONGOING / COMPLETED (lihat ST-13, CP-09) | LB |
| MG-11 | Rating dari booking sekunder | Ditolak "beri rating dari booking utama #..." | — |
| MG-12 | BBM dicatat di booking utama | Tampil juga di detail booking sekunder, tanpa dobel | DB |
| MG-13 | Kalender kendaraan setelah gabung | Satu trip, bukan dua jadwal terpisah | KL |
| MG-14 | Antrian approval: booking yang punya kandidat gabung | Ada tanda/badge kandidat merge | AQ |

## 13. Laporan pengembalian — RR

| ID | Skenario | Diharapkan | Menu lain ikut berubah |
|---|---|---|---|
| RR-01 | DRV-2 mengirim laporan (catatan, lokasi GPS, foto, odometer akhir) saat ONGOING | Tersimpan; admin dapat notifikasi; odometer K2 maju | DB, KD (odometer), NT admin |
| RR-02 | Kirim saat OVERDUE | Diterima | DB |
| RR-03 | Kirim kedua kalinya | Ditolak "sudah dikirim" | — |
| RR-04 | Odometer akhir < odometer awal trip | Ditolak, menyebut angka awal | — |
| RR-05 | Odometer akhir jauh melewati 10.000 km sejak servis terakhir | **Tidak ada** maintenance otomatis; K2 tetap sesuai statusnya (maintenance hanya diajukan admin) | MT, KD |
| RR-06 | Admin / pemohon mengirim laporan | Tidak ada tombol; API 403 (hanya supir) | — |
| RR-07 | Laporan untuk booking ruangan | Ditolak | — |
| RR-08 | Pemohon & admin membuka laporan | Foto & alamat lokasi tampil | DB |

## 14. Rating — RT

| ID | Skenario | Diharapkan | Menu lain ikut berubah |
|---|---|---|---|
| RT-01 | EMP-A memberi rating supir 5★ + ulasan pada booking COMPLETED miliknya | Tersimpan; supir dapat notifikasi; rata-rata supir naik | DB, DR (rating), LP (driver) |
| RT-02 | Rating kedua kali | Ditolak "sudah dinilai"; panel menampilkan hasil, bukan form kosong | DB |
| RT-03 | Rating booking belum COMPLETED / bukan pemilik | Ditolak | — |
| RT-04 | Rating supir pada booking tanpa supir | Ditolak | — |
| RT-05 | Rating ruangan R2 (tanpa penjaga) | Tersimpan, tidak masuk ringkasan penjaga mana pun | RG |
| RT-06 | Rating ruangan R1 | Masuk ringkasan RK-1 | RK |
| RT-07 | Pengingat rating: booking kendaraan selesai, pemohon login ulang | Prompt/modal rating muncul sekali; bisa ditutup; notifikasi tetap ada | NT |

## 15. Maintenance — MT

| ID | Skenario | Diharapkan | Menu lain ikut berubah |
|---|---|---|---|
| MT-01 | Admin membuat maintenance K2 mulai hari ini | K2 → MAINTENANCE; tidak bisa dibooking | MT, KD, DS, PK |
| MT-02 | K2 sudah punya maintenance terbuka, buat lagi | Ditolak "masih dalam maintenance yang belum selesai" | — |
| MT-03 | Buat maintenance K2 yang bentrok dengan booking APPROVED | Berhasil + **peringatan** ada booking bentrok; booking itu tidak bisa dimulai (ST-12) → admin pindah kendaraan | MT, DB |
| MT-04 | Buat maintenance K2 untuk **minggu depan** | K2 tetap AVAILABLE sampai tanggal mulai; hanya tanggal maintenance yang terblokir (✅ B11) | KD, PK |
| MT-05 | Selesaikan maintenance (foto bukti) | Status selesai; K2 → AVAILABLE | MT, KD, DS, PK |
| MT-06 | Selesaikan maintenance yang sudah selesai | Ditolak | — |
| MT-07 | Edit maintenance, ubah status jadi "selesai" tanpa tanggal selesai | K2 → AVAILABLE (✅ B12) | KD |
| MT-08 | Buat maintenance langsung berstatus "selesai" | K2 tidak terkunci (✅ B12) | KD |
| MT-09 | Hapus maintenance saat K2 sedang IN_USE (trip berjalan) | K2 tetap IN_USE (✅ B10) | KD, DS |
| MT-10 | Hapus maintenance terbuka (K2 tidak dipakai) | K2 → AVAILABLE, bisa dibooking | MT, KD, PK |
| MT-11 | Isi BBM dengan odometer jauh melewati 10.000 km sejak servis terakhir | **Tidak ada** maintenance otomatis; K2 tetap AVAILABLE | MT, KD, BBM |
| MT-12 | _(dihapus 2026-09-30 — maintenance otomatis ditiadakan)_ | — | — |
| MT-13 | Maintenance tanpa tanggal selesai | Memblokir semua tanggal setelah tanggal mulai sampai diselesaikan | KL |
| MT-14 | Setelah maintenance selesai, booking K2 untuk tanggal yang tadinya diblokir | Diterima | PK |
| MT-15 | 📱 Record maintenance lama hasil sistem (status `ongoing`) tampil di mobile | Tampil "Berlangsung", bisa diselesaikan admin, tidak error | MT |
| MT-16 | Karyawan/supir membuka menu maintenance | Tidak ada menu; API 403 | — |
| MT-17 | _(dihapus 2026-09-30 — pengingat sisa km servis ditiadakan)_ | — | — |
| MT-18 | Maintenance K2 terjadwal (K2 masih AVAILABLE), lalu tanggal mulainya tiba | K2 otomatis MAINTENANCE saat daftar/detail kendaraan atau daftar booking dibuka (tanpa penjadwal); bila K2 sedang dipakai trip, tetap IN_USE dan jadi MAINTENANCE setelah trip selesai | KD, DS, PK |

## 16. Kendaraan — VH

| ID | Skenario | Diharapkan | Menu lain ikut berubah |
|---|---|---|---|
| VH-01 | Admin ubah status K2 AVAILABLE → INACTIVE | Tidak muncul di picker; booking baru ditolak (BC-08, ✅ B16) | KD, PK, DS |
| VH-02 | Admin ubah status manual saat K2 sedang dipakai trip (IN_USE) | Ditolak 409 — selesaikan booking dulu (✅ B10) | KD, DS |
| VH-03 | 📱 Ubah status dari mobile | Berhasil; web ikut berubah tanpa reload | KD (web), DS |
| VH-04 | Odometer diturunkan saat edit | Ditolak, menyebut angka tercatat | — |
| VH-05 | Odometer dinaikkan jauh (edit kendaraan) | **Tidak ada** maintenance otomatis | MT, KD |
| VH-06 | Plat nomor duplikat | Ditolak | — |
| VH-07 | Hapus kendaraan yang punya riwayat booking | Ditolak 409 dengan pesan jelas — sarankan INACTIVE (juga ruangan & kategori) (✅ B18) | — |
| VH-08 | Ganti foto kendaraan | Foto baru tampil di daftar, detail, picker, dan kartu booking | KD, PK, LB |
| VH-09 | Ubah nama/kapasitas kendaraan | Nama baru tampil di booking terkait; sisa kursi di picker ikut berubah | LB, DB, PK |
| VH-10 | _(dihapus 2026-09-30 — tidak ada lagi baseline/interval servis)_ | — | — |
| VH-11 | Admin ubah status manual jadi AVAILABLE saat maintenance K2 masih berlangsung | Ditolak 409 — selesaikan maintenance dulu; ubah ke INACTIVE tetap boleh | KD, MT |

## 17. Ruangan & penjaga ruangan — RM

| ID | Skenario | Diharapkan | Menu lain ikut berubah |
|---|---|---|---|
| RM-01 | Tetapkan RK-1 sebagai penjaga R2 | R2 menampilkan RK-1; daftar penjaga menampilkan ruangannya | RG, RK |
| RM-02 | Nonaktifkan RK-1 | Tidak bisa memulai/menyelesaikan booking ruangan; ❓ R1 tetap menampilkan penjaga nonaktif? | RK, RG |
| RM-03 | 📱 Penjaga ruangan ubah status ruangan | Berhasil untuk ruangan yang dijaganya; ruangan lain → 403 "Anda bukan penjaga ruangan ini" (✅ B19) | RG |
| RM-04 | Booking ruangan disetujui | Penjaga ruangan dapat notifikasi | NT |
| RM-05 | ✅ B8 — Dua booking ruangan bentrok jam (lihat BC-11) | Yang kedua tidak bisa disetujui | KL |

## 18. Driver & pengguna — DU

| ID | Skenario | Diharapkan | Menu lain ikut berubah |
|---|---|---|---|
| DU-01 | Buat pengguna role DRIVER tanpa no. SIM / telepon | Ditolak | — |
| DU-02 | Buat pengguna DRIVER lengkap | Muncul di menu Driver & picker supir | PG, DR, PK |
| DU-03 | Nonaktifkan supir (menu Driver) yang punya booking APPROVED | ❓ Usulan: peringatan ada booking aktif; supir tidak muncul di picker | DR, PK |
| DU-04 | Nonaktifkan **akun** supir (menu Pengguna) | Tidak muncul di picker & tidak dipilih otomatis; tidak bisa login | PG, DR, PK |
| DU-05 | Aktifkan kembali | Muncul lagi di picker | DR, PK |
| DU-06 | Hapus pengguna yang punya riwayat booking | Ditolak "nonaktifkan saja" | — |
| DU-07 | Ubah role pengguna EMPLOYEE → DRIVER | Wajib SIM & telepon; muncul di menu Driver | PG, DR |
| DU-08 | Tugaskan / lepas kendaraan supir secara manual (menu Driver) | Status "memegang kendaraan" berubah; ikut memengaruhi pemilihan otomatis | DR, PK |
| DU-09 | Riwayat penugasan supir | Menampilkan kendaraan & periode yang benar | DR |
| DU-10 | Import pengguna massal (Excel) dengan baris salah | Baris valid masuk, baris salah dilaporkan | PG |
| DU-11 | Ganti foto profil sendiri | Tampil di header, daftar pengguna, dan menu Driver bila supir | HD, PG, DR |
| DU-12 | Tambah/ubah/hapus departemen | Ringkasan & filter pengguna ikut berubah; hapus departemen yang masih dipakai ditolak | PG |

## 19. BBM / listrik — FL

| ID | Skenario | Diharapkan | Menu lain ikut berubah |
|---|---|---|---|
| FL-01 | DRV-2 mencatat BBM K2 (odometer sebelum ≥ tercatat, sesudah > sebelum, liter, foto) | Tersimpan; odometer K2 maju; biaya dihitung dari harga master bila harga kosong | BBM, KD, DB (bila terkait booking), LP (biaya) |
| FL-02 | Odometer sebelum < tercatat | Ditolak, menyebut angka tercatat | — |
| FL-03 | Odometer sesudah ≤ sebelum | Ditolak | — |
| FL-04 | Isi "listrik" untuk kendaraan BBM | UI mengunci jenis sesuai energi kendaraan; ❓ API saat ini menerima | — |
| FL-05 | Pengisian dengan odometer jauh melewati 10.000 km | Lihat MT-11: tidak ada maintenance otomatis | MT, KD |
| FL-06 | ❓ Admin menghapus catatan BBM | Terhapus; odometer kendaraan **tidak** mundur (usulan: tampilkan peringatan) | BBM, LP |
| FL-07 | 📱 Mencatat BBM hanya lewat kamera | Galeri tidak bisa dipakai sebagai bukti | — |
| FL-08 | Ubah harga master jenis BBM (web) | Pengisian berikutnya memakai harga baru; yang lama tidak berubah | BBM |
| FL-09 | Catatan BBM di booking gabungan | Lihat MG-12 | DB |

## 20. Sinkronisasi antar menu & perangkat — SY

> Aturan: setiap perubahan data terlihat di SEMUA menu yang memakainya, di tab/perangkat
> yang sama maupun lain, **tanpa reload** dan **tanpa polling** (fetch hanya saat ada
> perubahan). Cara cek: buka DevTools → Network, pastikan pindah menu tanpa perubahan
> tidak memicu request.

| ID | Skenario | Diharapkan |
|---|---|---|
| SY-01 | 🌐 Dua browser (ADM & ADM2). ADM ubah status K2 | Daftar & detail K2 di browser ADM2 berubah ±1 detik |
| SY-02 | 🌐 ADM approve booking, lalu buka Kendaraan, Driver, Dashboard, Laporan di tab yang sama | Semua sudah terbaru tanpa refresh |
| SY-03 | 📱↔🌐 Supir memulai perjalanan di HP; admin membuka web | Status booking ONGOING, K2 IN_USE, dashboard berubah tanpa reload |
| SY-04 | 🌐→📱 Admin menolak booking di web; pemohon membuka tab Booking di HP | Status REJECTED tampil |
| SY-05 | Pindah-pindah menu tanpa ada perubahan data | **Tidak ada** request baru |
| SY-06 | 📱 Tab tersembunyi (mis. Kendaraan) saat data berubah | Dimuat ulang **sekali** saat tab dibuka, bukan saat tersembunyi |
| SY-07 | 📱 App di background saat data berubah, lalu dibuka | Layar yang terlihat memuat ulang |
| SY-08 | Koneksi internet putus 1 menit, data berubah, koneksi pulih | Data terbaru muncul otomatis setelah tersambung |
| SY-09 | Aksi sendiri (approve, ubah status, dll.) | Data diperbarui **sekali** (tidak fetch dobel) |
| SY-10 | 🌐 Admin mengubah nama akunnya sendiri dari browser lain | Nama di header berubah (AU-15) |
| SY-11 | Detail booking terbuka, lalu dari layar lain status kendaraannya berubah; kembali ke detail | Detail sudah terbaru |
| SY-12 | Laporan terbuka di tab lain saat banyak perubahan | Laporan dimuat ulang saat dibuka, tidak terus-menerus |
| SY-13 | Notifikasi masuk (lonceng) | Badge bertambah; daftar notifikasi terbaru; data booking terkait ikut terbaru |

## 21. Waktu & zona WIB — TZ

| ID | Skenario | Diharapkan |
|---|---|---|
| TZ-01 | Buat booking 09:00 di web, buka di HP | Tampil 09:00 di keduanya |
| TZ-02 | Ubah zona waktu komputer/HP ke UTC atau WITA, buka booking yang sama | Tetap tampil 09:00 WIB |
| TZ-03 | Booking pukul 00:30–06:59 WIB (tanggal UTC berbeda) | Tanggal tampil & tersimpan benar, tidak mundur sehari |
| TZ-04 | Waktu notifikasi | Sesuai jam kejadian (tidak 7 jam lebih maju) |
| TZ-05 | Filter laporan "Hari Ini" | 00:00–23:59 WIB; sama persis di web & mobile |
| TZ-06 | Filter laporan rentang tanggal: tanggal terakhir | Ikut terhitung sampai 23:59:59 WIB |
| TZ-07 | Laporan bulanan dibuka 1 Okt pukul 03:00 WIB | Sudah menghitung bulan Oktober |
| TZ-08 | SPD lintas tengah malam (BC-16) | Blokir hari menurut kalender WIB |
| TZ-09 | Ekspor Excel laporan | Jam di Excel = jam WIB |
| TZ-10 | Overtime tercatat (CP-02) | Jam jadwal & aktual tampil WIB |

## 22. Notifikasi — NT

| ID | Pemicu | Penerima | Diharapkan |
|---|---|---|---|
| NT-01 | Booking dibuat | Semua admin | "Booking baru" |
| NT-02 | Approve | Pemohon + supir | "Booking disetujui" / "Booking aktif baru" |
| NT-03 | Approve ruangan | Penjaga ruangan | "Ruangan dipesan" |
| NT-04 | Reject | Pemohon (+ email) | "Booking ditolak" + alasan |
| NT-05 | Cancel | Admin + supir | "Booking dibatalkan" |
| NT-06 | Tugaskan kendaraan | Supir | "Penugasan kendaraan" |
| NT-07 | Start / Complete | Pemohon (+ pemilik booking gabungan) | "Perjalanan dimulai" / "Booking selesai" |
| NT-08 | Complete kendaraan | Pemohon | Ajakan rating supir |
| NT-09 | Overtime | Supir + admin | Menit overtime |
| NT-10 | Rating supir | Supir | Jumlah bintang |
| NT-11 | Substitute / merge | Pemohon / supir | Sesuai aksi |
| NT-12 | Laporan pengembalian | Admin | "Laporan pengembalian" |
| NT-13 | 📱 App ditutup total saat notifikasi datang | Push muncul (FCM); tap → membuka detail booking |
| NT-14 | 📱 App terbuka | Notifikasi sistem muncul sekali (tidak dobel WebSocket + FCM) |
| NT-15 | Tandai dibaca / tandai semua dibaca | Badge berkurang di web & mobile |

## 23. Dashboard & laporan — DL

| ID | Skenario | Diharapkan |
|---|---|---|
| DL-01 | Angka dashboard admin (kendaraan, ruangan, supir tersedia) | Sama dengan jumlah nyata di menu masing-masing; "supir tersedia" = definisi picker (akun aktif & tidak memegang kendaraan) (✅ B14) |
| DL-02 | Setiap aksi di §3–§15 | Angka dashboard terkait langsung berubah |
| DL-03 | Dashboard karyawan | Hanya data miliknya |
| DL-04 | Laporan tab Booking/Resource/Keuangan/Driver/Audit dengan filter periode | Angka konsisten antar tab & dengan data mentah; tidak ada angka > 100% |
| DL-05 | Audit log setiap aksi (buat, approve, ubah status, dll.) | Tercatat dengan pelaku, IP, dan perangkat; aksi sistem (EXPIRED dll.) tanpa pelaku |
| DL-06 | Ekspor Excel | Semua sheet terisi; grafik tampil; jam WIB |
| DL-07 | 📱 Ekspor CSV laporan | File bisa dibagikan & dibuka |
| DL-08 | Laporan biaya setelah BBM & maintenance baru | Total & tren ikut naik |

## 25. Alur ujung-ke-ujung (E2E) — E2E

Rangkaian yang menggabungkan banyak skenario di atas — jalankan berurutan, cek setiap menu.

| ID | Cerita | Rangkaian |
|---|---|---|
| E2E-01 | **Perjalanan dinas normal** | EMP-A booking K2 NON_SPD besok (BC-01) → supir kosong otomatis (SP-03) → ADM approve (AP-01) → DRV-2 mulai (ST-01) → isi BBM di jalan (FL-01) → laporan pengembalian (RR-01) → ADM selesaikan (CP-01) → EMP-A rating (RT-01) → cek laporan biaya & trip supir (DL-08) |
| E2E-02 | **Kendaraan rusak sebelum berangkat** | Booking K2 APPROVED → ADM buat maintenance K2 hari itu (MT-03, ada peringatan) → supir gagal mulai (ST-12) → ADM pindah ke K3 (AS-02) → supir mulai dengan K3 (AS-11) → K2 tetap MAINTENANCE, K3 IN_USE |
| E2E-03 | **Pindah supir mendadak** | Booking K2 APPROVED dengan DRV-2 → DRV-2 sakit → ADM pindah ke DRV-3 (AS-03) → DRV-3 mulai & selesai → cek DRV-2 kembali kosong & bisa dipilih otomatis (AS-04) |
| E2E-04 | **Pilih kendaraan & supir lalu batal** | EMP-A booking K1 (DRV-1) → cancel saat PENDING (CN-01) → K1 & DRV-1 langsung dipakai EMP-B (CN-02) · ulangi setelah APPROVED (CN-03 ❓) |
| E2E-05 | **Dua karyawan searah** | EMP-A & EMP-B booking kendaraan berbeda di jam berdekatan → ADM gabung (MG-01) → satu supir, satu kendaraan → mulai & selesai sekali untuk keduanya (MG-10) → rating hanya dari booking utama (MG-11) |
| E2E-06 | **Dinas SPD 2 hari** | Booking SPD K1 H+2 06:00 – H+3 18:00 → booking lain K1 / DRV-1 di H+2 atau H+3 ditolak (BC-14, SP-07) → selesai terlambat tanpa overtime (CP-03) |
| E2E-07 | **Booking terlupakan** | Booking PENDING tak direspons → IGNORED (TO-01) · booking APPROVED tak dimulai → EXPIRED → supir & kendaraan kembali kosong (TO-02, TO-03) |
| E2E-08 | **Servis kendaraan (diajukan admin)** | ADM membuat maintenance K2 (MT-01) → K2 tidak bisa dibooking & tidak muncul di picker → ADM selesaikan dengan foto bukti (MT-05) → K2 bisa dibooking lagi (MT-14). Isi BBM sebanyak apa pun tidak pernah membuat maintenance sendiri (MT-11) |
| E2E-09 | **Rapat di ruangan** | EMP-A booking R1 → approve (AP-07, RK-1 dapat notifikasi) → EMP-A mulai & selesai sendiri (ST-08, CP-07) → rating ruangan masuk ringkasan RK-1 (RT-06) |
| E2E-10 | **Supir keluar perusahaan** | Nonaktifkan akun DRV-2 (DU-04) saat masih punya booking APPROVED → cek picker, pemilihan otomatis, dan nasib booking-nya (DU-03 ❓) |

---

## Temuan dari membaca kode

> **Status (run 2026-09-30):** hasil pengujian nyata tiap temuan — terkonfirmasi, terbantah, dan temuan baru
> (B1–B21) — ada di [HASIL_TESTING_2026-09-30.md](HASIL_TESTING_2026-09-30.md). Tabel di bawah adalah
> dugaan awal sebelum diuji.

Perilaku berikut ditemukan dari membaca kode backend (commit `aed39c6`, 2026-09-30) dan
**belum dibuktikan** di aplikasi berjalan — skenario 🔴 terkait wajib diuji untuk memastikan.

| # | Temuan | Skenario | Dampak |
|---|---|---|---|
| T1 | ✅ _(diperbaiki 2026-09-30)_ Transisi EXPIRED/IGNORED **tidak melepas supir** yang memegang kendaraan sejak approve | TO-03, E2E-07 | Supir "sibuk" selamanya: tidak dipilih otomatis; kendaraan menolak supir lain |
| T2 | ✅ _(diperbaiki 2026-09-30)_ "Tugaskan kendaraan" pada booking APPROVED **tidak memperbarui** catatan supir pemegang kendaraan | AS-03, AS-04, E2E-03 | Supir lama tertahan; supir baru tidak tercatat memegang |
| T3 | ✅ _(diperbaiki 2026-09-30)_ Substitute hanya mengganti resource, **bukan** kendaraan yang ditugaskan; penanda "Dialihkan" tidak terisi | SB-02, SB-03 | Supir memegang kendaraan lama; bentrok dicek di kendaraan lama |
| T4 | ✅ _(diperbaiki 2026-09-30)_ Tidak ada pengecekan **bentrok ruangan** saat buat/approve | BC-11, RM-05 | Dua rapat bisa disetujui di ruangan & jam yang sama |
| T5 | ✅ _(diperbaiki 2026-09-30)_ Resource **INACTIVE** bisa dibooking | BC-08, VH-01 | Kendaraan nonaktif tetap dipakai |
| T6 | ✅ _(diperbaiki 2026-09-30)_ Tanggal **masa lalu** diterima saat buat booking | BC-04 | Booking langsung hangus / data kotor |
| T7 | ✅ _(diperbaiki 2026-09-30)_ Selesaikan booking & hapus maintenance **selalu** mengubah kendaraan jadi AVAILABLE | CP-10, MT-09 | Status kendaraan salah (MAINTENANCE/IN_USE tertimpa) |
| T8 | ✅ _(diperbaiki 2026-09-30)_ Maintenance terjadwal di masa depan langsung mengunci kendaraan; status "selesai" lewat edit/buat tidak membebaskan | MT-04, MT-07, MT-08 | Kendaraan terkunci tanpa alasan |
| T9 | ✅ _(diperbaiki 2026-09-30)_ DRIVER/ROOM_KEEPER bisa **membatalkan booking PENDING orang lain** (API) | RL-07 | Penyalahgunaan lewat API |
| T10 | ✅ _(diperbaiki 2026-09-30)_ Pencatatan BBM terbuka untuk semua role (API) | RL-08 | Data BBM dari pihak yang tidak berwenang |
| T11 | ✅ _(diperbaiki 2026-09-30)_ Admin boleh **menyetujui** booking sendiri, tapi tidak boleh **menolak** booking sendiri | AP-11 | Aturan tidak konsisten |
| T12 | ✅ _(diperbaiki 2026-09-30)_ Transisi otomatis hanya jalan saat daftar booking dibuka & tidak disiarkan | TO-05 | Dashboard/kalender/perangkat lain bisa menampilkan status lama |
| T13 | ✅ _(diperbaiki 2026-09-30)_ Status manual kendaraan tidak dicek terhadap booking/maintenance aktif | VH-02 | Status kendaraan tidak mencerminkan kenyataan |
| T15 | ✅ _(diperbaiki 2026-09-30)_ `POST /auth/register` publik & menerima `roleId` apa pun | AU-16 | **Keamanan**: siapa pun bisa membuat akun admin |
| T16 | Booking APPROVED tidak bisa dibatalkan siapa pun | CN-03, CN-04 | ❓ Perlu keputusan alur pembatalan |
| T17 | ✅ _(diperbaiki 2026-09-30)_ Mobile memberi ROOM_KEEPER izin ubah status ruangan, backend hanya ADMIN | RM-03 | Tombol tampil tapi selalu gagal 403 |
| T18 | ✅ _(diperbaiki 2026-09-30)_ Hapus kendaraan/ruangan yang punya booking gagal di foreign key tanpa penanganan | VH-07 | Error server (500) alih-alih pesan jelas |
| T19 | Profil di mobile tidak ikut berubah saat data akun diubah dari luar | AU-15 | Nama/role usang sampai app dibuka ulang |
