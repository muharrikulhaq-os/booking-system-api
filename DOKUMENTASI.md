# Dokumentasi API — Resource Booking System

> **Stack:** Go 1.23 · Fiber v2 · SQLC · PostgreSQL  
> **Base URL:** `http://localhost:{APP_PORT}/api/v1`  
> **Versi Schema:** v2 (2026-02-24)

---

## Daftar Isi

1. [Gambaran Umum](#1-gambaran-umum)
2. [Arsitektur Sistem](#2-arsitektur-sistem)
3. [Struktur Direktori](#3-struktur-direktori)
4. [Role & Hak Akses](#4-role--hak-akses)
5. [Model Data (ERD Singkat)](#5-model-data-erd-singkat)
6. [Autentikasi & Alur Token](#6-autentikasi--alur-token)
7. [Alur Booking Utama](#7-alur-booking-utama)
8. [Alur Guest Booking](#8-alur-guest-booking)
9. [Referensi Endpoint](#9-referensi-endpoint)
10. [Format Respons Standar](#10-format-respons-standar)
11. [Enum & Konstanta](#11-enum--konstanta)

---

## 1. Gambaran Umum

Sistem ini adalah REST API untuk pemesanan (booking) **kendaraan** dan **ruang rapat** di lingkungan perusahaan. Fitur utama:

| Fitur | Keterangan |
|---|---|
| Manajemen Kendaraan | CRUD kendaraan + kategori + foto |
| Manajemen Ruangan | CRUD ruang rapat + foto |
| Booking Resource | Karyawan memesan kendaraan atau ruangan |
| Approval Flow | Admin menyetujui / menolak booking |
| Assign Driver & Kendaraan | Admin menentukan driver dan kendaraan spesifik setelah approve |
| Guest Booking | Tamu eksternal bisa booking tanpa akun via token unik |
| Pengelolaan Driver | CRUD driver + riwayat penugasan ke kendaraan |
| Rating Driver | Pengguna memberi rating 1–5 setelah booking selesai |
| Fuel Expense | Driver mencatat pengeluaran BBM atau listrik (EV) |
| Maintenance | Admin mencatat servis / perawatan resource |
| Laporan | Ringkasan booking, utilisasi, BBM, maintenance, rating |
| Audit Log | Setiap aksi penting tercatat otomatis |

---

## 2. Arsitektur Sistem

```
┌──────────────┐     HTTP/JSON      ┌────────────────────────────────────────────┐
│   Client     │ ─────────────────► │              Go Fiber App                  │
│ (Web / App)  │                    │                                            │
└──────────────┘                    │  ┌──────────┐  ┌──────────┐  ┌─────────┐  │
                                    │  │ Handler  │  │ Service  │  │  Repo   │  │
                                    │  │(delivery)│─►│(business)│─►│ (SQLC)  │  │
                                    │  └──────────┘  └──────────┘  └────┬────┘  │
                                    │        ▲                           │       │
                                    │  ┌─────┴──────┐                   │       │
                                    │  │ Middleware  │                   ▼       │
                                    │  │ Auth / RBAC │          ┌──────────────┐ │
                                    │  └────────────┘          │  PostgreSQL  │ │
                                    └────────────────────────────────────────────┘
```

**Lapisan aplikasi:**

| Lapisan | Paket | Tanggung Jawab |
|---|---|---|
| **Delivery** | `internal/delivery/http` | Parse request, validasi input, panggil service, format respons |
| **Middleware** | `internal/middleware` | JWT auth, RBAC (role check), error handler |
| **Service** | `internal/service` | Business logic, aturan status, audit log |
| **Repository** | `internal/repository` | Akses database via SQLC (generated) |
| **Util** | `internal/util` | JWT helper, bcrypt, OTP, upload file, email, respons standar |
| **Config** | `internal/config` | Load env vars |

---

## 3. Struktur Direktori

```
booking-system-api/
├── cmd/api/
│   └── main.go                    # Entry point — inisialisasi app, route
├── internal/
│   ├── config/config.go           # Konfigurasi env
│   ├── delivery/http/
│   │   ├── auth_handler.go
│   │   ├── user_handler.go
│   │   ├── vehicle_handler.go
│   │   ├── room_handler.go
│   │   ├── driver_handler.go
│   │   ├── booking_handler.go
│   │   ├── remaining_handlers.go  # Fuel, Maintenance, Attachment, Guest, Setting, Report
│   │   └── helper.go              # queryInt, parseID, bindAndValidate
│   ├── middleware/
│   │   ├── auth.go                # JWT auth + RBAC
│   │   └── error_handler.go
│   ├── repository/                # Auto-generated oleh SQLC
│   │   ├── models.go
│   │   ├── querier.go
│   │   └── *.sql.go
│   ├── service/                   # Business logic per domain
│   │   └── *.go
│   └── util/
│       ├── jwt.go
│       ├── hash.go
│       ├── otp.go
│       ├── email.go
│       ├── upload.go
│       ├── errors.go
│       └── response.go
├── sql/
│   ├── schema/000001_init.up.sql  # Migrasi DDL
│   └── query/*.sql                # Query SQLC per domain
├── schema.sql                     # Schema lengkap + seed data
└── sqlc.yaml
```

---

## 4. Role & Hak Akses

Sistem memiliki tiga role:

| Role | Keterangan |
|---|---|
| `EMPLOYEE` | Karyawan biasa — bisa membuat booking, melihat booking milik sendiri |
| `ADMIN` | Dapat melakukan semua aksi termasuk approve, assign, kelola resource |
| `DRIVER` | Melihat booking yang di-assign kepadanya, start booking, input fuel expense |

**Ringkasan akses per domain:**

| Domain | EMPLOYEE | ADMIN | DRIVER |
|---|---|---|---|
| Auth (login, register, dll) | ✅ | ✅ | ✅ |
| Users — list/CRUD | ❌ | ✅ | ❌ |
| Users — profil sendiri | ✅ | ✅ | ✅ |
| Vehicles — list/detail | ✅ | ✅ | ✅ |
| Vehicles — CRUD + status | ❌ | ✅ | ❌ |
| Rooms — list/detail | ✅ | ✅ | ✅ |
| Rooms — CRUD + status | ❌ | ✅ | ❌ |
| Drivers — list/detail | ✅ | ✅ | ✅ |
| Drivers — CRUD + assign | ❌ | ✅ | ❌ |
| Booking — buat/cancel | ✅ | ✅ | ❌ |
| Booking — approve/reject/assign | ❌ | ✅ | ❌ |
| Booking — start | ❌ | ✅ | ✅ (booking milik sendiri) |
| Booking — complete | ❌ | ✅ | ❌ |
| Booking — rating driver | ✅ (booking milik sendiri) | ❌ | ❌ |
| Fuel Expense — create | ✅ | ✅ | ✅ |
| Fuel Expense — delete | ❌ | ✅ | ❌ |
| Maintenance — semua aksi | ❌ | ✅ | ❌ |
| Guest Booking — buat/cek (publik) | — | — | — |
| Guest Booking — approve/reject | ❌ | ✅ | ❌ |
| Master Settings — lihat | ✅ | ✅ | ✅ |
| Master Settings — ubah | ❌ | ✅ | ❌ |
| Laporan — semua | ❌ | ✅ | ❌ |

---

## 5. Model Data (ERD Singkat)

```
roles ──────────────────────────┐
departments ────────────────────┤
                                ▼
                             users ◄── refresh_tokens
                               │   ◄── password_reset_otps
                               │
                 ┌─────────────┤
                 ▼             ▼
              drivers        bookings ◄── approval_logs
                │               │
    driver_assignments     (resourceId)
    driver_ratings              │
                         ┌──────┴──────┐
                         ▼             ▼
                      vehicles       rooms
                         │
                  vehicle_categories
                         │
                    fuel_expenses
                    attachments ◄── (vehicleId / roomId / bookingId)

maintenance_records ── resources
guest_bookings ──────── resources
master_settings (standalone)
audit_logs (standalone)
```

**Tabel utama dan relasi kunci:**

| Tabel | Relasi Penting |
|---|---|
| `users` | → `roles`, → `departments` |
| `resources` | Parent abstrak dari `vehicles` dan `rooms` (1:1) |
| `bookings` | → `users`, → `resources`, optional → `drivers`, → `vehicles` |
| `drivers` | → `users` (1:1, user dengan role DRIVER) |
| `driver_assignments` | → `drivers`, → `vehicles` (riwayat penugasan permanen) |
| `driver_ratings` | → `bookings` (unique), → `drivers`, → `users` |
| `fuel_expenses` | → `drivers`, → `vehicles`, optional → `bookings` |
| `attachments` | → `users`, salah satu dari: → `vehicles` / `rooms` / `bookings` |
| `guest_bookings` | → `resources`, optional → `users` (approver) |
| `maintenance_records` | → `resources`, → `users` (createdBy) |

---

## 6. Autentikasi & Alur Token

Sistem menggunakan **JWT** dengan dua jenis token:

| Jenis | TTL | Kegunaan |
|---|---|---|
| `access` | Pendek (env) | Otorisasi request, dikirim di header `Authorization: Bearer <token>` |
| `refresh` | Panjang (env) | Memperbarui access token tanpa login ulang |
| `reset` | Pendek | Digunakan sekali untuk reset password |

### 6.1 Alur Login Normal

```
Client                          API
  │                              │
  │── POST /auth/login ─────────►│
  │   { email, password }        │ 1. Cek email & bcrypt password
  │                              │ 2. Cek isActive
  │                              │ 3. Buat access + refresh token
  │                              │ 4. Simpan refresh token ke DB
  │                              │ 5. Catat audit log LOGIN
  │◄── { accessToken,           │
  │      refreshToken,           │
  │      user: {...} } ──────────│
```

### 6.2 Alur Refresh Token

```
Client                          API
  │                              │
  │── POST /auth/refresh ───────►│
  │   { refreshToken }           │ 1. Parse & validasi JWT refresh
  │                              │ 2. Cek token di DB (tidak revoked, belum expired)
  │                              │ 3. Cek user masih aktif
  │                              │ 4. Buat access token baru
  │◄── { accessToken } ─────────│
```

### 6.3 Alur Reset Password

```
Client                          API
  │                              │
  │── POST /auth/forgot-password►│ 1. Cari user by email (silent jika tidak ada)
  │   { email }                  │ 2. Invalidate OTP lama
  │                              │ 3. Buat OTP 6-digit, simpan ke DB
  │                              │ 4. Kirim email OTP
  │◄── 200 OK (selalu) ─────────│
  │                              │
  │── POST /auth/verify-otp ───►│ 1. Cari user, cek OTP valid & belum dipakai
  │   { email, otpCode }         │ 2. Tandai OTP sebagai used
  │                              │ 3. Buat JWT reset token (type=reset)
  │◄── { resetToken } ──────────│
  │                              │
  │── POST /auth/reset-password►│ 1. Parse reset token (type=reset)
  │   { resetToken,              │ 2. Hash password baru
  │     newPassword }            │ 3. Update password di DB
  │◄── 200 OK ──────────────────│
```

---

## 7. Alur Booking Utama

### 7.1 Status Lifecycle Booking

```
                    ┌──────────────────┐
             buat   │                  │ cancel (EMPLOYEE/ADMIN)
    ─────────────►  │     PENDING      │ ─────────────────────► CANCELLED
                    │                  │
                    └──────────────────┘
                           │        │
               approve     │        │ reject
               (ADMIN)     │        │ (ADMIN)
                           ▼        ▼
                        APPROVED  REJECTED
                           │
               assign +    │
               start       │
               (ADMIN/DRIVER)
                           ▼
                        ONGOING
                           │           (endDate terlewat)
                           │──────────────────────────────► OVERDUE
                           │
               complete    │
               (ADMIN)     │
                           ▼
                       COMPLETED
                           │
               rate driver │  (EMPLOYEE — opsional)
                           ▼
                    driver_ratings
```

### 7.2 Langkah Detail Booking Kendaraan

```
[1] EMPLOYEE membuat booking
    POST /api/v1/bookings
    Body: { resourceId, startDate, endDate, purpose }

    Validasi server:
    - endDate > startDate
    - resource status = AVAILABLE
    - tidak ada booking konflик (PENDING/APPROVED/ONGOING) di periode yang sama

    Status: PENDING ✅

─────────────────────────────────────────────────────────

[2] ADMIN melihat daftar booking pending
    GET /api/v1/bookings?status=PENDING

─────────────────────────────────────────────────────────

[3a] ADMIN menyetujui
    POST /api/v1/bookings/:id/approve
    Body: { note? }

    Validasi server:
    - Admin tidak boleh approve booking miliknya sendiri
    - Status harus PENDING
    - Cek ulang konflik

    Status: APPROVED ✅
    Efek samping:
    - Buat approval_log (action=APPROVED)
    - Buat audit_log (action=APPROVE)
    - Kirim email notifikasi ke pemohon

[3b] ADMIN menolak
    POST /api/v1/bookings/:id/reject
    Body: { note }  ← wajib diisi

    Status: REJECTED ✅
    Efek samping:
    - Buat approval_log (action=REJECTED)
    - Kirim email notifikasi ke pemohon

─────────────────────────────────────────────────────────

[4] ADMIN assign driver & kendaraan (hanya booking VEHICLE)
    POST /api/v1/bookings/:id/assign-vehicle
    Body: { driverId, vehicleId }

    Validasi server:
    - Status harus APPROVED
    - Resource harus tipe VEHICLE
    - Driver harus aktif
    - Kendaraan tidak boleh sudah di-assign di booking lain pada periode yang sama

    Status: tetap APPROVED (assignedDriverId dan assignedVehicleId diisi)

─────────────────────────────────────────────────────────

[5] START perjalanan (ADMIN atau DRIVER yang di-assign)
    PATCH /api/v1/bookings/:id/start

    Validasi server:
    - Status harus APPROVED
    - Resource harus tipe VEHICLE
    - Waktu sekarang harus antara startDate dan endDate
    - Jika DRIVER: hanya driver yang di-assign yang bisa start

    Status: ONGOING ✅

─────────────────────────────────────────────────────────

[6] ADMIN menyelesaikan booking
    PATCH /api/v1/bookings/:id/complete

    Validasi server:
    - Status harus ONGOING atau OVERDUE

    Status: COMPLETED ✅

─────────────────────────────────────────────────────────

[7] EMPLOYEE memberi rating driver (opsional)
    POST /api/v1/bookings/:id/rate-driver
    Body: { rating (1-5), review? }

    Validasi server:
    - Booking harus COMPLETED
    - Hanya pemohon booking yang bisa rating
    - Resource harus tipe VEHICLE & driver di-assign
    - Setiap booking hanya bisa dirating sekali
```

### 7.3 Booking Ruangan (Simplified)

Booking ruangan mengikuti alur yang sama **kecuali**:
- Tidak ada step assign driver/kendaraan
- Tidak ada step Start
- Setelah APPROVED, admin langsung bisa Complete
- Tidak ada rating driver

---

## 8. Alur Guest Booking

Guest booking memungkinkan **tamu eksternal** (tanpa akun) untuk memesan resource.

```
[1] Tamu membuat booking (PUBLIC — tanpa auth)
    POST /api/v1/guest-bookings
    Body: { guestName, guestEmail, guestPhone, departmentName,
            resourceId, startDate, endDate, purpose }

    Server menghasilkan accessToken unik (64 char) dan menyimpannya.
    Status: PENDING ✅
    → accessToken dikirim ke tamu (via respons / email)

─────────────────────────────────────────────────────────

[2] Tamu cek status booking (PUBLIC)
    GET /api/v1/guest-bookings/:token

─────────────────────────────────────────────────────────

[3a] ADMIN menyetujui
     POST /api/v1/guest-bookings/:id/approve
     Status: APPROVED ✅

[3b] ADMIN menolak
     POST /api/v1/guest-bookings/:id/reject
     Body: { note }
     Status: REJECTED ✅

─────────────────────────────────────────────────────────

[4] ADMIN start (opsional, jika perlu)
    PATCH /api/v1/guest-bookings/:id/start
    Status: ONGOING ✅

─────────────────────────────────────────────────────────

[5a] Tamu menyelesaikan booking via token (PUBLIC)
     PATCH /api/v1/guest-bookings/:token/complete
     Status: COMPLETED ✅

[5b] Tamu membatalkan booking via token (PUBLIC)
     PATCH /api/v1/guest-bookings/:token/cancel
     Status: CANCELLED ✅
```

---

## 9. Referensi Endpoint

### 9.1 Auth — `/api/v1/auth`

| Method | Path | Auth | Keterangan |
|---|---|---|---|
| POST | `/register` | — | Daftar akun baru |
| POST | `/login` | — | Login, dapatkan token |
| POST | `/refresh` | — | Perbarui access token |
| POST | `/logout` | ✅ | Revoke refresh token |
| POST | `/forgot-password` | — | Kirim OTP ke email |
| POST | `/verify-otp` | — | Verifikasi OTP, dapat reset token |
| POST | `/reset-password` | — | Reset password dengan reset token |
| PATCH | `/change-password` | ✅ | Ganti password (user aktif) |
| GET | `/me` | ✅ | Profil user yang sedang login |

### 9.2 Users — `/api/v1/users`

| Method | Path | Auth | Role | Keterangan |
|---|---|---|---|---|
| GET | `` | ✅ | ADMIN | List user (filter: search, roleId) |
| GET | `/me` | ✅ | Semua | Profil sendiri |
| GET | `/roles` | ✅ | Semua | Daftar role |
| GET | `/departments` | ✅ | Semua | Daftar departemen |
| GET | `/:id` | ✅ | ADMIN | Detail user |
| POST | `` | ✅ | ADMIN | Buat user baru |
| PUT | `/:id` | ✅ | ADMIN | Update user |
| PATCH | `/:id/toggle-active` | ✅ | ADMIN | Aktifkan / nonaktifkan user |
| DELETE | `/:id` | ✅ | ADMIN | Hapus user |
| PUT | `/me/profile-photo` | ✅ | Semua | Upload foto profil sendiri |
| DELETE | `/me/profile-photo` | ✅ | Semua | Hapus foto profil sendiri |
| PUT | `/:id/profile-photo` | ✅ | ADMIN | Upload foto profil user lain |

### 9.3 Vehicles — `/api/v1/vehicles`

| Method | Path | Auth | Role | Keterangan |
|---|---|---|---|---|
| GET | `` | ✅ | Semua | List kendaraan (filter: search, categoryId, status) |
| GET | `/categories` | ✅ | Semua | Daftar kategori kendaraan |
| GET | `/:id` | ✅ | Semua | Detail kendaraan |
| POST | `` | ✅ | ADMIN | Tambah kendaraan baru |
| POST | `/categories` | ✅ | ADMIN | Tambah kategori baru |
| PUT | `/:id` | ✅ | ADMIN | Update data kendaraan |
| PATCH | `/:id/status` | ✅ | ADMIN | Ubah status kendaraan |
| PATCH | `/:id/photo` | ✅ | ADMIN | Upload foto kendaraan |
| DELETE | `/:id` | ✅ | ADMIN | Hapus kendaraan |
| DELETE | `/categories/:id` | ✅ | ADMIN | Hapus kategori |
| GET | `/:id/attachments` | ✅ | Semua | Daftar lampiran kendaraan |
| POST | `/:id/attachments` | ✅ | ADMIN | Upload lampiran kendaraan |

### 9.4 Rooms — `/api/v1/rooms`

| Method | Path | Auth | Role | Keterangan |
|---|---|---|---|---|
| GET | `` | ✅ | Semua | List ruangan (filter: search, status) |
| GET | `/:id` | ✅ | Semua | Detail ruangan |
| POST | `` | ✅ | ADMIN | Tambah ruangan |
| PUT | `/:id` | ✅ | ADMIN | Update ruangan |
| PATCH | `/:id/status` | ✅ | ADMIN | Ubah status ruangan |
| PATCH | `/:id/photo` | ✅ | ADMIN | Upload foto ruangan |
| DELETE | `/:id` | ✅ | ADMIN | Hapus ruangan |
| GET | `/:id/attachments` | ✅ | Semua | Daftar lampiran ruangan |
| POST | `/:id/attachments` | ✅ | ADMIN | Upload lampiran ruangan |

### 9.5 Drivers — `/api/v1/drivers`

| Method | Path | Auth | Role | Keterangan |
|---|---|---|---|---|
| GET | `` | ✅ | Semua | List driver |
| GET | `/:driver_id` | ✅ | Semua | Detail driver |
| POST | `` | ✅ | ADMIN | Tambah driver |
| PUT | `/:driver_id` | ✅ | ADMIN | Update driver |
| PATCH | `/:driver_id/toggle-active` | ✅ | ADMIN | Aktifkan / nonaktifkan driver |
| POST | `/:driver_id/assign` | ✅ | ADMIN | Assign driver ke kendaraan permanen |
| PATCH | `/:driver_id/release` | ✅ | ADMIN | Lepas penugasan driver dari kendaraan |
| GET | `/:driver_id/assignments` | ✅ | ADMIN | Riwayat penugasan driver |

### 9.6 Bookings — `/api/v1/bookings`

| Method | Path | Auth | Role | Keterangan |
|---|---|---|---|---|
| GET | `` | ✅ | Semua* | List booking (filter: status, userId, resourceId, driverId, tanggal) |
| GET | `/:id` | ✅ | Semua* | Detail booking |
| POST | `` | ✅ | EMPLOYEE/ADMIN | Buat booking baru |
| PATCH | `/:id/cancel` | ✅ | Semua* | Batalkan booking (hanya status PENDING) |
| POST | `/:id/approve` | ✅ | ADMIN | Setujui booking |
| POST | `/:id/reject` | ✅ | ADMIN | Tolak booking |
| POST | `/:id/assign-vehicle` | ✅ | ADMIN | Assign driver & kendaraan ke booking |
| PATCH | `/:id/start` | ✅ | ADMIN/DRIVER | Mulai perjalanan |
| PATCH | `/:id/complete` | ✅ | ADMIN | Selesaikan booking |
| POST | `/:id/rate-driver` | ✅ | EMPLOYEE | Beri rating driver |
| GET | `/drivers/:driver_id/ratings` | ✅ | ADMIN | Semua rating seorang driver |
| GET | `/:id/approval-log` | ✅ | ADMIN | Log approve/reject booking |
| GET | `/:id/attachments` | ✅ | Semua | Daftar lampiran booking |
| POST | `/:id/attachments` | ✅ | Semua | Upload lampiran booking |

> *EMPLOYEE hanya melihat booking milik sendiri; DRIVER hanya melihat booking yang di-assign kepadanya.

### 9.7 Fuel Expenses — `/api/v1/fuel-expenses`

| Method | Path | Auth | Role | Keterangan |
|---|---|---|---|---|
| GET | `` | ✅ | Semua | List pengeluaran (filter: driverId, vehicleId, fuelType) |
| GET | `/:id` | ✅ | Semua | Detail pengeluaran |
| POST | `/bbm` | ✅ | Semua | Catat pengisian BBM |
| POST | `/listrik` | ✅ | Semua | Catat pengisian listrik (EV/SPKLU) |
| DELETE | `/:id` | ✅ | ADMIN | Hapus catatan |

**Request POST `/bbm`:**
```json
{
  "vehicleId": 1,
  "bookingId": 6,
  "liter": 40.5,
  "pricePerLiter": 10000,
  "odometerBefore": 15000,
  "odometerAfter": 15400,
  "totalAmount": 405000,
  "note": "SPBU Pertamina Jl. Sudirman"
}
```

**Request POST `/listrik`:**
```json
{
  "vehicleId": 7,
  "bookingId": 9,
  "kwh": 45.0,
  "pricePerKwh": 2466,
  "batteryBefore": 20.0,
  "batteryAfter": 95.0,
  "totalAmount": 110970,
  "note": "SPKLU PLN Kemayoran"
}
```

### 9.8 Maintenance — `/api/v1/maintenance`

Seluruh endpoint memerlukan auth **ADMIN**.

| Method | Path | Keterangan |
|---|---|---|
| GET | `` | List catatan maintenance (filter: resourceId) |
| GET | `/:id` | Detail catatan |
| POST | `` | Buat catatan baru |
| PUT | `/:id` | Update catatan (termasuk tanggal selesai & biaya) |
| DELETE | `/:id` | Hapus catatan |

### 9.9 Attachments — `/api/v1/attachments`

| Method | Path | Auth | Keterangan |
|---|---|---|---|
| DELETE | `/:id` | ✅ | Hapus lampiran (pemilik atau ADMIN) |

> Upload lampiran dilakukan via endpoint resource masing-masing (vehicles, rooms, bookings).

### 9.10 Guest Bookings — `/api/v1/guest-bookings`

| Method | Path | Auth | Role | Keterangan |
|---|---|---|---|---|
| POST | `` | — | Publik | Buat guest booking |
| GET | `/:token` | — | Publik | Cek status booking via token |
| PATCH | `/:token/complete` | — | Publik | Tamu menyelesaikan booking |
| PATCH | `/:token/cancel` | — | Publik | Tamu membatalkan booking |
| GET | `` | ✅ | ADMIN | List semua guest booking |
| POST | `/:id/approve` | ✅ | ADMIN | Setujui guest booking |
| POST | `/:id/reject` | ✅ | ADMIN | Tolak guest booking |
| PATCH | `/:id/start` | ✅ | ADMIN | Mulai guest booking |

### 9.11 Master Settings — `/api/v1/master-settings`

| Method | Path | Auth | Role | Keterangan |
|---|---|---|---|---|
| GET | `` | ✅ | Semua | Daftar semua setting |
| GET | `/:key` | ✅ | Semua | Nilai setting by key |
| PUT | `/:key` | ✅ | ADMIN | Buat/update setting |

**Default keys:**
- `price_per_liter_bbm` — Harga bensin per liter (IDR/liter)
- `price_per_kwh_listrik` — Tarif listrik per kWh (IDR/kWh)

### 9.12 Reports — `/api/v1/reports`

Seluruh endpoint memerlukan auth **ADMIN**.

| Method | Path | Query Params | Keterangan |
|---|---|---|---|
| GET | `/bookings` | startDate, endDate | Ringkasan statistik booking |
| GET | `/resource-usage` | — | Utilisasi resource (view v_vehicle_summary) |
| GET | `/fuel-expenses` | — | Laporan BBM & listrik per kendaraan (view v_fuel_expense_summary) |
| GET | `/maintenance-cost` | — | Total biaya maintenance per resource |
| GET | `/driver-ratings` | — | Ringkasan rating driver (view v_driver_ratings_summary) |
| GET | `/driver-activity` | — | Aktivitas booking per driver |
| GET | `/overdue-bookings` | — | Booking yang melewati waktu pengembalian |
| GET | `/audit-logs` | page, limit, entityType, userId | Log semua aksi sistem |

### 9.13 File Server — `/files/*`

| Method | Path | Auth | Keterangan |
|---|---|---|---|
| GET | `/files/*` | ✅ | Serve file yang sudah diupload |

---

## 10. Format Respons Standar

### Sukses

```json
{
  "success": true,
  "message": "Booking created",
  "data": { ... }
}
```

### Sukses Paginated

```json
{
  "success": true,
  "message": "Bookings retrieved",
  "data": [ ... ],
  "meta": {
    "page": 1,
    "limit": 20,
    "total": 87
  }
}
```

### Error

```json
{
  "success": false,
  "message": "resource is not available"
}
```

### HTTP Status Code Umum

| Kode | Kondisi |
|---|---|
| 200 | Sukses |
| 201 | Resource berhasil dibuat |
| 400 | Bad request / validasi gagal |
| 401 | Token tidak ada / tidak valid |
| 403 | Tidak punya izin (role tidak sesuai / bukan milik sendiri) |
| 404 | Resource tidak ditemukan |
| 409 | Konflik (booking bentrok, duplikasi, dll.) |
| 503 | Database tidak dapat dijangkau (health check) |

---

## 11. Enum & Konstanta

### Booking Status

| Nilai | Keterangan |
|---|---|
| `PENDING` | Menunggu approval admin |
| `APPROVED` | Disetujui, belum dimulai |
| `REJECTED` | Ditolak admin |
| `ONGOING` | Sedang berlangsung |
| `COMPLETED` | Selesai |
| `CANCELLED` | Dibatalkan oleh pemohon atau admin |
| `OVERDUE` | Melewati endDate tanpa dikembalikan |

### Resource Status

| Nilai | Keterangan |
|---|---|
| `AVAILABLE` | Tersedia untuk dibooking |
| `MAINTENANCE` | Sedang dalam perawatan |
| `INACTIVE` | Tidak aktif / tidak digunakan |

### Resource Type

| Nilai | Keterangan |
|---|---|
| `VEHICLE` | Kendaraan |
| `ROOM` | Ruang rapat |

### Fuel Type

| Nilai | Keterangan |
|---|---|
| `BBM` | Bahan bakar minyak (bensin / solar) |
| `LISTRIK` | Kendaraan listrik (pengisian di SPKLU) |

### Role Name

| Nilai | Keterangan |
|---|---|
| `EMPLOYEE` | Karyawan biasa |
| `ADMIN` | Administrator |
| `DRIVER` | Pengemudi |

### Audit Log — Action

| Nilai | Dipicu oleh |
|---|---|
| `LOGIN` | User login |
| `LOGOUT` | User logout |
| `CREATE` | Membuat entity baru |
| `UPDATE` | Memperbarui entity |
| `DELETE` | Menghapus entity |
| `APPROVE` | Admin approve booking |
| `REJECT` | Admin reject booking |
| `ASSIGN` | Admin assign driver + kendaraan |
| `START` | Booking dimulai |
| `COMPLETE` | Booking diselesaikan |
| `RATE_DRIVER` | User memberi rating driver |
