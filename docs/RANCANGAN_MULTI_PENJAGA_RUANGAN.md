# Banyak Penjaga per Ruangan

Status: diterapkan 2026-11-12 (migrasi `000024_room_keepers_multi`).

## Keputusan pemilik produk

| Topik | Keputusan |
|---|---|
| Relasi | Satu ruangan bisa dijaga beberapa room keeper; satu room keeper bisa menjaga beberapa ruangan (N:N). |
| Daftar booking | Room keeper hanya melihat booking ruangan yang ia jaga. Semua penjaga ruangan yang sama melihat booking yang sama. Kalender ketersediaan (`?resourceId=`) tetap menampilkan semua jadwal. |
| Notifikasi | Sama seperti supir, tapi versi ruangan: semua penjaga menerima notifikasi yang sama. |
| Ruangan tanpa penjaga | Tidak bisa dibooking. Admin wajib memilih minimal satu penjaga aktif. |
| Rating | Per penjaga (bintang + ulasan masing-masing), tidak wajib. Penjaga yang dilewati bisa dinilai belakangan. |

## Data

- Tabel `room_keeper_rooms ("roomId", "roomKeeperId")`, PK gabungan, cascade dari `rooms` dan `room_keepers`.
- `rooms."roomKeeperId"` tidak dipakai lagi. Migrasi memindahkan isinya ke tabel baru lalu mengosongkannya, supaya CI yang menjalankan ulang semua migrasi tidak memasukkan penjaga lama kembali.
- `room_ratings`: UNIQUE `("bookingId")` diganti UNIQUE `("bookingId", "roomKeeperId")`, jadi satu booking punya satu rating per penjaga.
- "Penjaga aktif" berarti `room_keepers."isActive"` dan `users."isActive"` sama-sama true.

## Aturan

- **Boleh dibooking** (`GET /rooms?bookable=true`, `POST /bookings`, approve, substitute ke ruangan, `POST /guest-bookings`): status ruangan bukan INACTIVE/MAINTENANCE dan ada minimal satu penjaga aktif. Kalau tidak, API mengembalikan 409.
- **Mulai/selesai oleh room keeper dan ubah status ruangan:** hanya penjaga ruangan tersebut.
- **Mengubah penjaga ruangan** tidak mengubah booking yang sudah ada. Booking otomatis mengikuti penjaga ruangan saat ini, untuk daftar, notifikasi, maupun rating.

## Notifikasi ke penjaga aktif ruangan

| Kejadian | Tipe | Catatan |
|---|---|---|
| Booking dibuat | `NEW_BOOKING` | menunggu persetujuan |
| Disetujui | `ROOM_BOOKED` | dulu dikirim ke SEMUA room keeper |
| Ditolak | `BOOKING_REJECTED` | |
| Dibatalkan / hangus | `BOOKING_CANCELLED` | |
| Dimulai / selesai | `BOOKING_STARTED` / `BOOKING_COMPLETED` | pelaku aksi tidak dikirimi |
| Ruangan diganti | `BOOKING_SUBSTITUTED` ke penjaga lama, `NEW_BOOKING` ke penjaga baru | |
| Dinilai pemesan | `ROOM_KEEPER_RATED` | hanya penjaga yang dinilai |

## API

| Endpoint | Keterangan |
|---|---|
| `PATCH /rooms/:id/room-keepers {roomKeeperIds: []}` | Ganti seluruh penjaga (admin). Bentuk lama `PATCH /rooms/:id/room-keeper {roomKeeperId}` tetap diterima. |
| `GET /rooms`, `GET /rooms/:id` | + `roomKeepers[] {id,userId,name,phoneNumber,isActive}`, `hasActiveKeeper`. `roomKeeper` (penjaga pertama) tetap ada untuk aplikasi versi lama. |
| `GET /bookings/:id` | Booking ruangan: + `roomKeepers[]`. |
| `POST /bookings/:id/rate-room {ratings: [{roomKeeperId, rating, review?}]}` | Bentuk lama `{rating, review}` tetap diterima: nilainya dipakai untuk semua penjaga yang belum dinilai. |
| `GET /bookings/:id/room-ratings` | `{keepers: [{roomKeeperId, name, rating\|null, review, ratedAt}], ratedCount, total}`. `GET /:id/room-rating` (rating pertama) tetap ada untuk aplikasi lama. |

## Transisi

- Sejak migrasi berjalan sampai binary baru aktif, API lama membaca `rooms."roomKeeperId"` yang sudah kosong. Selama jeda deploy itu, room keeper tidak bisa memulai atau menyelesaikan booking ruangan.
- Ruangan yang belum punya penjaga langsung tidak bisa dibooking setelah deploy. Admin perlu memilihkan penjaganya di menu Ruangan.
