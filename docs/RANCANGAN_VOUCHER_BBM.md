# Rancangan Ulang: Voucher BBM Berbasis Odometer

> Status: **DISETUJUI** (2026-10-01) — implementasi bertahap, lihat §8.
> Menggantikan pencatatan BBM bebas yang sekarang, sekaligus menjawab FL-06
> (hapus catatan BBM) di `docs/SKENARIO_TESTING.md`.

## 1. Keputusan pemilik (hasil tanya-jawab)

| Topik | Keputusan |
|---|---|
| Rumus konsumsi | **Per kendaraan** — field km/liter di data kendaraan |
| Satuan saldo | **Liter**. Rupiah dihitung saat voucher terbit, pakai harga BBM saat itu |
| Pembuat voucher | **Admin** (web). Kalau mendadak (SPD / perjalanan jauh), **driver isi langsung** tanpa voucher: catat odometer, liter, struk |
| Isi di SPBU non-mitra / isi langsung | **Mengurangi saldo sesuai liter aktual**. Saldo boleh minus, dan voucher berikutnya jadi lebih kecil |
| Penukaran | Voucher **dicetak**, lalu **admin merekonsiliasi** dengan tagihan mitra |
| Konfirmasi pemakaian | **Driver konfirmasi di aplikasi** ("Sudah diisi" + foto struk) |
| Masa berlaku | Diatur di Pengaturan, **default 1 hari (hari yang sama, WIB)**. Tidak dipakai → kedaluwarsa, liter kembali ke saldo |
| Selisih saat rekonsiliasi | **Tidak ada**. Voucher dikirim ke SPBU dengan nominal tetap dan dianggap terpakai penuh |
| Pembatas | **Maksimal kapasitas tangki**. Sisa hak tetap di saldo |
| SPBU mitra | **Ada master SPBU mitra**. Voucher mencantumkan SPBU tujuan |
| Titik awal | Pakai odometer kendaraan yang bisa diubah admin (default 0 untuk kendaraan baru), dipakai sebagai titik awal saat pengisian pertama (lihat §3.7) |
| Listrik | Charging dilakukan di kantor. Pemilik minta saran proses kWh yang jelas (lihat §4) |

## 2. Konsep inti: Saldo BBM kendaraan

Setiap kendaraan punya **buku saldo BBM** dalam liter, disimpan di tabel `fuel_ledger`
(catatan hanya ditambah, tidak pernah diubah atau dihapus).

```
hak liter (akrual) = (odometer sekarang − odometer kejadian terakhir) ÷ km per liter
saldo              = saldo sebelumnya + hak liter − liter keluar
```

"Kejadian" adalah hal yang membawa bacaan odometer dan mengubah saldo:
voucher terbit, isi langsung, atau titik awal.

- **Voucher terbit**: liter voucher = `min(saldo tersedia, kapasitas tangki)`.
  Liter voucher langsung mengurangi saldo **saat terbit**, supaya tidak bisa
  diterbitkan dua kali.
- **Isi langsung** (SPD, darurat, SPBU non-mitra, atau SPBU mitra tanpa voucher):
  saldo dikurangi liter aktual di struk.
- **Voucher kedaluwarsa atau dibatalkan**: liter voucher dikembalikan ke saldo.

### Contoh (Avanza, 12 km/L, tangki 45 L, Pertalite Rp10.000)

| # | Kejadian | Odometer | Hak (L) | Keluar (L) | Saldo (L) | Keterangan |
|---|---|---|---|---|---|---|
| 1 | Titik awal | 50.000 | — | — | 0 | |
| 2 | Admin terbitkan voucher | 50.360 | 360 ÷ 12 = **30** | 30 | 0 | Voucher 30 L = **Rp300.000** |
| 3 | SPD: driver isi langsung di SPBU non-mitra | 50.900 | 540 ÷ 12 = 45 | 40 (struk) | **5** | Sisa hak 5 L terbawa |
| 4 | Admin terbitkan voucher | 51.200 | 300 ÷ 12 = 25 | 30 | 0 | 25 + 5 = 30 L |
| 5 | Driver isi langsung 50 L | 51.600 | 400 ÷ 12 ≈ 33,3 | 50 | **−16,7** | Isi lebih dari hak, saldo minus |
| 6 | Admin terbitkan voucher | 52.200 | 600 ÷ 12 = 50 | 33,3 | 0 | 50 − 16,7 = 33,3 L |

