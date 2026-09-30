# Runner skenario testing (API)

Menjalankan skenario di [`docs/SKENARIO_TESTING.md`](../../docs/SKENARIO_TESTING.md) secara otomatis
terhadap backend **lokal**. Hasil terakhir: [`docs/HASIL_TESTING_2026-09-30.md`](../../docs/HASIL_TESTING_2026-09-30.md).

> ⚠️ Hanya untuk database & backend **test lokal**. Runner menolak alamat selain `localhost`/`127.0.0.1`,
> dan mengubah data langsung di database (menonaktifkan supir lama, menggeser tanggal booking).

## Persiapan (sekali)
1. PostgreSQL lokal, buat database `kce_test`, lalu jalankan semua `sql/schema/*.up.sql` berurutan.
2. Jalankan backend dari folder **di luar repo** (supaya `.env` repo tidak terbaca) dengan env:
   ```
   APP_PORT=8090
   DATABASE_URL=postgres://postgres:<password>@localhost:5432/kce_test?sslmode=disable
   JWT_SECRET=<bebas>
   SMTP_HOST=127.0.0.1  SMTP_PORT=1        # matikan email
   FIREBASE_CREDENTIALS_FILE=<file-yang-tidak-ada>   # matikan push
   UPLOAD_DIR=<folder sementara>
   ```

## Menjalankan
```bash
KCE_PGPASS=<password postgres> PSQL_PATH="/c/Program Files/PostgreSQL/18/bin/psql.exe" node run.mjs        # semua batch
KCE_PGPASS=<password postgres> PSQL_PATH="..." node run.mjs 2 3                                           # batch tertentu
```
Batch: `1` AU/RL/BC/SP · `2` AP/SB/AS/CN · `3` ST/CP/TO/MG/RR/RT · `4` MT/VH/RM/DU/FL/SY/TZ/DL.
Hasil per run ditulis ke `results-<run>.json` (tidak di-commit).

Admin pertama dibuat langsung di database (pendaftaran publik sudah ditutup) memakai hash dari
`go run generate_password.go` di root repo — jadi `go` harus ada di PATH.

Setiap akun uji memakai email `@kce-test.local` dan password acak per run (tidak disimpan).
Setiap skenario membuat kendaraan/supir/ruangan sendiri agar tidak saling mengganggu.
