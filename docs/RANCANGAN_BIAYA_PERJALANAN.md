# Rancangan Biaya Perjalanan (Tol, Parkir, Lainnya)

Status: diimplementasikan 2026-10-14. Dua tingkat fitur memakai satu tabel dan
satu set endpoint; tingkat yang tampil dipilih lewat master setting
tersembunyi `trip_expense_full_mode` (0 = SEDERHANA, 1 = LENGKAP).

## 1. Dua tingkat fitur

| | SEDERHANA (aktif, default) | LENGKAP (tersembunyi) |
|---|---|---|
| Jenis | Tol saja | Tol, Parkir, Lainnya |
| Isian tol | Titik bayar: **Tol Masuk** / **Tol Keluar** | Gerbang masuk + gerbang keluar (ketik, dengan saran dari riwayat) |
| Nominal | Wajib | Wajib |
| Foto struk / bukti saldo | Wajib | Wajib |
| Cara bayar | – (tidak dicatat) | **E-money kantor** atau **uang/e-money pribadi supir** |
| Reimburse | – | Transaksi uang pribadi → "perlu diganti" sampai admin membayar (rekap per supir) |
| Anggaran tol SPD | – | Admin mengisi "Anggaran Tol" per booking (saldo e-money kantor yang dibawa); tampil anggaran vs terpakai |

Mengaktifkan mode lengkap (admin, tanpa UI):
`PUT /api/v1/master-settings/trip_expense_full_mode {"value": 1}`.
Data mode sederhana tetap terbaca di mode lengkap (dan sebaliknya) — kolom
yang tidak dipakai mode aktif hanya tidak ditampilkan.

## 2. Aturan bisnis

- Dicatat untuk booking **kendaraan** yang berjalan: status `ONGOING` / `OVERDUE`
  oleh **supir yang ditugaskan**; **admin** juga boleh saat `RETURNED` /
  `COMPLETED` (input susulan / koreksi).
- Catatan tidak dihapus — admin **membatalkan** dengan alasan (pola sama dengan
  BBM). Catatan yang sudah direimburse tidak bisa dibatalkan.
- Biaya perjalanan masuk ke **total biaya booking** (BBM + biaya perjalanan) dan
  ke **laporan biaya** (ringkasan, per kendaraan, per departemen, tren, Excel).
- SPD (mode lengkap): biaya tol dibayar dari saldo e-money kantor (= anggaran).
  Kekurangan di jalan dibayar supir dengan uang/e-money pribadi → transaksi
  bertanda pribadi → masuk rekap **Reimburse Supir**. Admin memilih transaksi
  lalu menandai "sudah dibayar" (tanggal + catatan) — tercatat sebagai satu
  pembayaran reimburse.

## 3. Data

`trip_expenses` — satu baris per transaksi:
`bookingId`, `vehicleId`, `driverId`, `category` (TOLL/PARKING/OTHER),
`tollPoint` (ENTRY/EXIT, mode sederhana), `entryGate`, `exitGate`,
`amount`, `paymentMethod` (COMPANY_EMONEY/PERSONAL, mode lengkap),
`proofPhotoUrl`, `note`, `occurredAt`, `recordedById`, `reimbursementId`,
`voidedAt/voidedById/voidReason`, `createdAt`.

`trip_reimbursements` — pembayaran reimburse ke supir: `driverId`, `amount`,
`paidAt`, `note`, `createdById`.

`bookings."tollBudget"` — anggaran tol (mode lengkap, admin).

## 4. API

| Method | Path | Siapa |
|---|---|---|
| GET | `/trip-expenses/config` | semua login → `{mode}` |
| GET | `/bookings/:id/trip-expenses` | yang boleh melihat booking → `{mode, tollBudget, items, summary}` |
| POST | `/bookings/:id/trip-expenses` (multipart, `proofPhoto`) | supir pemegang / admin |
| PATCH | `/trip-expenses/:id/void` `{reason}` | admin |
| GET | `/trip-expenses/gates?q=` | semua login — saran nama gerbang |
| PATCH | `/bookings/:id/toll-budget` `{amount}` | admin |
| GET | `/trip-reimbursements/pending` | admin — total & rincian per supir |
| POST | `/trip-reimbursements` `{driverId, expenseIds[], paidAt?, note?}` | admin |
| GET | `/trip-reimbursements` | admin — riwayat pembayaran |

Semua request tulis menyiarkan `DATA_CHANGED` topik `booking`.

## 5. Tampilan

- **Detail booking → Catatan Perjalanan**: tab baru "Tol" (sederhana) /
  "Biaya Perjalanan" (lengkap): daftar transaksi + foto, total, tombol
  **Catat Tol** (supir/admin), batalkan (admin). Mode lengkap: kartu anggaran
  vs terpakai, badge cara bayar & status reimburse.
- **Web, mode lengkap saja**: halaman **Reimburse Supir** (`/trip-reimbursements`).
- **Laporan**: komponen biaya "Tol & Parkir" di ringkasan biaya, per kendaraan,
  per departemen, tren, dan total biaya overview (web & mobile).