Jika saldo melebihi kapasitas tangki (mis. saldo 60 L, tangki 45 L), voucher hanya
45 L dan 15 L tetap di saldo. Saldo yang jauh di atas kapasitas tangki ditandai
peringatan: "kemungkinan ada pengisian yang belum dicatat".

Perubahan km/L kendaraan hanya berlaku untuk jarak **sesudah** kejadian terakhir.
Hak yang sudah tercatat tidak dihitung ulang.

## 3. Alur

### 3.1 Admin menerbitkan voucher (web)
1. Menu **Bahan Bakar → Saldo Kendaraan** → pilih kendaraan → **Terbitkan Voucher**.
2. Isian:
   - **Odometer sekarang**: terisi otomatis dari `currentOdometer`, tidak boleh lebih kecil.
   - **Jenis BBM**: sesuai energi kendaraan.
   - **SPBU mitra**.
   - **Driver penerima**: default driver tetap kendaraan, atau driver booking yang sedang ONGOING.
   - Booking (opsional), foto odometer (opsional), catatan.
3. Pratinjau hitungan tampil langsung: jarak, hak liter, saldo terbawa, batas tangki,
   **liter voucher**, harga per liter, **nominal Rp**, berlaku sampai.
4. **Terbitkan** → status `ISSUED`, kode unik (mis. `VB-260930-7K2Q`) + QR → halaman **Cetak**.
5. Aturan:
   - Maksimal **1 voucher aktif (`ISSUED`) per kendaraan**.
   - Voucher tidak bisa terbit bila km/L kendaraan belum diisi, atau bila liter hasil hitungan ≤ 0.

### 3.2 Driver memakai voucher (mobile)
1. Driver melihat voucher aktifnya di **Voucher BBM** (beranda dan detail booking ONGOING).
   Kode/QR juga bisa ditunjukkan dari layar ponsel.
2. Setelah mengisi di SPBU mitra, driver tekan **Sudah Diisi**, lalu isi foto struk
   (kamera, wajib) dan odometer saat isi (wajib, ≥ odometer voucher).
3. Status berubah menjadi `USED`. Sistem juga membuat catatan pengisian (`fuel_expenses`,
   sumber `VOUCHER`), sehingga biaya BBM tetap satu sumber untuk laporan. Catatan ini
   **tidak** mengurangi saldo lagi, karena saldo sudah dipotong saat voucher terbit.
4. Admin juga bisa menandai "Sudah Diisi" dari web, untuk driver yang tidak memegang
   aplikasi.

### 3.3 Kedaluwarsa (otomatis)
- `validUntil` = pukul 23:59:59 WIB pada hari ke-N (N dari Pengaturan, default 1 = hari
  yang sama).
- Sweeper backend yang sudah ada menjadikan `ISSUED` yang lewat waktu sebagai `EXPIRED`.
  Setelah itu:
  - ledger mencatat `VOUCHER_RETURN` (+liter);
  - audit log dicatat;
  - DATA_CHANGED `fuel` dan `vehicle` disiarkan lewat `Publisher` (tanpa polling).
- Jika ternyata tagihan mitra menunjukkan voucher itu dipakai, admin bisa **Tandai Terpakai**
  pada voucher `EXPIRED`. Status menjadi `USED`, dan ledger mencatat `VOUCHER_REINSTATE`
  (−liter).

### 3.4 Isi langsung tanpa voucher (driver/admin)
Ini pengganti form "Catat Pengisian" yang sekarang, disederhanakan:
- **Odometer saat isi**: wajib, ≥ `currentOdometer`. Satu isian saja; "odometer isi terakhir"
  ditampilkan otomatis. Kolom `odometerBefore`/`odometerAfter` tetap disimpan untuk kompatibilitas.
- **SPBU**: pilih mitra, atau "SPBU lain" dengan nama bebas (`stationName`).
- **Alasan**: SPD / Perjalanan jauh / Darurat / Lainnya.
- Jenis BBM, liter, harga per liter (default dari master), foto struk (kamera, wajib),
  catatan.
- Form menampilkan saldo dan hak liter saat ini. Muncul peringatan (tidak memblokir) bila
  liter > hak atau liter > kapasitas tangki.
