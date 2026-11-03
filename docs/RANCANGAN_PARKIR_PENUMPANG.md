# Rancangan: Parkir & Reimburse Supir, Penumpang Booking, Fix Tanggal Dashboard Mobile

Status: **SELESAI DIKERJAKAN (3 Nov 2026)** — keputusan pemilik produk di §6. Migrasi `000022_parking_passengers`.
Tanggal: 3 November 2026.

Terkait: `docs/RANCANGAN_BIAYA_PERJALANAN.md` (tol, dua tingkat), `docs/RANCANGAN_VOUCHER_BBM.md` (BBM).

---

## 1. Kondisi sekarang (hasil penelusuran kode)

### 1.1 Tol (`trip_expenses`)
| | Mode SEDERHANA (aktif) | Mode LENGKAP (tersembunyi, `trip_expense_full_mode = 1`) |
|---|---|---|
| Jenis biaya | Tol saja | Tol, **Parkir**, Lainnya |
| Isian | Tol Masuk/Keluar, nominal, foto struk | + gerbang tol, **cara bayar** (e-money kantor / uang pribadi) |
| Uang siapa | **Tidak dicatat** | Dicatat per transaksi |
| Reimburse | Tidak ada | Transaksi "uang pribadi" → rekap per supir → admin **Tandai Dibayar** |
| Menu web Reimburse Supir | Disembunyikan | Tampil |

Satu perjalanan **sudah bisa** punya banyak catatan (daftar, bukan satu kolom).
Biaya ini sudah masuk laporan keuangan (kolom "Tol" / "Tol & Parkir").

### 1.2 BBM (`fuel_expenses`)
- Dua sumber: **Voucher** (SPBU mitra, ditagih ke kantor lalu direkonsiliasi admin) dan **Isi langsung**.
- Isi langsung mencatat liter, harga, odometer, foto struk — **tidak mencatat uang siapa yang dipakai**,
  jadi tidak pernah ada reimburse BBM.

### 1.3 Penumpang
- Booking hanya punya `passengerCount` (angka). Tidak ada daftar siapa yang ikut.

### 1.4 Bug tanggal di Dashboard mobile
- Kartu **"Booking Saya Terbaru"** (dan daftar booking lain di Dashboard) dibentuk dari `GET /bookings`,
  yang mengirim waktu sebagai **`startDate` / `endDate`**.
- `RecentBookingModel` (mobile, `features/dashboard/data/models/dashboard_summary_model.dart`) membaca
  `start_time` / `startTime` → selalu kosong → jatuh ke **`DateTime.now()`**.
- Akibatnya semua kartu menampilkan tanggal & jam **saat Dashboard dibuka** (di foto: 02 Nov 2026, 15:29).
- Daftar di menu **Booking** memakai model lain (`BookingModel`, `startDate`) dan tidak terdampak.

---

## 2. Uang parkir per booking + reimburse supir

### 2.1 Keputusan (3 Nov 2026)
- **Admin menentukan per booking** apakah kantor membekali **uang parkir** dan **berapa nominalnya**.
- Supir mencatat **setiap kali parkir** (satu perjalanan bisa berkali-kali) beserta **foto karcis/struk**.
- Sistem menghitung **selisih** di akhir perjalanan:

| Kondisi | Hasil |
|---|---|
| Tidak dibekali | Seluruh biaya parkir = uang supir → **diganti ke supir** (reimburse) |
| Dibekali, terpakai **kurang** dari bekal | **Sisa** dikembalikan supir ke kantor |
| Dibekali, terpakai **lebih** dari bekal | **Kekurangan** diganti ke supir (reimburse) |
| Dibekali, terpakai **pas** | Selesai, tidak ada tindakan |

Contoh: dibekali Rp 50.000. Parkir 3 kali: Rp 5.000 + Rp 10.000 + Rp 15.000 = Rp 30.000 → sisa **Rp 20.000**
dikembalikan supir. Kalau parkirnya Rp 65.000 → kekurangan **Rp 15.000** diganti ke supir.

- **Tol tidak berubah** (mode Sederhana: tol masuk/keluar, tanpa cara bayar). **BBM tidak berubah.**

### 2.2 Data
- Catatan parkir = `trip_expenses` kategori `PARKING` (tabel yang sama dengan tol, jadi laporan & sinkronisasi
  sudah jalan). Lokasi parkir memakai kolom `note` yang sudah ada.
- Model "bekal" ini berlaku untuk parkir di **kedua** mode. Di mode Lengkap (tersembunyi), pilihan cara bayar per
  transaksi tetap untuk tol & lainnya, tidak untuk parkir.

