# Rancangan — Maintenance oleh Vendor/Bengkel Luar

> Status: **DISETUJUI (2026-10-01)** — dikerjakan di branch `feat/maintenance-vendor`.
> Dibuat 2026-09-30 dari diskusi dengan pemilik produk.

## 1. Latar belakang

Maintenance kendaraan dikerjakan **pihak luar**. Sebagian kendaraan milik perusahaan
(dikerjakan bengkel rekanan), sebagian **milik vendor/sewa** (dikerjakan/diurus vendor
pemiliknya). Perusahaan **mengajukan** maintenance dengan **surat** yang dicetak dari
sistem. Fitur sekarang (admin mencatat maintenance manual, status `pending`/`completed`)
belum mengenal vendor, surat, serah terima, maupun dokumen.

## 2. Keputusan pemilik produk (2026-09-30)

| Topik | Keputusan |
|---|---|
| Contoh surat | Belum ada → pakai format usulan §6 |
| Persetujuan atasan | **Tidak perlu** — surat langsung dibuat (generate) saat diajukan |
| Pengiriman ke vendor | **Cetak & unduh** saja (tanpa email dari sistem) |
| Biaya | **Bebas/fleksibel** (boleh kosong, boleh diisi kapan saja); **invoice** vendor sebagai file tambahan |
| Laporan kerusakan oleh supir | **Boleh**, terutama untuk masalah di jalan |
| Berita acara serah terima | **Dipakai, ringan** — dicetak dari data serah terima & pengembalian |
| Kop surat & penandatangan | **Diisi admin di Pengaturan** (nama perusahaan, alamat, telepon, logo, nama & jabatan penandatangan) |
| Nomor surat, checklist, level BBM | Sesuai usulan §6 (disetujui) |
| Layout PDF | Dibuat bebas oleh tim pengembang (belum ada format baku) |

## 3. Data baru

### 3.1 Vendor/Bengkel (master, kelola di web)
`vendors`: nama, jenis (`OWNER` pemilik kendaraan sewa · `WORKSHOP` bengkel rekanan · `BOTH`),
alamat, PIC, telepon, email, catatan, aktif/nonaktif. Hapus → nonaktifkan bila sudah dipakai.

### 3.2 Kepemilikan kendaraan (kolom baru `vehicles`)
- `ownership`: `COMPANY` (default, semua kendaraan lama) · `VENDOR`
- `ownerVendorId` (wajib bila `VENDOR`), `rentalContractNo` (opsional, tampil di surat)

### 3.3 Maintenance (perluasan `maintenance_records`)
| Kolom | Keterangan |
|---|---|
| `requestNo` | Nomor surat otomatis, unik (§6.1); terisi saat DIAJUKAN |
| `vendorId` | Vendor tujuan — otomatis = `ownerVendorId` untuk kendaraan `VENDOR`, dipilih admin untuk `COMPANY` |
| `category` | Servis berkala · Perbaikan · Ganti part · Body/cat · Lainnya |
| `complaint` / `description` | Keluhan & uraian pekerjaan yang diminta |
| `plannedDate`, `estimatedDays` | Rencana dari pengajuan |
| `scheduledDate` | Jadwal final dari konfirmasi vendor |
| `pickupMethod` | Diantar ke vendor · Dijemput vendor |
| `estimatedCost`, `actualCost` | Opsional, bebas diisi |
| `costBearer` | Perusahaan · Vendor · Belum ditentukan (opsional) |
| `status` | §4 |
| `sourceIssueId` | Laporan kerusakan asal (§3.5), bila ada |
| data serah terima | `handoverAt`, `handoverOdometer`, `handoverFuelLevel`, `handoverReceiverName`, `handoverChecklist` (JSON), `handoverNote` |
| data pengembalian | `returnedAt`, `returnOdometer`, `returnFuelLevel`, `returnHandlerName`, `returnChecklist` (JSON), `workDone`, `partsReplaced`, `returnNote` |

### 3.4 Dokumen maintenance
`maintenance_documents`: `maintenanceId`, `kind` (`INVOICE` · `SIGNED_REQUEST` scan surat bertanda tangan ·
`SIGNED_HANDOVER` · `SIGNED_RETURN` · `PHOTO` · `OTHER`), file, pengunggah, waktu.
PDF buatan sistem **tidak disimpan** — selalu dibuat ulang dari data (isi selalu mutakhir).