- Tidak perlu persetujuan. Saldo langsung dikurangi (`DIRECT_FILL`).

### 3.5 Pembatalan catatan (pengganti hapus, jawaban FL-06)
Catatan pengisian dan voucher **tidak lagi dihapus permanen**. Admin melakukan
**Batalkan** dengan alasan wajib. Datanya tetap ada dengan status `VOID`/`CANCELLED`
dan tampil dicoret.
- **Liter keluar** dikembalikan ke saldo (entri `VOID`).
- **Odometer tidak mundur**, karena odometer adalah fakta kendaraan. Satu pengecualian:
  jika catatan itu **kejadian terakhir** kendaraan, admin bisa mencentang
  **"Odometer di catatan ini salah ketik"**. Maka:
  - hak liternya ikut dibatalkan;
  - `currentOdometer` dikembalikan ke bacaan sah tertinggi yang tersisa (catatan BBM/voucher,
    laporan trip, dan titik awal yang masih berlaku).

  Untuk catatan di tengah riwayat, koreksi dilakukan lewat **Penyesuaian Saldo**.
- `DELETE /fuel-expenses/:id` tetap ada demi kompatibilitas aplikasi lama, tetapi isinya
  kini melakukan pembatalan (soft).

### 3.6 Penyesuaian saldo (admin)
Tambah atau kurangi liter secara manual dengan alasan wajib, mis. isi tangki awal atau
koreksi. Tercatat di ledger (`ADJUSTMENT`) dan audit log.

### 3.7 Titik awal
Kendaraan punya **"Odometer awal BBM"**:
- default = `currentOdometer` kendaraan (0 untuk kendaraan baru);
- bisa diubah admin **selama belum ada kejadian BBM**;
- dipakai sebagai titik hitung pengisian/voucher pertama.

Saat fitur diluncurkan, kendaraan yang sudah ada mulai dari odometer saat ini dengan
saldo 0. Catatan BBM lama tetap tampil sebagai riwayat, tetapi tidak memengaruhi saldo.
_(Disetujui 2026-10-01.)_

### 3.8 Rekonsiliasi tagihan mitra (admin, web)
Menu **Voucher**:
1. Filter SPBU + periode + status `USED`.
2. Cocokkan dengan tagihan mitra.
3. Pilih beberapa voucher → **Rekonsiliasi** (no. tagihan).
4. Sistem mengisi `reconciledAt` dan `invoiceNumber`.

Hasilnya bisa diekspor ke Excel per mitra/periode. Voucher yang ada di tagihan tetapi
berstatus `EXPIRED` → **Tandai Terpakai** (§3.3).

### Status voucher
```
ISSUED ──(driver/admin: Sudah Diisi)──▶ USED ──(admin: rekonsiliasi)──▶ USED + reconciledAt
  │                                       │
  ├──(lewat masa berlaku)──▶ EXPIRED ──(admin: Tandai Terpakai)──▶ USED
  └──(admin: Batalkan)─────▶ CANCELLED ◀──(admin: Batalkan, alasan)──┘
```

## 4. Usulan untuk kendaraan listrik (charging di kantor)

Tidak memakai voucher, tetapi tetap memakai **buku saldo yang sama dengan satuan kWh**,
supaya perhitungannya jelas dan bisa diaudit.

1. **Data kendaraan**: km per kWh (efisiensi standar) dan kapasitas baterai (kWh).
2. **Sesi charging kantor dicatat** oleh driver/admin:
   - odometer (wajib);
   - baterai % sebelum/sesudah (kolomnya sudah ada di tabel);
   - **kWh terpakai**:
     - **bebas** (belum diketahui apakah charger kantor punya meter): form menerima
       salah satu — angka meter awal & akhir (+ foto meter), kWh langsung, atau bila
       keduanya kosong dipakai estimasi `(%sesudah − %sebelum) × kapasitas baterai ÷ 0,9`
       (0,9 = rugi-rugi charger);
   - biaya = kWh × tarif listrik kantor (master "Listrik PLN").
3. **Saldo kWh** = hak kWh (km ÷ km/kWh) − kWh aktual. Laporan efisiensi per kendaraan
   membandingkan hak dan aktual. Selisih di atas toleransi (Pengaturan, mis. 15%) ditandai
   "boros / periksa". Contoh penyebabnya: pemakaian di luar dinas atau charger dipakai
   kendaraan lain.