Migrasi `000022` (idempoten):
```sql
-- Bekal uang parkir dari kantor. NULL / 0 = tidak dibekali.
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS "parkingAllowance" NUMERIC(14,2) NULL;

-- Penyelesaian selisih parkir per booking (satu kali per booking).
CREATE TABLE IF NOT EXISTS parking_settlements (
  id                SERIAL PRIMARY KEY,
  "bookingId"       INT NOT NULL UNIQUE REFERENCES bookings(id) ON DELETE CASCADE,
  "driverId"        INT NOT NULL REFERENCES drivers(id),
  direction         VARCHAR(10) NOT NULL,      -- TO_DRIVER (diganti ke supir) | TO_OFFICE (sisa dikembalikan)
  amount            NUMERIC(14,2) NOT NULL,
  "reimbursementId" INT NULL REFERENCES trip_reimbursements(id), -- TO_DRIVER: masuk pembayaran reimburse
  "settledAt"       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  note              TEXT NULL,
  "createdById"     INT NOT NULL REFERENCES users(id)
);
```

### 2.3 Aturan
- Bekal diatur **admin** kapan saja sebelum selisih diselesaikan (sebelum, saat, atau sesudah perjalanan).
- Catat parkir: supir yang ditugaskan saat perjalanan berjalan; admin bisa mencatat susulan (sama seperti tol).
- Selisih baru bisa diselesaikan setelah perjalanan **kembali/selesai** (RETURNED/COMPLETED), supaya tidak
  ada parkir yang tertinggal.
- Setelah selisih diselesaikan, catatan parkir & bekal booking itu **terkunci** (tidak bisa ditambah/dibatalkan/
  diubah) — sama seperti tol yang sudah di-reimburse.
- Booking digabung (merge): bekal & parkir dihitung **per perjalanan gabungan** (satu supir, satu mobil),
  dicatat di booking utama.

### 2.4 Tampilan
- **Detail booking → Catatan Perjalanan**: tab **"Tol"** menjadi **"Tol & Parkir"**.
  - Kartu **Uang Parkir**: bekal (admin: tombol **Atur**), terpakai, dan hasilnya —
    "Sisa Rp 20.000 dikembalikan supir" / "Kekurangan Rp 15.000 diganti ke supir" / "Tidak dibekali — Rp 30.000
    diganti ke supir" — plus status **Belum diselesaikan / Sudah diselesaikan (tanggal)**.
  - Tombol **Catat Tol** dan **Catat Parkir**; daftar catatan dengan badge jenis, nominal, lokasi, foto.
- **Web, menu "Reimburse Supir"** (kini tampil untuk admin di mode Sederhana):
  - **Diganti ke supir**: per supir, daftar booking dengan kekurangan/tanpa bekal → centang beberapa → **Tandai
    Dibayar** (tanggal + catatan). Tercatat di riwayat pembayaran yang sudah ada.
  - **Sisa dikembalikan supir**: daftar booking dengan sisa bekal → **Tandai Sudah Dikembalikan**.
- **Mobile**: supir melihat kartu Uang Parkir yang sama (baca saja) dan mencatat parkir. Pengaturan bekal &
  penyelesaian hanya di web (sesuai kebijakan mobile = versi ringkas).
- **Laporan keuangan**: kolom "Tol" menjadi "Tol & Parkir" (biaya parkir dihitung dari catatan, bukan dari bekal).

### 2.5 API
| Endpoint | Siapa | Isi |
|---|---|---|
| `POST /bookings/:id/trip-expenses` | supir/admin | mode Sederhana menerima `category` `TOLL` atau `PARKING` (parkir: nominal, foto, lokasi opsional) |
| `PATCH /bookings/:id/parking-allowance` | admin | `{ amount }` — 0 = tidak dibekali |
| `GET /bookings/:id/trip-expenses` | yang boleh melihat booking | ringkasan + `parking: { allowance, used, balance, direction, settled }` |
| `GET /parking-settlements/pending` | admin | booking selesai dengan selisih belum diselesaikan, dikelompokkan per supir |
| `POST /parking-settlements` | admin | selesaikan: `{ bookingIds[], direction, settledAt, note }`; TO_DRIVER sekaligus membuat pembayaran reimburse |

Sinkronisasi: semua tulis memicu topik `booking` (web `DATA_CHANGED`, mobile `DataSync`).

---

## 3. Karyawan yang ikut (penumpang)

### 3.1 Alur
1. **Buat booking kendaraan** (web & mobile): kartu baru **"Karyawan yang ikut"** — cari nama / ID karyawan,
   pilih beberapa. Pemesan tidak perlu dipilih (otomatis ikut).
2. **Jumlah penumpang** otomatis minimal = 1 (pemesan) + jumlah karyawan terpilih; masih bisa ditambah untuk
   tamu/non-karyawan. Kapasitas kendaraan tetap dicek seperti sekarang.
3. **Detail booking**: bagian **"Penumpang"** berisi pemesan + karyawan yang ikut, masing-masing dengan
   departemen dan **nomor HP** (tombol Telepon & WhatsApp, komponen `PhoneContact` yang sudah ada).
   Nomor kosong → "No. telepon belum diisi".