### 3.5 Laporan kerusakan dari supir
`vehicle_issue_reports`: kendaraan, booking (bila sedang trip), pelapor, waktu, lokasi, uraian,
foto, **masih bisa jalan?** (ya/tidak), status (`OPEN` · `CONVERTED` jadi pengajuan · `DISMISSED`).

## 4. Status & alur

```
                ┌───────── laporan kerusakan supir (opsional) ─────────┐
                ▼                                                        │
DRAFT ──ajukan──► DIAJUKAN ──vendor konfirmasi jadwal──► DIJADWALKAN ──serah terima──► DIKERJAKAN ──kendaraan kembali──► SELESAI
  │                 │  (nomor surat + PDF dibuat)         │                                │
  └──── DIBATALKAN ◄┴─────────────────────────────────────┘                                └─ (tidak bisa dibatalkan; selesaikan)
```

| # | Langkah | Oleh | Wajib diisi | Efek |
|---|---|---|---|---|
| 1 | Buat **DRAFT** | Admin (web/HP), atau dari laporan kerusakan | kendaraan, kategori, uraian | — |
| 2 | **Ajukan** → DIAJUKAN | Admin | vendor, rencana tanggal | Nomor surat dibuat; **PDF Surat Pengajuan** bisa dicetak/diunduh |
| 3 | **Jadwalkan** → DIJADWALKAN | Admin (mencatat jawaban vendor) | tanggal jadwal final | Tanggal itu **terblokir untuk booking baru**; admin diberi peringatan bila ada booking yang sudah disetujui bentrok (seperti MT-03). Kendaraan tetap bisa dipakai sampai diserahkan |
| 4 | **Serah terima** → DIKERJAKAN | Admin / supir yang mengantar | waktu, odometer, nama penerima di vendor, checklist | Kendaraan **MAINTENANCE** (hilang dari pilihan booking); **PDF Berita Acara Serah Terima** tersedia |
| 5 | **Terima kembali** → SELESAI | Admin | waktu kembali, odometer, pekerjaan yang dilakukan | Kendaraan **AVAILABLE**; odometer kendaraan diperbarui; **PDF Berita Acara Pengembalian** tersedia; biaya & invoice boleh menyusul |
| — | **Batalkan** | Admin | alasan | Hanya sebelum DIKERJAKAN; blokir tanggal dilepas |

Biaya aktual, penanggung biaya, dan invoice **boleh diisi/diunggah kapan saja** (juga setelah SELESAI).

### 4.1 Status kendaraan (menyambung aturan B10–B12 yang sudah jalan)
- Kendaraan **MAINTENANCE** selama ada maintenance berstatus **DIKERJAKAN** — dasar dari serah terima nyata, **bukan tanggal**. Jadi tidak terkunci hanya karena jadwal, dan tidak bebas sebelum benar-benar kembali.
- Trip berjalan tetap menang (IN_USE); INACTIVE tetap dipertahankan (logika `decideResourceStatus`).
- Blokir **tanggal** booking memakai `scheduledDate` (atau `plannedDate` saat DIAJUKAN) + `estimatedDays`; selama DIKERJAKAN blokir terbuka sampai SELESAI (sama seperti maintenance tanpa tanggal selesai sekarang).

### 4.2 Laporan kerusakan di jalan (supir, dari HP)
1. Supir yang sedang trip (atau memegang kendaraan) menekan **"Laporkan Kendala"**: uraian, foto, lokasi, *masih bisa jalan?*
2. Semua admin dapat **notifikasi** (prioritas tinggi bila *tidak bisa jalan*).
3. Admin memilih: **Buat pengajuan maintenance** dari laporan (data terisi otomatis) atau **Abaikan** dengan catatan.
4. Bila kendaraan tidak bisa jalan saat trip: admin menyelesaikan/menangani trip-nya secara manual (mis. menugaskan kendaraan lain untuk booking berikutnya). _Pengganti kendaraan di tengah trip berjalan belum didukung — kandidat pengembangan terpisah._

## 5. Hak akses

| Aksi | ADMIN | DRIVER | EMPLOYEE / ROOM_KEEPER |
|---|:--:|:--:|:--:|
| Master vendor, kepemilikan kendaraan | ✅ web | — | — |
| Buat/ajukan/jadwalkan/serah terima/terima kembali/batalkan | ✅ web & HP | — | — |
| Cetak/unduh PDF, unggah dokumen | ✅ | — | — |
| Laporan kerusakan | ✅ (lihat & tindak lanjut) | ✅ buat (HP) | — |