4. Charging di luar kantor (SPKLU) saat perjalanan dicatat sebagai **isi langsung** dengan
   struk.
5. **Hybrid**: buku saldo BBM (voucher berlaku) dan buku saldo kWh berjalan terpisah.

## 5. Model data (migrasi `000017_fuel_ledger.up.sql`, idempoten)

Semua memakai `IF NOT EXISTS` / `ON CONFLICT DO NOTHING` / `DO $$ … EXCEPTION WHEN
duplicate_object`, karena CI menjalankan ulang semua `*.up.sql` di setiap deploy.

```sql
-- Kendaraan
ALTER TABLE vehicles ADD COLUMN IF NOT EXISTS "kmPerLiter"           NUMERIC(6,2) NULL CHECK ("kmPerLiter" > 0);
ALTER TABLE vehicles ADD COLUMN IF NOT EXISTS "tankCapacityLiter"    NUMERIC(6,2) NULL CHECK ("tankCapacityLiter" > 0);
ALTER TABLE vehicles ADD COLUMN IF NOT EXISTS "kmPerKwh"             NUMERIC(6,2) NULL CHECK ("kmPerKwh" > 0);
ALTER TABLE vehicles ADD COLUMN IF NOT EXISTS "batteryCapacityKwh"   NUMERIC(6,2) NULL CHECK ("batteryCapacityKwh" > 0);
ALTER TABLE vehicles ADD COLUMN IF NOT EXISTS "fuelBaselineOdometer" INTEGER NULL CHECK ("fuelBaselineOdometer" >= 0);
-- backfill sekali: baseline = odometer saat ini; km/L awal dari kategori (lihat §9 no. 2)
UPDATE vehicles SET "fuelBaselineOdometer" = "currentOdometer" WHERE "fuelBaselineOdometer" IS NULL;

-- SPBU mitra
CREATE TABLE IF NOT EXISTS fuel_stations (
  id SERIAL PRIMARY KEY, name VARCHAR(150) NOT NULL UNIQUE, address TEXT NULL,
  phone VARCHAR(30) NULL, "contactPerson" VARCHAR(100) NULL,
  "isActive" BOOLEAN NOT NULL DEFAULT TRUE,
  "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(), "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW());

-- Voucher
-- enum fuel_voucher_status: ISSUED, USED, EXPIRED, CANCELLED
CREATE TABLE IF NOT EXISTS fuel_vouchers (
  id SERIAL PRIMARY KEY, code VARCHAR(20) NOT NULL UNIQUE,
  "vehicleId" INT NOT NULL REFERENCES vehicles(id), "fuelTypeId" INT NOT NULL REFERENCES fuel_types(id),
  "stationId" INT NOT NULL REFERENCES fuel_stations(id),
  "driverId" INT NULL REFERENCES drivers(id), "bookingId" INT NULL REFERENCES bookings(id),
  "issuedById" INT NOT NULL REFERENCES users(id),
  odometer INT NOT NULL, "odometerPhotoUrl" VARCHAR(255) NULL,
  "distanceKm" INT NOT NULL, "kmPerLiter" NUMERIC(6,2) NOT NULL,     -- snapshot
  "accruedLiter" NUMERIC(10,2) NOT NULL, "carriedLiter" NUMERIC(10,2) NOT NULL,
  liter NUMERIC(10,2) NOT NULL CHECK (liter > 0),
  "pricePerLiter" NUMERIC(12,2) NOT NULL, amount NUMERIC(14,2) NOT NULL,
  "validUntil" TIMESTAMPTZ NOT NULL, status fuel_voucher_status NOT NULL DEFAULT 'ISSUED',
  "usedAt" TIMESTAMPTZ NULL, "usedById" INT NULL REFERENCES users(id), "usedOdometer" INT NULL,
  "receiptPhotoUrl" VARCHAR(255) NULL, "fuelExpenseId" INT NULL REFERENCES fuel_expenses(id),
  "cancelledAt" TIMESTAMPTZ NULL, "cancelledById" INT NULL REFERENCES users(id), "cancelReason" TEXT NULL,
  "reconciledAt" TIMESTAMPTZ NULL, "reconciledById" INT NULL REFERENCES users(id), "invoiceNumber" VARCHAR(100) NULL,
  note TEXT NULL, "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW(), "updatedAt" TIMESTAMPTZ NOT NULL DEFAULT NOW());
CREATE UNIQUE INDEX IF NOT EXISTS uq_fuel_vouchers_one_active ON fuel_vouchers("vehicleId") WHERE status = 'ISSUED';

-- Buku saldo (append-only)
-- enum fuel_ledger_type: OPENING, VOUCHER, DIRECT_FILL, VOUCHER_RETURN, VOUCHER_REINSTATE, VOID, ADJUSTMENT
CREATE TABLE IF NOT EXISTS fuel_ledger (
  id SERIAL PRIMARY KEY, "vehicleId" INT NOT NULL REFERENCES vehicles(id),
  energy fuel_category NOT NULL,                       -- BBM (liter) / LISTRIK (kWh)
  "entryType" fuel_ledger_type NOT NULL,
  odometer INT NULL, "distanceKm" INT NULL, "kmPerUnit" NUMERIC(6,2) NULL,
  accrued NUMERIC(10,2) NOT NULL DEFAULT 0, debit NUMERIC(10,2) NOT NULL DEFAULT 0,  -- debit negatif = kredit
  "balanceAfter" NUMERIC(10,2) NOT NULL,
  "voucherId" INT NULL REFERENCES fuel_vouchers(id), "fuelExpenseId" INT NULL REFERENCES fuel_expenses(id),
  "createdById" INT NULL REFERENCES users(id),        -- NULL = aksi sistem (sweeper)
  note TEXT NULL, "createdAt" TIMESTAMPTZ NOT NULL DEFAULT NOW());

-- Pengisian (tabel lama diperluas)
ALTER TABLE fuel_expenses ADD COLUMN IF NOT EXISTS source      VARCHAR(10) NOT NULL DEFAULT 'DIRECT'; -- DIRECT | VOUCHER
ALTER TABLE fuel_expenses ADD COLUMN IF NOT EXISTS "voucherId" INT NULL REFERENCES fuel_vouchers(id);
ALTER TABLE fuel_expenses ADD COLUMN IF NOT EXISTS "stationId" INT NULL REFERENCES fuel_stations(id);
ALTER TABLE fuel_expenses ADD COLUMN IF NOT EXISTS reason      VARCHAR(20) NULL;  -- SPD | LONG_TRIP | EMERGENCY | OTHER
ALTER TABLE fuel_expenses ADD COLUMN IF NOT EXISTS "voidedAt"  TIMESTAMPTZ NULL;
ALTER TABLE fuel_expenses ADD COLUMN IF NOT EXISTS "voidedById" INT NULL REFERENCES users(id);
ALTER TABLE fuel_expenses ADD COLUMN IF NOT EXISTS "voidReason" TEXT NULL;
-- (listrik, fase 4) "meterStartKwh"/"meterEndKwh" NUMERIC NULL

-- Pengaturan
INSERT INTO master_settings (key, value, unit, description) VALUES
  ('fuel_voucher_validity_days', 1, 'hari', 'Masa berlaku voucher BBM (1 = hari yang sama, WIB)')
ON CONFLICT (key) DO NOTHING;
```