4. **Ubah daftar**: pemesan (selama PENDING/APPROVED) dan admin bisa menambah/mengurangi karyawan yang ikut.
5. **Booking digabung (merge)**: supir melihat gabungan penumpang dari semua booking dalam satu perjalanan.
6. Hanya untuk booking **kendaraan** (ruangan tidak).

### 3.2 Data (migrasi `000022`)
```sql
CREATE TABLE IF NOT EXISTS booking_passengers (
  id          SERIAL PRIMARY KEY,
  "bookingId" INT NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
  "userId"    INT NOT NULL REFERENCES users(id),
  "addedById" INT NULL REFERENCES users(id),
  "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE ("bookingId", "userId")
);
```

### 3.3 API
| Endpoint | Siapa | Isi |
|---|---|---|
| `GET /users/lookup?q=` | semua yang login | cari karyawan aktif: id, nama, ID karyawan, departemen (**tanpa** nomor HP — daftar pencarian tidak membocorkan nomor) |
| `POST /bookings` | pemesan | + `passengerIds: number[]` |
| `PUT /bookings/:id/passengers` | pemesan (PENDING/APPROVED) & admin | ganti daftar |
| `GET /bookings/:id` | yang boleh melihat booking | + `passengers: [{id, name, employeeId, department, phoneNumber}]` |

Validasi: karyawan harus aktif, tidak boleh duplikat, pemesan sendiri diabaikan, maksimal = kapasitas − 1.

### 3.4 Notifikasi, akses, dan hitungan
- Karyawan yang ikut mendapat **notifikasi** saat booking **disetujui**, saat **supir/kendaraan ditugaskan atau
  dialihkan**, dan saat booking **dibatalkan/ditolak**.
- Mereka boleh **membuka detail** booking itu (baca saja): jadwal, kendaraan, supir + nomor supir, penumpang lain.
  Tidak bisa membatalkan, mengubah, atau menilai supir — itu tetap hak pemesan.
- **Booking tetap terhitung satu**: karyawan yang ikut tidak membuat booking baru. Booking itu tidak masuk
  hitungan "Booking Saya", statistik, maupun laporan atas nama mereka — semua tetap atas nama pemesan
  (laporan per departemen pun memakai departemen pemesan).
- Supaya mudah dibuka kembali: di Dashboard karyawan ada bagian kecil **"Perjalanan yang Anda ikuti"**
  (yang akan datang & sedang berjalan), selain lewat notifikasi.

---

## 4. Fix tanggal Dashboard mobile
- `RecentBookingModel.fromJson` membaca `startDate`/`endDate` (tetap menerima `startTime` lama sebagai cadangan).
- Tanggal yang benar-benar tidak ada **tidak lagi** diganti waktu sekarang: kartu menampilkan "-".
- Test dengan respons asli `GET /bookings` sebagai fixture.
- Ikut diperiksa: tempat lain yang memakai `DateTime.now()` sebagai cadangan tanggal (lampiran, biaya perjalanan).

Ini perbaikan bug — tidak butuh keputusan; dikerjakan bersama yang lain setelah rancangan disetujui.

---

## 5. Urutan pengerjaan
1. Fix tanggal Dashboard mobile (kecil, berdiri sendiri).
2. Uang parkir: bekal per booking, catat parkir, selisih & penyelesaian (backend → web → mobile).
3. Karyawan yang ikut (backend → web → mobile).
Masing-masing diuji di database test + unit test + `tsc`/`next build` + `flutter analyze/test`.

---

## 6. Keputusan pemilik produk (3 Nov 2026)
| # | Pertanyaan | Keputusan |
|---|---|---|
| K1 | Parkir: uang siapa? | **Admin menentukan bekal uang parkir per booking (nominal)**; supir mencatat tiap parkir + bukti; selisih → sisa dikembalikan / kekurangan diganti ke supir |
| K2 | Tol ikut diberi pilihan cara bayar? | **Tidak**, tol tetap seperti sekarang |
| K3 | BBM isi langsung ikut reimburse? | **Tidak sekarang** |
| K4 | Karyawan yang ikut: notifikasi & lihat detail? | **Ya**, baca saja — dan **booking tetap terhitung satu** |

---

## 7. Tambahan: tamu tanpa akun (4 Nov 2026)
- Penumpang boleh orang yang **tidak punya akun**: isi manual **nama** (wajib), **nomor HP** & **keterangan**
  (mis. instansi/jabatan) opsional. Disimpan di `booking_passengers` dengan `userId` NULL (migrasi `000023`).
- API: `POST /bookings` & `PUT /bookings/:id/passengers` menerima `guests: [{name, phoneNumber?, note?}]`.
  Di `PUT`, `guests` tidak dikirim = tamu tidak diubah (klien lama aman); `[]` = hapus semua tamu.
- Detail booking: `passengers[]` berisi `role: "GUEST"` (`userId` 0, `department` = keterangan).
- Kapasitas: karyawan + tamu ≤ kapasitas − 1; jumlah penumpang otomatis disesuaikan.
- Tamu tidak mendapat notifikasi dan tidak bisa membuka booking.