Mobile tetap "utuh" untuk maintenance (CLAUDE.md mobile §0.8), kecuali master vendor & kepemilikan kendaraan (web saja).

## 6. Dokumen PDF (dibuat backend, diunduh web & HP)

Dibuat di backend supaya web & HP menghasilkan file yang sama dan nomor surat tidak bentrok.
Data kop surat (nama perusahaan, alamat, logo, telepon) dan **nama + jabatan penandatangan**
disimpan di **Pengaturan** (master settings), bisa diubah tanpa deploy.

### 6.1 Nomor surat
`{urut 3 digit}/KCE-MNT/{bulan romawi}/{tahun}`, mis. `012/KCE-MNT/IX/2026`; urutan direset tiap tahun
(format bisa diatur di Pengaturan).

### 6.2 Surat Pengajuan Maintenance
Kop · nomor, tanggal, perihal "Permohonan Maintenance Kendaraan" · kepada: vendor, alamat, PIC ·
pembuka · **data kendaraan** (no. polisi, merk/model, tahun, odometer, kepemilikan + no. kontrak sewa) ·
**kategori & uraian pekerjaan / keluhan** · rencana jadwal, estimasi lama, cara serah (antar/jemput) ·
estimasi biaya & penanggung (bila diisi) · kontak penanggung jawab · penutup · tanda tangan
(nama & jabatan dari Pengaturan).

### 6.3 Berita Acara Serah Terima (ke vendor) & Berita Acara Pengembalian
Satu template, dua jenis. Isi: nomor (turunan nomor surat, mis. `012/KCE-MNT/IX/2026-BAST`),
tanggal & jam, data kendaraan, odometer, level BBM, **checklist kelengkapan**
(STNK, kunci utama, kunci cadangan, ban serep, dongkrak & kunci roda, segitiga pengaman, P3K, lainnya),
catatan kondisi + foto (opsional, dicetak kecil), untuk pengembalian: pekerjaan yang dilakukan &
part yang diganti · kolom tanda tangan **pihak perusahaan** dan **pihak vendor** (nama diisi dari data).
Scan yang sudah ditandatangani boleh diunggah sebagai dokumen (§3.4).

## 7. Sinkronisasi & notifikasi
- Semua endpoint tulis memakai topik DATA_CHANGED `maintenance` (+ `vehicle` saat status kendaraan berubah; + `booking` saat ada peringatan bentrok). Master vendor: topik baru `vendor`.
- Notifikasi: laporan kerusakan baru → admin; jadwal maintenance bentrok dengan booking disetujui → admin; kendaraan diserahkan/kembali → supir yang memegang kendaraan (bila ada).

## 8. Migrasi data lama
- Kendaraan lama → `ownership = COMPANY`.
- `vendorName` lama → dibuat vendor `WORKSHOP` otomatis (per nama unik), lalu ditautkan.
- Status lama: belum selesai (`pending`/`ongoing`) → **DIKERJAKAN**; `completed` → **SELESAI**.
  Record lama tidak punya nomor surat (PDF surat tidak tersedia; data tetap terbaca).
- Migrasi satu kali (penanda `data_fixes`), idempoten seperti migrasi sebelumnya.

## 9. Tahapan pengerjaan
1. **Fondasi** — master vendor, kepemilikan kendaraan, perluasan tabel + migrasi, status baru & aturan status kendaraan, API, dokumen/unggah invoice.
2. **PDF** — Pengaturan kop & penandatangan, nomor surat, Surat Pengajuan, Berita Acara Serah Terima/Pengembalian.
3. **UI web** — master vendor, form kendaraan (kepemilikan), daftar & detail maintenance per status, tombol cetak/unduh.
4. **UI mobile** — alur maintenance baru, unduh/bagikan PDF, **Laporkan Kendala** untuk supir.
5. **Laporan** — biaya per kendaraan/vendor/kategori, riwayat maintenance, laporan kerusakan.
6. **Testing** — skenario MT diperbarui di `SKENARIO_TESTING.md` + runner.

## 10. Pertanyaan tersisa
Tidak ada — semua terjawab 2026-10-01 (lihat §2).