**Konsistensi:** setiap penulisan saldo berjalan dalam satu transaksi dengan
`SELECT … FROM vehicles WHERE id = $1 FOR UPDATE`. Dengan begitu dua voucher/pengisian
serentak pada kendaraan yang sama tidak bisa membaca saldo yang sama.
`balanceAfter` dihitung dari baris ledger terakhir kendaraan+energi tersebut.

**Harga BBM satu sumber:** saat ini harga ada di `fuel_types.default_price` (dipakai form)
dan di `master_settings.fuel_price_*` (angkanya berbeda). Usulan: voucher memakai
**`fuel_types.default_price`**, dan entri `fuel_price_*` dihapus (seed-nya di 000001 ikut dibuang). `GET /settings/fuel-prices`
dibaca dari `fuel_types` supaya kompatibel. **Disetujui: satu master harga saja.**

## 6. API

| Method & path | Role | Keterangan |
|---|---|---|
| `GET /fuel-balances` | ADMIN | Saldo semua kendaraan: km/L, odometer kejadian terakhir, odometer kini, jarak tertunda, saldo L, estimasi Rp, voucher aktif, peringatan |
| `GET /fuel-balances/:vehicleId` | ADMIN, DRIVER | Saldo satu kendaraan + pratinjau hak |
| `GET /fuel-balances/:vehicleId/ledger?energy=` | ADMIN | Mutasi saldo (paginated) |
| `POST /fuel-balances/:vehicleId/adjustments` | ADMIN | Penyesuaian saldo `{energy, amount ±, note}` |
| `PUT /fuel-balances/:vehicleId/profile` | ADMIN | km/L, tangki, km/kWh, baterai, odometer awal (terkunci setelah ada kejadian) |
| `POST /fuel-vouchers/preview` | ADMIN | Hitung tanpa menyimpan (untuk pratinjau form) |
| `POST /fuel-vouchers` | ADMIN | Terbitkan voucher |
| `GET /fuel-vouchers` | ADMIN semua; DRIVER miliknya | Filter status/kendaraan/SPBU/periode/rekonsiliasi |
| `GET /fuel-vouchers/:id` | ADMIN, DRIVER penerima | Detail + data cetak |
| `PATCH /fuel-vouchers/:id/use` | DRIVER penerima, ADMIN | Multipart: foto struk + odometer → USED (juga dari EXPIRED, khusus ADMIN) |
| `PATCH /fuel-vouchers/:id/cancel` | ADMIN | Alasan wajib → CANCELLED, liter kembali |
| `POST /fuel-vouchers/reconcile` | ADMIN | `{ids[], invoiceNumber}` |
| `CRUD /fuel-stations` | GET semua login; tulis ADMIN | Master SPBU mitra |
| `POST /fuel-expenses` | ADMIN, DRIVER | Isi langsung (diperluas: `stationId`/`stationName`, `reason`, `odometer`) → ledger `DIRECT_FILL` |
| `PATCH /fuel-expenses/:id/void` | ADMIN | Alasan + opsi "odometer salah ketik" |
| `DELETE /fuel-expenses/:id` | ADMIN | Kompatibilitas → sama dengan void |

