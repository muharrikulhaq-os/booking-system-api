# Riwayat Booking & Ekspor Excel (admin)

Status: diterapkan 2026-11-17. Hanya di web, menu Admin → Riwayat Booking (`/booking-history`).

## Keputusan pemilik produk

| Topik | Keputusan |
|---|---|
| Platform | Web saja (mobile tetap versi ringkas). |
| Cakupan | Booking kendaraan + ruangan. |
| Filter | Periode (wajib, default bulan berjalan, maks. 1 tahun), jenis resource, tipe SPD/Non-SPD (khusus kendaraan), beberapa status sekaligus, kata kunci. Semua filter bisa digabung. |
| Excel | Banyak sheet, dihubungkan lewat No. Booking. Foto & detail booking berupa link yang bisa diklik. |
| Bagian | Admin memilih bagian yang disertakan. Default semua. |

## API

- `GET /bookings/history?from&to&resourceType&bookingType&status=A,B&search&page&limit` mengembalikan daftar berpaginasi, urut jadwal terbaru.
- `GET /bookings/history/export?…filter sama…&sections=a,b` mengembalikan file `.xlsx` bernama `Riwayat Booking <from> sd <to>.xlsx`.
  - `sections` kosong berarti semua bagian.
  - Maksimal 20.000 booking per ekspor.
- `from`/`to` berformat `YYYY-MM-DD`, dibaca sebagai hari WIB, dan tanggal akhir ikut dihitung. Filter berlaku pada jadwal mulai booking.
- Tipe SPD/Non-SPD hanya mencocokkan booking kendaraan.

## Isi Excel

| Sheet | Isi |
|---|---|
| Info | Filter yang dipakai, waktu & pengekspor, jumlah per status. Selalu ada. |
| Booking | 1 baris per booking: status, resource, pemohon, jadwal, persetujuan, supir/penjaga, mulai/kembali/selesai, overtime, anggaran tol, bekal parkir, link detail. Selalu ada. |
| `passengers`: Penumpang | Karyawan & tamu, nomor HP. |
| `departure`: Keberangkatan | Dimulai pada/oleh, odometer awal, lokasi, link foto. |
| `fuel`: Pengisian BBM | Jumlah, biaya, odometer, SPBU, link foto struk, status batal. |
| `return`: Laporan Pengembalian | Waktu, pengirim, odometer akhir, lokasi, catatan, jumlah foto. |
| `expenses`: Biaya Tol & Parkir | Transaksi tol/parkir/lainnya, link foto bukti, status batal. |
| `ratings`: Rating | Rating supir & penjaga ruangan. |
| `activity`: Riwayat Aktivitas | Seluruh jejak `audit_logs` booking. |
| `attachments`: Lampiran & Foto | Semua lampiran booking + foto laporan pengembalian (1 link per baris). |

- Jam ditulis dalam WIB.
- Setiap sheet tabel punya header yang dibekukan dan filter otomatis.

## Link di Excel

- **Foto** = `PUBLIC_BASE_URL` + `/uploads/...`. Kalau `PUBLIC_BASE_URL` kosong, alamatnya diambil dari header `X-Forwarded-Proto/Host` atau `Host` request. Isi `PUBLIC_BASE_URL` di `.env` server (mis. `https://reservasi.kce.co.id`) supaya link tidak mengarah ke alamat internal.
- **Detail booking** = origin pertama di `FRONTEND_ORIGIN` + `/booking/<id>`.