**DATA_CHANGED** (`internal/middleware/data_changed.go` → `TopicsForPath` + test):
`fuel-expenses`, `fuel-vouchers`, `fuel-balances` → `fuel` + `vehicle` (kecuali `…/preview`); `fuel-stations` → `fuel`.
`fuel-expenses` juga ditambah `vehicle`, karena memajukan odometer dan mengubah saldo.
Kedaluwarsa oleh sweeper → `Publisher("fuel","vehicle")`.
Web: tambah query key baru (`FUEL_VOUCHERS`, `FUEL_BALANCES`, `FUEL_STATIONS`, `FUEL_LEDGER`)
ke `SYNC_TOPIC_QUERY_KEYS.fuel` (dan `vehicle` untuk saldo).
Mobile: `DataSync.invalidate({fuel, vehicle})` setelah tulis; controller voucher/saldo
memakai `DataSyncMixin` dengan `syncTopics: {fuel, vehicle}`.

## 7. Tampilan

### Web (admin)
- **Bahan Bakar** menjadi tab:
  - **Saldo Kendaraan**: tabel saldo + tombol *Terbitkan Voucher*, *Penyesuaian*, *Mutasi*.
  - **Voucher**: daftar, filter, *Cetak*, *Tandai Terpakai*, *Batalkan*, pilih banyak → *Rekonsiliasi*, export Excel.
  - **Pengisian**: daftar lama + kolom sumber (Voucher/Langsung), SPBU, alasan, status dibatalkan.
- **Modal Terbitkan Voucher** dengan pratinjau hitungan langsung (§3.1).
- **Halaman cetak voucher** (`/fuel/vouchers/[id]/print`) **format printer thermal** — lebar 80 mm
  (default) atau 58 mm, dipilih di halaman cetak; `@page { size: 80mm auto }`, font
  monospace, QR ±30 mm, tanpa warna. Isi:
  kode + QR, kendaraan/plat, driver, SPBU, jenis BBM, **liter**, **nominal Rp**,
  berlaku sampai, tanda tangan admin.
- **Form kendaraan**: km/L, kapasitas tangki, odometer awal BBM
  (+ km/kWh & kapasitas baterai untuk LISTRIK/HYBRID).
- **Pengaturan**: *SPBU Mitra* (CRUD), *Masa berlaku voucher*, harga di *Jenis Bahan Bakar*
  (satu sumber).
- **Laporan**:
  - rekonsiliasi per SPBU mitra (terbit / terpakai / kedaluwarsa / belum direkonsiliasi, liter & Rp);
  - efisiensi per kendaraan (km, hak L, aktual L, selisih %).

### Mobile
- **Driver**:
  - kartu **Voucher BBM aktif** di beranda dan di detail booking ONGOING;
  - layar detail voucher (kode/QR besar untuk ditunjukkan);
  - tombol **Sudah Diisi** (kamera struk + odometer);
  - riwayat voucher.
- **Isi Langsung** (form lama dirombak):
  - odometer **wajib**. Ini sekaligus memperbaiki bug sekarang: form menulis "opsional",
    tetapi backend menolak tanpa odometer;
  - saldo & hak liter tampil, pilih SPBU mitra / SPBU lain, alasan, peringatan melebihi hak/tangki.
- **Admin**: lihat saldo & voucher (read-only di fase awal; menerbitkan tetap dari web).

## 8. Rencana implementasi bertahap

Tiap fase: branch fitur di masing-masing repo → test (Go test `-count=N -shuffle=on`
dengan `GOTOOLCHAIN=go1.26.2`, runner skenario lokal) → update `SKENARIO_TESTING.md` →
**minta izin sebelum merge/push ke `develop`** (auto-deploy produksi).

| Fase | Backend | Web | Mobile | Hasil setelah deploy |
|---|---|---|---|---|
| **1. Fondasi saldo** | Migrasi 000017 (kendaraan, SPBU, ledger, kolom fuel_expenses, setting); `FuelLedgerService` (transaksi + row lock); isi langsung menulis ledger; void menggantikan hapus; `/fuel-balances`, `/fuel-ledger`, `/fuel-stations`; topik DATA_CHANGED; unit test rumus & kasus contoh §2 | Field kendaraan; tab Saldo & Mutasi; Pengaturan SPBU Mitra; Batalkan catatan | Odometer wajib; saldo & peringatan di form isi; pilih SPBU & alasan | Saldo mulai terhitung dari odometer saat ini; belum ada voucher |
| **2. Voucher** | `/fuel-vouchers` (preview, terbit, use, cancel), kedaluwarsa di sweeper + Publisher, audit | Modal terbit + pratinjau, tab Voucher, halaman cetak | Voucher aktif, detail/QR, Sudah Diisi, riwayat | Alur voucher end-to-end |
| **3. Rekonsiliasi & laporan** | Reconcile bulk, query laporan per mitra & efisiensi, export Excel | Rekonsiliasi, laporan | Ringkasan di tab laporan (bila perlu) | Rekonsiliasi tagihan bulanan |
| **4. Listrik** | Ledger kWh, kolom meter, toleransi | Form sesi charging, laporan efisiensi kWh | Form charging kantor | Sesuai jawaban §9 no. 5 |

Skenario testing baru (bagian **VC-xx**) mencakup:
- rumus & saldo terbawa;
- batas tangki;
- saldo minus;
- 1 voucher aktif;
- kedaluwarsa → saldo kembali;
- Tandai Terpakai dari EXPIRED;
- batal voucher/pengisian;
- koreksi odometer salah ketik (kejadian terakhir vs di tengah);
- penerbitan serentak;
- sinkronisasi web ↔ mobile.

FL-06 diganti menjadi "Batalkan catatan BBM".

## 9. Keputusan tambahan (2026-10-01)

1. Titik awal: sesuai §3.7 (saldo 0 di odometer saat ini untuk kendaraan yang sudah berjalan).
2. km/L awal per kategori: MPV 12, SUV 9, Sedan 13, Pickup 10, Bus/Minibus 7, lainnya 10
   (diisi hanya bila kosong; admin menyesuaikan per kendaraan). Kapasitas tangki wajib
   sebelum voucher terbit.
3. **Nominal tidak dibulatkan**: liter 2 desimal, Rp = liter × harga (2 desimal, ditampilkan
   tanpa pembulatan ke ribuan).
4. Cetak voucher: **printer thermal** (80 mm default, 58 mm opsional).
5. Listrik: pencatatan kWh **bebas** (meter awal/akhir, kWh langsung, atau estimasi % baterai).
6. Harga BBM: **satu master** — `fuel_types.default_price`.
