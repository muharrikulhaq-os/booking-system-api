# 📘 API Documentation — Booking System API

> **Base URL:** `http://localhost:8080`  
> **API Prefix:** `/api/v1`  
> **Auth:** Bearer JWT Token via `Authorization: Bearer <access_token>`  
> **Content-Type:** `application/json` (kecuali upload file: `multipart/form-data`)

---

## 📐 Standar Response Format

### Success Response
```json
{
  "success": true,
  "message": "string",
  "data": {}
}
```

### Paginated Response
```json
{
  "success": true,
  "message": "string",
  "data": [],
  "pagination": {
    "total": 100,
    "page": 1,
    "limit": 20,
    "totalPages": 5
  }
}
```

### Error Response
```json
{
  "success": false,
  "message": "string",
  "error": {
    "code": "ERROR_CODE",
    "message": "detail pesan error"
  }
}
```

### HTTP Status Codes
| Code | Keterangan |
|------|-----------|
| `200` | OK — Request berhasil |
| `201` | Created — Data berhasil dibuat |
| `400` | Bad Request — Request tidak valid / file tidak ditemukan |
| `401` | Unauthorized — Token tidak valid atau expired |
| `403` | Forbidden — Akses ditolak (role tidak cukup) |
| `404` | Not Found — Data tidak ditemukan |
| `409` | Conflict — Duplikat data / jadwal bentrok |
| `422` | Unprocessable Entity — Validasi request gagal |
| `500` | Internal Server Error |

### Role yang Tersedia
| Role | Keterangan |
|------|-----------|
| `ADMIN` | Akses penuh ke semua fitur manajemen |
| `USER` | Pengguna internal, bisa membuat booking |
| `DRIVER` | Pengemudi, bisa mulai perjalanan & input BBM |

---

## 💓 Health Check

### `GET /health`
Mengecek status aplikasi dan koneksi database.

**Akses:** Public

**Response `200`:**
```json
{
  "success": true,
  "message": "OK",
  "data": {
    "status": "healthy",
    "db": "connected"
  }
}
```

---

## 🔐 Modul 1 — Autentikasi (`/api/v1/auth`)

### `POST /api/v1/auth/register`
Mendaftarkan akun pengguna baru.

**Akses:** Public

**Request Body:**
```json
{
  "employeeId": "EMP-001",
  "name": "Budi Santoso",
  "email": "budi@example.com",
  "password": "password123",
  "roleId": 2,
  "departmentId": 3
}
```

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `employeeId` | string | ✅ | ID karyawan unik |
| `name` | string | ✅ | Nama lengkap |
| `email` | string | ✅ | Format email valid |
| `password` | string | ✅ | Minimal 8 karakter |
| `roleId` | integer | ✅ | ID role (dari `/users/roles`) |
| `departmentId` | integer | ✅ | ID departemen (dari `/users/departments`) |

**Response `201`:**
```json
{
  "success": true,
  "message": "Registration successful",
  "data": {
    "id": 1,
    "employeeId": "EMP-001",
    "name": "Budi Santoso",
    "email": "budi@example.com",
    "role": { "id": 2, "name": "USER" },
    "department": { "id": 3, "name": "Operations" },
    "isActive": true,
    "createdAt": "2025-05-26T10:00:00Z"
  }
}
```

---

### `POST /api/v1/auth/login`
Login dan mendapatkan access + refresh token.

**Akses:** Public

**Request Body:**
```json
{
  "email": "budi@example.com",
  "password": "password123"
}
```

**Response `200`:**
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refreshToken": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4...",
    "tokenType": "Bearer",
    "user": {
      "id": 1,
      "employeeId": "EMP-001",
      "name": "Budi Santoso",
      "email": "budi@example.com",
      "role": "USER",
      "department": "Operations"
    }
  }
}
```

---

### `POST /api/v1/auth/refresh`
Memperbarui access token dengan refresh token.

**Akses:** Public

**Request Body:**
```json
{
  "refreshToken": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4..."
}
```

**Response `200`:**
```json
{
  "success": true,
  "message": "Token refreshed",
  "data": {
    "accessToken": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refreshToken": "bmV3UmVmcmVzaFRva2Vu...",
    "tokenType": "Bearer"
  }
}
```

---

### `POST /api/v1/auth/logout`
Logout dan revoke refresh token.

**Akses:** User (Auth required)

**Request Body:**
```json
{
  "refreshToken": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4..."
}
```

**Response `200`:**
```json
{
  "success": true,
  "message": "Logged out successfully",
  "data": null
}
```

---

### `POST /api/v1/auth/forgot-password`
Meminta OTP reset password ke email.

**Akses:** Public

**Request Body:**
```json
{
  "email": "budi@example.com"
}
```

**Response `200`:**
```json
{
  "success": true,
  "message": "If the email exists, an OTP has been sent.",
  "data": null
}
```

---

### `POST /api/v1/auth/verify-otp`
Verifikasi OTP dan dapatkan reset token.

**Akses:** Public

**Request Body:**
```json
{
  "email": "budi@example.com",
  "otpCode": "123456"
}
```

**Response `200`:**
```json
{
  "success": true,
  "message": "OTP verified",
  "data": {
    "resetToken": "abc123resettoken..."
  }
}
```

---

### `POST /api/v1/auth/reset-password`
Reset password menggunakan reset token dari OTP.

**Akses:** Public

**Request Body:**
```json
{
  "resetToken": "abc123resettoken...",
  "newPassword": "newpassword123"
}
```

**Response `200`:**
```json
{
  "success": true,
  "message": "Password reset successfully",
  "data": null
}
```

---

### `PATCH /api/v1/auth/change-password`
Ubah password dalam sesi aktif.

**Akses:** User (Auth required)

**Request Body:**
```json
{
  "currentPassword": "password123",
  "newPassword": "newpassword456"
}
```

**Response `200`:**
```json
{
  "success": true,
  "message": "Password changed",
  "data": null
}
```

---

### `GET /api/v1/auth/me`
Ambil profil pengguna yang sedang login.

**Akses:** User (Auth required)

**Response `200`:**
```json
{
  "success": true,
  "message": "Profile retrieved",
  "data": {
    "id": 1,
    "employeeId": "EMP-001",
    "name": "Budi Santoso",
    "email": "budi@example.com",
    "profilePhoto": "/files/photos/profile-1.jpg",
    "isActive": true,
    "role": { "id": 2, "name": "USER" },
    "department": { "id": 3, "name": "Operations" },
    "createdAt": "2025-05-26T10:00:00Z"
  }
}
```

---

## 👥 Modul 2 — Pengguna (`/api/v1/users`)

### `GET /api/v1/users`
Daftar semua pengguna sistem.

**Akses:** Admin

**Query Parameters:**
| Parameter | Type | Keterangan |
|-----------|------|-----------|
| `page` | integer | Halaman (default: 1) |
| `limit` | integer | Jumlah per halaman (default: 20) |
| `search` | string | Cari berdasarkan nama atau email |
| `roleId` | integer | Filter berdasarkan role |

**Response `200`:**
```json
{
  "success": true,
  "message": "Users retrieved",
  "data": [
    {
      "id": 1,
      "employeeId": "EMP-001",
      "name": "Budi Santoso",
      "email": "budi@example.com",
      "profilePhoto": null,
      "isActive": true,
      "role": { "id": 2, "name": "USER" },
      "department": { "id": 3, "name": "Operations" },
      "createdAt": "2025-05-26T10:00:00Z"
    }
  ],
  "pagination": {
    "total": 50,
    "page": 1,
    "limit": 20,
    "totalPages": 3
  }
}
```

---

### `GET /api/v1/users/me`
Profil diri sendiri.

**Akses:** User (Auth required)

**Response `200`:** *(sama seperti `/auth/me`)*

---

### `GET /api/v1/users/roles`
Daftar semua role (untuk dropdown).

**Akses:** User (Auth required)

**Response `200`:**
```json
{
  "success": true,
  "message": "Roles retrieved",
  "data": [
    { "id": 1, "name": "ADMIN" },
    { "id": 2, "name": "USER" },
    { "id": 3, "name": "DRIVER" }
  ]
}
```

---

### `GET /api/v1/users/departments`
Daftar semua departemen (untuk dropdown).

**Akses:** User (Auth required)

**Response `200`:**
```json
{
  "success": true,
  "message": "Departments retrieved",
  "data": [
    { "id": 1, "name": "HR" },
    { "id": 2, "name": "Finance" },
    { "id": 3, "name": "Operations" }
  ]
}
```

---

### `GET /api/v1/users/:id`
Detail pengguna berdasarkan ID.

**Akses:** Admin

**Response `200`:** *(sama seperti item dalam list users)*

---

### `POST /api/v1/users`
Buat pengguna baru (oleh Admin).

**Akses:** Admin

**Request Body:**
```json
{
  "employeeId": "EMP-002",
  "name": "Siti Rahma",
  "email": "siti@example.com",
  "password": "password123",
  "roleId": 2,
  "departmentId": 1
}
```

**Response `201`:** *(sama seperti data user)*

---

### `PUT /api/v1/users/:id`
Update informasi pengguna.

**Akses:** Admin

**Request Body:**
```json
{
  "name": "Siti Rahmawati",
  "email": "siti.baru@example.com",
  "roleId": 3,
  "departmentId": 2
}
```

**Response `200`:** *(data user yang diperbarui)*

---

### `PATCH /api/v1/users/:id/toggle-active`
Aktifkan atau nonaktifkan akun pengguna.

**Akses:** Admin

**Response `200`:**
```json
{
  "success": true,
  "message": "User status toggled",
  "data": {
    "id": 2,
    "isActive": false
  }
}
```

---

### `DELETE /api/v1/users/:id`
Hapus pengguna dari sistem.

**Akses:** Admin

**Response `200`:**
```json
{
  "success": true,
  "message": "User deleted",
  "data": null
}
```

---

### `PUT /api/v1/users/me/profile-photo`
Upload/update foto profil sendiri.

**Akses:** User (Auth required)

**Content-Type:** `multipart/form-data`

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `photo` | file | ✅ | File gambar (jpg/png) |

**Response `200`:**
```json
{
  "success": true,
  "message": "Profile photo updated",
  "data": {
    "profilePhoto": "/files/photos/profile-1.jpg"
  }
}
```

---

### `DELETE /api/v1/users/me/profile-photo`
Hapus foto profil sendiri.

**Akses:** User (Auth required)

**Response `200`:**
```json
{
  "success": true,
  "message": "Profile photo deleted",
  "data": null
}
```

---

### `PUT /api/v1/users/:id/profile-photo`
Update foto profil user lain (oleh Admin).

**Akses:** Admin

**Content-Type:** `multipart/form-data`

| Field | Type | Required |
|-------|------|----------|
| `photo` | file | ✅ |

**Response `200`:** *(sama seperti update foto profil sendiri)*

---

## 🚗 Modul 3 — Kendaraan (`/api/v1/vehicles`)

### `GET /api/v1/vehicles`
Daftar semua kendaraan.

**Akses:** User (Auth required)

**Query Parameters:**
| Parameter | Type | Keterangan |
|-----------|------|-----------|
| `page` | integer | Halaman (default: 1) |
| `limit` | integer | Jumlah per halaman (default: 20) |
| `search` | string | Cari berdasarkan nama/plat/merek |
| `categoryId` | integer | Filter berdasarkan kategori |
| `status` | string | `AVAILABLE` \| `MAINTENANCE` \| `INACTIVE` |

**Response `200`:**
```json
{
  "success": true,
  "message": "Vehicles retrieved",
  "data": [
    {
      "id": 1,
      "resourceId": 10,
      "name": "Toyota Avanza",
      "plateNumber": "B 1234 ABC",
      "brand": "Toyota",
      "model": "Avanza",
      "year": 2022,
      "currentOdometer": 15000,
      "capacity": 7,
      "category": { "id": 1, "name": "MPV" },
      "status": "AVAILABLE",
      "photoUrl": "/files/vehicles/avanza.jpg"
    }
  ],
  "pagination": { "total": 10, "page": 1, "limit": 20, "totalPages": 1 }
}
```

---

### `GET /api/v1/vehicles/categories`
Daftar kategori kendaraan.

**Akses:** User (Auth required)

**Response `200`:**
```json
{
  "success": true,
  "message": "Categories retrieved",
  "data": [
    { "id": 1, "name": "MPV" },
    { "id": 2, "name": "Sedan" },
    { "id": 3, "name": "Truck" }
  ]
}
```

---

### `GET /api/v1/vehicles/:id`
Detail kendaraan berdasarkan ID.

**Akses:** User (Auth required)

**Response `200`:**
```json
{
  "success": true,
  "message": "Vehicle retrieved",
  "data": {
    "id": 1,
    "resourceId": 10,
    "name": "Toyota Avanza",
    "plateNumber": "B 1234 ABC",
    "brand": "Toyota",
    "model": "Avanza",
    "year": 2022,
    "currentOdometer": 15000,
    "capacity": 7,
    "category": { "id": 1, "name": "MPV" },
    "status": "AVAILABLE",
    "photoUrl": "/files/vehicles/avanza.jpg"
  }
}
```

---

### `POST /api/v1/vehicles`
Tambah kendaraan baru.

**Akses:** Admin

**Request Body:**
```json
{
  "name": "Honda Civic",
  "plateNumber": "B 5678 XYZ",
  "brand": "Honda",
  "model": "Civic",
  "year": 2023,
  "currentOdometer": 0,
  "categoryId": 2,
  "capacity": 5
}
```

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `name` | string | ✅ | Nama kendaraan |
| `plateNumber` | string | ✅ | Nomor plat |
| `brand` | string | ✅ | Merek |
| `model` | string | ✅ | Model/tipe |
| `year` | integer | ✅ | Tahun produksi |
| `currentOdometer` | integer | ❌ | Odometer saat ini (km) |
| `categoryId` | integer | ✅ | ID kategori |
| `capacity` | integer | ✅ | Kapasitas penumpang (min: 1) |

**Response `201`:** *(data kendaraan)*

---

### `POST /api/v1/vehicles/categories`
Tambah kategori kendaraan baru.

**Akses:** Admin

**Request Body:**
```json
{
  "name": "Electric Vehicle"
}
```

**Response `201`:**
```json
{
  "success": true,
  "message": "Category created",
  "data": { "id": 5, "name": "Electric Vehicle" }
}
```

---

### `PUT /api/v1/vehicles/:id`
Update informasi kendaraan.

**Akses:** Admin

**Request Body:** *(sama seperti POST, semua field wajib)*

**Response `200`:** *(data kendaraan yang diperbarui)*

---

### `PATCH /api/v1/vehicles/:id/status`
Ubah status operasional kendaraan.

**Akses:** Admin

**Request Body:**
```json
{
  "status": "MAINTENANCE"
}
```

> Nilai yang valid: `AVAILABLE` | `MAINTENANCE` | `INACTIVE`

**Response `200`:** *(data kendaraan)*

---

### `PATCH /api/v1/vehicles/:id/photo`
Upload/update foto utama katalog kendaraan.

**Akses:** Admin

**Content-Type:** `multipart/form-data`

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `photo` | file | ✅ | File gambar (jpg/png) |

**Response `200`:**
```json
{
  "success": true,
  "message": "Vehicle photo updated",
  "data": {
    "photoUrl": "/files/vehicles/avanza-new.jpg"
  }
}
```

---

### `DELETE /api/v1/vehicles/:id`
Hapus kendaraan.

**Akses:** Admin

**Response `200`:**
```json
{
  "success": true,
  "message": "Vehicle deleted",
  "data": null
}
```

---

### `DELETE /api/v1/vehicles/categories/:id`
Hapus kategori kendaraan.

**Akses:** Admin

**Response `200`:**
```json
{
  "success": true,
  "message": "Category deleted",
  "data": null
}
```

---

### `GET /api/v1/vehicles/:id/attachments`
Daftar dokumen lampiran kendaraan (STNK, asuransi, foto detail, dll).

**Akses:** User (Auth required)

**Response `200`:**
```json
{
  "success": true,
  "message": "Attachments retrieved",
  "data": [
    {
      "id": 1,
      "uploadedById": 1,
      "uploaderName": "Admin",
      "vehicleId": 1,
      "roomId": null,
      "bookingId": null,
      "filePath": "/files/attachments/stnk-avanza.pdf",
      "fileName": "stnk-avanza.pdf",
      "fileType": "application/pdf",
      "fileSize": 204800,
      "description": "STNK Toyota Avanza 2022",
      "createdAt": "2025-05-26T10:00:00Z"
    }
  ]
}
```

---

### `POST /api/v1/vehicles/:id/attachments`
Upload dokumen lampiran kendaraan.

**Akses:** Admin

**Content-Type:** `multipart/form-data`

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `file` | file | ✅ | File dokumen/gambar |
| `description` | string | ❌ | Deskripsi dokumen |

**Response `201`:** *(data attachment)*

---

## 🏢 Modul 4 — Ruangan (`/api/v1/rooms`)

### `GET /api/v1/rooms`
Daftar semua ruangan rapat.

**Akses:** User (Auth required)

**Query Parameters:**
| Parameter | Type | Keterangan |
|-----------|------|-----------|
| `page` | integer | Halaman (default: 1) |
| `limit` | integer | Jumlah per halaman (default: 20) |
| `search` | string | Cari berdasarkan nama/lokasi |
| `status` | string | `AVAILABLE` \| `MAINTENANCE` \| `INACTIVE` |

**Response `200`:**
```json
{
  "success": true,
  "message": "Rooms retrieved",
  "data": [
    {
      "id": 1,
      "resourceId": 20,
      "name": "Ruang Rapat A",
      "location": "Lantai 3, Gedung Utama",
      "capacity": 20,
      "status": "AVAILABLE",
      "photoUrl": "/files/rooms/ruang-a.jpg"
    }
  ],
  "pagination": { "total": 5, "page": 1, "limit": 20, "totalPages": 1 }
}
```

---

### `GET /api/v1/rooms/:id`
Detail ruangan berdasarkan ID.

**Akses:** User (Auth required)

**Response `200`:**
```json
{
  "success": true,
  "message": "Room retrieved",
  "data": {
    "id": 1,
    "resourceId": 20,
    "name": "Ruang Rapat A",
    "location": "Lantai 3, Gedung Utama",
    "capacity": 20,
    "status": "AVAILABLE",
    "photoUrl": "/files/rooms/ruang-a.jpg"
  }
}
```

---

### `POST /api/v1/rooms`
Tambah ruangan baru.

**Akses:** Admin

**Request Body:**
```json
{
  "name": "Ruang Rapat B",
  "location": "Lantai 2, Gedung Barat",
  "capacity": 10
}
```

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `name` | string | ✅ | Nama ruangan |
| `location` | string | ✅ | Lokasi ruangan |
| `capacity` | integer | ✅ | Kapasitas orang (min: 1) |

**Response `201`:** *(data ruangan)*

---

### `PUT /api/v1/rooms/:id`
Update informasi ruangan.

**Akses:** Admin

**Request Body:** *(sama seperti POST)*

**Response `200`:** *(data ruangan yang diperbarui)*

---

### `PATCH /api/v1/rooms/:id/status`
Ubah status ruangan.

**Akses:** Admin

**Request Body:**
```json
{
  "status": "MAINTENANCE"
}
```

> Nilai yang valid: `AVAILABLE` | `MAINTENANCE` | `INACTIVE`

**Response `200`:** *(data ruangan)*

---

### `PATCH /api/v1/rooms/:id/photo`
Upload/update foto utama katalog ruangan.

**Akses:** Admin

**Content-Type:** `multipart/form-data`

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `photo` | file | ✅ | File gambar (jpg/png) |

**Response `200`:**
```json
{
  "success": true,
  "message": "Room photo updated",
  "data": {
    "photoUrl": "/files/rooms/ruang-a-new.jpg"
  }
}
```

---

### `DELETE /api/v1/rooms/:id`
Hapus ruangan.

**Akses:** Admin

**Response `200`:**
```json
{
  "success": true,
  "message": "Room deleted",
  "data": null
}
```

---

### `GET /api/v1/rooms/:id/attachments`
Daftar dokumen lampiran ruangan (SOP, layout, foto fasilitas, dll).

**Akses:** User (Auth required)

**Response `200`:** *(sama seperti vehicle attachments)*

---

### `POST /api/v1/rooms/:id/attachments`
Upload dokumen lampiran ruangan.

**Akses:** Admin

**Content-Type:** `multipart/form-data`

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `file` | file | ✅ | File dokumen/gambar |
| `description` | string | ❌ | Deskripsi dokumen |

**Response `201`:** *(data attachment)*

---

## 📅 Modul 5 — Booking (`/api/v1/bookings`)

### Status Booking
```
PENDING → APPROVED → ONGOING → COMPLETED
                ↓
            REJECTED
    ↑
  CANCELLED (dari PENDING)
```

---

### `GET /api/v1/bookings`
Daftar booking. User hanya melihat miliknya, Admin melihat semua.

**Akses:** User/Admin (Auth required)

**Query Parameters:**
| Parameter | Type | Keterangan |
|-----------|------|-----------|
| `page` | integer | Halaman (default: 1) |
| `limit` | integer | Jumlah per halaman (default: 20) |
| `userId` | integer | Filter berdasarkan user (Admin only) |
| `status` | string | `PENDING` \| `APPROVED` \| `ONGOING` \| `COMPLETED` \| `REJECTED` \| `CANCELLED` |
| `resourceId` | integer | Filter berdasarkan resource |
| `resourceType` | string | `VEHICLE` \| `ROOM` |
| `driverId` | integer | Filter berdasarkan driver |
| `startDate` | string | RFC3339 format: `2025-05-01T00:00:00Z` |
| `endDate` | string | RFC3339 format: `2025-05-31T23:59:59Z` |

**Response `200`:**
```json
{
  "success": true,
  "message": "Bookings retrieved",
  "data": [
    {
      "id": 1,
      "status": "PENDING",
      "purpose": "Kunjungan klien ke Bandung",
      "user": {
        "id": 1,
        "name": "Budi Santoso",
        "employeeId": "EMP-001",
        "department": "Operations"
      },
      "resource": {
        "id": 10,
        "name": "Toyota Avanza",
        "type": "VEHICLE",
        "status": "AVAILABLE"
      },
      "startDate": "2025-06-01T08:00:00Z",
      "endDate": "2025-06-01T18:00:00Z",
      "approvedBy": null,
      "approvedAt": null,
      "assignedAt": null,
      "returnedAt": null,
      "assignedDriver": null,
      "assignedVehicle": null,
      "createdAt": "2025-05-26T10:00:00Z",
      "updatedAt": "2025-05-26T10:00:00Z"
    }
  ],
  "pagination": { "total": 30, "page": 1, "limit": 20, "totalPages": 2 }
}
```

---

### `GET /api/v1/bookings/:id`
Detail satu booking.

**Akses:** User (hanya miliknya) / Admin (semua)

**Response `200`:**
```json
{
  "success": true,
  "message": "Booking retrieved",
  "data": {
    "id": 1,
    "status": "APPROVED",
    "purpose": "Kunjungan klien ke Bandung",
    "user": { "id": 1, "name": "Budi Santoso", "employeeId": "EMP-001", "department": "Operations" },
    "resource": { "id": 10, "name": "Toyota Avanza", "type": "VEHICLE", "status": "AVAILABLE" },
    "startDate": "2025-06-01T08:00:00Z",
    "endDate": "2025-06-01T18:00:00Z",
    "approvedBy": { "id": 2, "name": "Admin Utama" },
    "approvedAt": "2025-05-27T09:00:00Z",
    "assignedAt": "2025-05-27T09:05:00Z",
    "returnedAt": null,
    "assignedDriver": { "id": 3, "name": "Joko", "phoneNumber": "081234567890" },
    "assignedVehicle": { "id": 1, "plateNumber": "B 1234 ABC" },
    "createdAt": "2025-05-26T10:00:00Z",
    "updatedAt": "2025-05-27T09:05:00Z"
  }
}
```

---

### `POST /api/v1/bookings`
Ajukan permohonan booking baru.

**Akses:** User (Auth required)

**Request Body:**
```json
{
  "resourceId": 10,
  "startDate": "2025-06-01T08:00:00Z",
  "endDate": "2025-06-01T18:00:00Z",
  "purpose": "Kunjungan klien ke Bandung"
}
```

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `resourceId` | integer | ✅ | ID resource (kendaraan atau ruangan) |
| `startDate` | string | ✅ | Format RFC3339 |
| `endDate` | string | ✅ | Format RFC3339 |
| `purpose` | string | ✅ | Tujuan peminjaman |

> ⚠️ Sistem otomatis mengecek conflict jadwal. Jika resource sedang dipesan pada rentang waktu yang sama, request akan ditolak dengan `409 Conflict`.

**Response `201`:** *(data booking)*

---

### `PATCH /api/v1/bookings/:id/cancel`
Batalkan booking (hanya booking milik sendiri yang masih `PENDING`).

**Akses:** User (Auth required)

**Response `200`:**
```json
{
  "success": true,
  "message": "Booking cancelled",
  "data": { "id": 1, "status": "CANCELLED" }
}
```

---

### `POST /api/v1/bookings/:id/approve`
Setujui booking.

**Akses:** Admin

**Request Body (opsional):**
```json
{
  "note": "Disetujui. Silakan hubungi driver."
}
```

**Response `200`:** *(data booking dengan status `APPROVED`)*

---

### `POST /api/v1/bookings/:id/reject`
Tolak booking.

**Akses:** Admin

**Request Body:**
```json
{
  "note": "Kendaraan tidak tersedia pada tanggal tersebut."
}
```

**Response `200`:** *(data booking dengan status `REJECTED`)*

---

### `PATCH /api/v1/bookings/:id/substitute-resource`
Ganti resource (mobil/ruangan) pada booking yang masih berstatus **PENDING**. Admin dapat mengubah pilihan resource pemohon ke resource lain yang tersedia, selama tipe resource tetap sama (mobil→mobil, ruangan→ruangan).

**Akses:** Admin

**Request Body:**
```json
{
  "resourceId": 5,
  "note": "Mobil yang diminta sedang dalam perawatan mendadak, digantikan dengan Mobil B"
}
```

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `resourceId` | integer | ✅ | ID resource pengganti |
| `note` | string | ❌ | Alasan penggantian (opsional) |

**Validasi:**
- Booking harus berstatus `PENDING`
- Resource baru harus bertipe sama (VEHICLE↔VEHICLE, ROOM↔ROOM)
- Resource baru harus berstatus `AVAILABLE`
- Tidak boleh ada konflik jadwal di resource baru pada periode yang sama

**Response `200`:** *(data booking dengan `resource` yang sudah diperbarui)*

---

### `POST /api/v1/bookings/:id/assign-vehicle`
Tugaskan kendaraan dan driver ke booking yang sudah disetujui.

**Akses:** Admin

**Request Body:**
```json
{
  "driverId": 3,
  "vehicleId": 1
}
```

**Response `200`:** *(data booking dengan info driver dan vehicle)*

---

### `PATCH /api/v1/bookings/:id/start`
Mulai perjalanan/penggunaan (status: `ONGOING`).

**Akses:** Driver / Admin

**Response `200`:** *(data booking dengan status `ONGOING`)*

---

### `PATCH /api/v1/bookings/:id/complete`
Selesaikan booking (status: `COMPLETED`).

**Akses:** Admin

**Response `200`:** *(data booking dengan status `COMPLETED`)*

---

### `POST /api/v1/bookings/:id/rate-driver`
Berikan rating kepada driver setelah booking selesai.

**Akses:** User (Auth required)

**Request Body:**
```json
{
  "rating": 5,
  "review": "Driver sangat ramah dan tepat waktu."
}
```

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `rating` | integer | ✅ | Nilai 1-5 |
| `review` | string | ❌ | Ulasan teks |

**Response `201`:**
```json
{
  "success": true,
  "message": "Driver rated",
  "data": {
    "id": 1,
    "bookingId": 1,
    "driverId": 3,
    "rating": 5,
    "review": "Driver sangat ramah dan tepat waktu.",
    "createdAt": "2025-06-01T20:00:00Z"
  }
}
```

---

### `GET /api/v1/bookings/drivers/:driver_id/ratings`
Daftar rating/review driver tertentu.

**Akses:** Admin

**Response `200`:**
```json
{
  "success": true,
  "message": "Driver ratings retrieved",
  "data": [
    {
      "id": 1,
      "bookingId": 1,
      "rating": 5,
      "review": "Sangat baik",
      "reviewerName": "Budi Santoso",
      "createdAt": "2025-06-01T20:00:00Z"
    }
  ]
}
```

---

### `GET /api/v1/bookings/:id/approval-log`
Riwayat approve/reject suatu booking.

**Akses:** Admin

**Response `200`:**
```json
{
  "success": true,
  "message": "Approval log retrieved",
  "data": [
    {
      "id": 1,
      "bookingId": 1,
      "action": "APPROVED",
      "note": "Disetujui",
      "actionBy": { "id": 2, "name": "Admin Utama" },
      "actionAt": "2025-05-27T09:00:00Z"
    }
  ]
}
```

---

### `GET /api/v1/bookings/:id/activity`
Riwayat seluruh aktivitas pada suatu booking — termasuk pembuatan, approve/reject, penggantian resource, assignment, mulai, selesai, pembatalan, dan rating driver. Note dari setiap aksi (termasuk note penggantian resource) tersedia di field `description`.

**Akses:** User (hanya miliknya) / Admin (semua)

**Response `200`:**
```json
{
  "success": true,
  "message": "Activity log retrieved",
  "data": [
    {
      "id": 12,
      "action": "CREATE",
      "description": null,
      "actor": "Budi Santoso",
      "createdAt": "2025-06-01T08:00:00Z"
    },
    {
      "id": 15,
      "action": "SUBSTITUTE_RESOURCE",
      "description": "Mobil yang diminta sedang dalam perawatan mendadak, digantikan dengan Mobil B",
      "actor": "Admin Utama",
      "createdAt": "2025-06-01T09:30:00Z"
    },
    {
      "id": 18,
      "action": "APPROVE",
      "description": "Disetujui. Silakan hubungi driver.",
      "actor": "Admin Utama",
      "createdAt": "2025-06-01T10:00:00Z"
    }
  ]
}
```

| Field | Type | Keterangan |
|-------|------|-----------|
| `action` | string | `CREATE` \| `APPROVE` \| `REJECT` \| `CANCEL` \| `ASSIGN` \| `START` \| `COMPLETE` \| `RATE_DRIVER` \| `SUBSTITUTE_RESOURCE` |
| `description` | string\|null | Catatan/note yang dimasukkan saat aksi dilakukan |
| `actor` | string\|null | Nama user yang melakukan aksi (null jika aksi sistem) |
| `createdAt` | string | Waktu aksi dilakukan (RFC3339) |

---

### `GET /api/v1/bookings/:id/attachments`
Daftar lampiran pada suatu booking.

**Akses:** User (Auth required)

**Response `200`:** *(sama seperti vehicle/room attachments)*

---

### `POST /api/v1/bookings/:id/attachments`
Upload lampiran pada booking (surat tugas, dokumen, dll).

**Akses:** User (Auth required)

**Content-Type:** `multipart/form-data`

| Field | Type | Required |
|-------|------|----------|
| `file` | file | ✅ |
| `description` | string | ❌ |

**Response `201`:** *(data attachment)*

---

## 🧑‍✈️ Modul 6 — Driver (`/api/v1/drivers`)

### `GET /api/v1/drivers`
Daftar semua driver.

**Akses:** User/Admin (Auth required)

**Query Parameters:**
| Parameter | Type | Keterangan |
|-----------|------|-----------|
| `page` | integer | Halaman (default: 1) |
| `limit` | integer | Jumlah per halaman (default: 20) |

**Response `200`:**
```json
{
  "success": true,
  "message": "Drivers retrieved",
  "data": [
    {
      "id": 1,
      "userId": 5,
      "name": "Joko Susilo",
      "employeeId": "DRV-001",
      "email": "joko@example.com",
      "licenseNumber": "SIM123456",
      "phoneNumber": "081234567890",
      "isActive": true,
      "assignedPlate": "B 1234 ABC"
    }
  ],
  "pagination": { "total": 5, "page": 1, "limit": 20, "totalPages": 1 }
}
```

---

### `GET /api/v1/drivers/:driver_id`
Detail profil driver.

**Akses:** User/Admin (Auth required)

**Response `200`:**
```json
{
  "success": true,
  "message": "Driver retrieved",
  "data": {
    "id": 1,
    "userId": 5,
    "name": "Joko Susilo",
    "employeeId": "DRV-001",
    "email": "joko@example.com",
    "profilePhoto": "/files/photos/joko.jpg",
    "licenseNumber": "SIM123456",
    "phoneNumber": "081234567890",
    "isActive": true,
    "assignedPlate": "B 1234 ABC"
  }
}
```

---

### `POST /api/v1/drivers`
Daftarkan user ber-role DRIVER ke profil driver.

**Akses:** Admin

**Request Body:**
```json
{
  "userId": 5,
  "licenseNumber": "SIM123456",
  "phoneNumber": "081234567890"
}
```

**Response `201`:** *(data driver)*

---

### `PUT /api/v1/drivers/:driver_id`
Update informasi driver.

**Akses:** Admin

**Request Body:**
```json
{
  "licenseNumber": "SIM999999",
  "phoneNumber": "089876543210"
}
```

**Response `200`:** *(data driver)*

---

### `PATCH /api/v1/drivers/:driver_id/toggle-active`
Aktifkan/nonaktifkan driver.

**Akses:** Admin

**Response `200`:**
```json
{
  "success": true,
  "message": "Driver status toggled",
  "data": { "id": 1, "isActive": false }
}
```

---

### `POST /api/v1/drivers/:driver_id/assign`
Tugaskan driver ke kendaraan tertentu.

**Akses:** Admin

**Request Body:**
```json
{
  "vehicleId": 1
}
```

**Response `200`:** *(data driver dengan assignedPlate)*

---

### `PATCH /api/v1/drivers/:driver_id/release`
Lepaskan driver dari kendaraannya.

**Akses:** Admin

**Response `200`:** *(data driver dengan `assignedPlate: null`)*

---

### `GET /api/v1/drivers/:driver_id/assignments`
Riwayat kendaraan yang pernah dikemudikan driver.

**Akses:** Admin

**Response `200`:**
```json
{
  "success": true,
  "message": "Assignment history retrieved",
  "data": [
    {
      "vehicleId": 1,
      "plateNumber": "B 1234 ABC",
      "assignedAt": "2025-04-01T08:00:00Z",
      "releasedAt": "2025-05-01T08:00:00Z"
    }
  ]
}
```

---

## ⛽ Modul 7 — Pengeluaran BBM & Listrik (`/api/v1/fuel-expenses`)

### `GET /api/v1/fuel-expenses`
Daftar riwayat pengisian BBM/listrik.

**Akses:** Driver / Admin (Auth required)

**Query Parameters:**
| Parameter | Type | Keterangan |
|-----------|------|-----------|
| `page` | integer | Halaman (default: 1) |
| `limit` | integer | Jumlah per halaman (default: 20) |
| `driverId` | integer | Filter berdasarkan driver |
| `vehicleId` | integer | Filter berdasarkan kendaraan |
| `fuelType` | string | `BBM` \| `LISTRIK` |

**Response `200`:**
```json
{
  "success": true,
  "message": "Fuel expenses retrieved",
  "data": [
    {
      "id": 1,
      "driverId": 1,
      "driverName": "Joko Susilo",
      "vehicleId": 1,
      "fuelType": "BBM",
      "liter": 30.5,
      "pricePerLiter": 10000,
      "totalCost": 305000,
      "odometerBefore": 15000,
      "odometerAfter": 15350,
      "note": "Pengisian di SPBU Cibubur",
      "createdAt": "2025-06-01T12:00:00Z"
    }
  ],
  "pagination": { "total": 20, "page": 1, "limit": 20, "totalPages": 1 }
}
```

---

### `GET /api/v1/fuel-expenses/:id`
Detail satu entri pengeluaran.

**Akses:** Driver / Admin

**Response `200`:** *(data satu entri fuel expense)*

---

### `POST /api/v1/fuel-expenses/bbm`
Input pengeluaran BBM baru.

**Akses:** Driver (Auth required)

**Query Parameters:** `?driverId=1` (wajib)

**Request Body:**
```json
{
  "vehicleId": 1,
  "bookingId": 5,
  "liter": 30.5,
  "pricePerLiter": 10000,
  "odometerBefore": 15000,
  "odometerAfter": 15350,
  "note": "Pengisian di SPBU Cibubur"
}
```

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `vehicleId` | integer | ✅ | ID kendaraan |
| `bookingId` | integer | ❌ | ID booking terkait |
| `liter` | float | ✅ | Jumlah liter (> 0) |
| `pricePerLiter` | float | ✅ | Harga per liter (> 0) |
| `odometerBefore` | integer | ✅ | Odometer sebelum |
| `odometerAfter` | integer | ✅ | Odometer sesudah |
| `note` | string | ❌ | Catatan tambahan |

**Response `201`:** *(data fuel expense)*

---

### `POST /api/v1/fuel-expenses/listrik`
Input pengeluaran pengisian listrik (EV) baru.

**Akses:** Driver (Auth required)

**Query Parameters:** `?driverId=1` (wajib)

**Request Body:**
```json
{
  "vehicleId": 2,
  "bookingId": 6,
  "kwh": 25.5,
  "pricePerKwh": 2500,
  "batteryBefore": 20.0,
  "batteryAfter": 85.0,
  "note": "Pengisian di SPKLU Sudirman"
}
```

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `vehicleId` | integer | ✅ | ID kendaraan EV |
| `bookingId` | integer | ❌ | ID booking terkait |
| `kwh` | float | ✅ | Jumlah kWh (> 0) |
| `pricePerKwh` | float | ✅ | Harga per kWh (> 0) |
| `batteryBefore` | float | ❌ | % baterai sebelum (0-100) |
| `batteryAfter` | float | ❌ | % baterai sesudah (0-100) |
| `note` | string | ❌ | Catatan |

**Response `201`:** *(data fuel expense)*

---

### `DELETE /api/v1/fuel-expenses/:id`
Hapus entri pengeluaran.

**Akses:** Admin

**Response `200`:**
```json
{
  "success": true,
  "message": "Fuel expense deleted",
  "data": null
}
```

---

## 🔧 Modul 8 — Pemeliharaan (`/api/v1/maintenance`)

### `GET /api/v1/maintenance`
Daftar riwayat pemeliharaan.

**Akses:** Admin

**Query Parameters:**
| Parameter | Type | Keterangan |
|-----------|------|-----------|
| `page` | integer | Halaman (default: 1) |
| `limit` | integer | Jumlah per halaman (default: 20) |
| `resourceId` | integer | Filter berdasarkan resource |

**Response `200`:**
```json
{
  "success": true,
  "message": "Maintenance records retrieved",
  "data": [
    {
      "id": 1,
      "resourceId": 10,
      "resourceName": "Toyota Avanza",
      "resourceType": "VEHICLE",
      "description": "Ganti oli mesin dan filter",
      "startDate": "2025-06-01T08:00:00Z",
      "endDate": null,
      "cost": 350000,
      "createdBy": "Admin Utama",
      "createdAt": "2025-05-31T16:00:00Z"
    }
  ],
  "pagination": { "total": 10, "page": 1, "limit": 20, "totalPages": 1 }
}
```

---

### `GET /api/v1/maintenance/:id`
Detail record pemeliharaan.

**Akses:** Admin

**Response `200`:** *(data satu maintenance record)*

---

### `POST /api/v1/maintenance`
Buat record pemeliharaan baru (otomatis ubah status resource ke `MAINTENANCE`).

**Akses:** Admin

**Request Body:**
```json
{
  "resourceId": 10,
  "description": "Ganti oli mesin dan filter",
  "startDate": "2025-06-01T08:00:00Z",
  "cost": 350000
}
```

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `resourceId` | integer | ✅ | ID resource (kendaraan/ruangan) |
| `description` | string | ✅ | Deskripsi perbaikan |
| `startDate` | string | ✅ | Tanggal mulai (RFC3339) |
| `cost` | float | ❌ | Estimasi biaya |

**Response `201`:** *(data maintenance record)*

---

### `PUT /api/v1/maintenance/:id`
Update record pemeliharaan. Jika `endDate` diisi, status resource otomatis kembali `AVAILABLE`.

**Akses:** Admin

**Request Body:**
```json
{
  "description": "Ganti oli mesin, filter, dan busi",
  "startDate": "2025-06-01T08:00:00Z",
  "endDate": "2025-06-02T16:00:00Z",
  "cost": 500000
}
```

**Response `200`:** *(data maintenance record yang diperbarui)*

---

### `DELETE /api/v1/maintenance/:id`
Hapus record pemeliharaan.

**Akses:** Admin

**Response `200`:**
```json
{
  "success": true,
  "message": "Maintenance record deleted",
  "data": null
}
```

---

## 📎 Modul 9 — Lampiran Global (`/api/v1/attachments`)

### `DELETE /api/v1/attachments/:id`
Hapus lampiran berdasarkan ID (berlaku untuk semua jenis attachment).

**Akses:** User/Admin (Auth required)

**Response `200`:**
```json
{
  "success": true,
  "message": "Attachment deleted",
  "data": null
}
```

---

## 👤 Modul 10 — Booking Tamu (`/api/v1/guest-bookings`)

### `POST /api/v1/guest-bookings`
Buat reservasi tamu tanpa login.

**Akses:** Public

**Request Body:**
```json
{
  "guestName": "Andi Wijaya",
  "guestEmail": "andi@external.com",
  "guestPhone": "0821XXXXXXXX",
  "departmentName": "PT. Mitra Utama",
  "resourceId": 20,
  "startDate": "2025-06-05T09:00:00Z",
  "endDate": "2025-06-05T12:00:00Z",
  "purpose": "Presentasi proyek kerjasama"
}
```

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `guestName` | string | ✅ | Nama tamu |
| `guestEmail` | string | ✅ | Email tamu (format valid) |
| `guestPhone` | string | ✅ | Nomor HP tamu |
| `departmentName` | string | ✅ | Nama perusahaan/departemen |
| `resourceId` | integer | ✅ | ID resource |
| `startDate` | string | ✅ | RFC3339 |
| `endDate` | string | ✅ | RFC3339 |
| `purpose` | string | ✅ | Tujuan reservasi |

**Response `201`:**
```json
{
  "success": true,
  "message": "Guest booking created",
  "data": {
    "id": 1,
    "guestName": "Andi Wijaya",
    "guestEmail": "andi@external.com",
    "accessToken": "gst_a1b2c3d4e5f6...",
    "status": "PENDING",
    "resource": { "id": 20, "name": "Ruang Rapat A", "type": "ROOM" },
    "startDate": "2025-06-05T09:00:00Z",
    "endDate": "2025-06-05T12:00:00Z"
  }
}
```

> ℹ️ `accessToken` dikirimkan ke email tamu untuk melacak status reservasi.

---

### `GET /api/v1/guest-bookings/:token`
Cek status reservasi tamu via token.

**Akses:** Public

**Response `200`:**
```json
{
  "success": true,
  "message": "Guest booking retrieved",
  "data": {
    "id": 1,
    "guestName": "Andi Wijaya",
    "guestEmail": "andi@external.com",
    "status": "APPROVED",
    "resource": { "id": 20, "name": "Ruang Rapat A", "type": "ROOM" },
    "startDate": "2025-06-05T09:00:00Z",
    "endDate": "2025-06-05T12:00:00Z",
    "approvedBy": "Admin Utama",
    "approvedAt": "2025-06-04T10:00:00Z",
    "rejectionNote": null,
    "returnedAt": null,
    "createdAt": "2025-06-03T14:00:00Z"
  }
}
```

---

### `PATCH /api/v1/guest-bookings/:token/complete`
Tamu menandai penggunaan selesai via token.

**Akses:** Public

**Response `200`:** *(data booking dengan status `COMPLETED`)*

---

### `PATCH /api/v1/guest-bookings/:token/cancel`
Tamu membatalkan reservasi via token.

**Akses:** Public

**Response `200`:** *(data booking dengan status `CANCELLED`)*

---

### `GET /api/v1/guest-bookings`
Daftar semua reservasi tamu (Admin).

**Akses:** Admin

**Query Parameters:**
| Parameter | Type | Keterangan |
|-----------|------|-----------|
| `page` | integer | Halaman (default: 1) |
| `limit` | integer | Jumlah per halaman (default: 20) |
| `status` | string | Filter status |

**Response `200`:** *(paginated list guest bookings)*

---

### `POST /api/v1/guest-bookings/:id/approve`
Setujui booking tamu.

**Akses:** Admin

**Response `200`:** *(data guest booking dengan status `APPROVED`)*

---

### `POST /api/v1/guest-bookings/:id/reject`
Tolak booking tamu.

**Akses:** Admin

**Request Body:**
```json
{
  "note": "Ruangan sedang dalam renovasi."
}
```

**Response `200`:** *(data guest booking dengan status `REJECTED`)*

---

### `PATCH /api/v1/guest-bookings/:id/start`
Mulai penggunaan booking tamu.

**Akses:** Admin

**Response `200`:** *(data guest booking dengan status `ONGOING`)*

---

## ⚙️ Modul 11 — Pengaturan Global (`/api/v1/master-settings`)

### `GET /api/v1/master-settings`
Daftar semua pengaturan global.

**Akses:** User/Admin (Auth required)

**Response `200`:**
```json
{
  "success": true,
  "message": "Settings retrieved",
  "data": [
    {
      "key": "bbm_price_per_liter",
      "value": "10000",
      "unit": "IDR",
      "description": "Harga acuan BBM per liter"
    },
    {
      "key": "electricity_price_per_kwh",
      "value": "2500",
      "unit": "IDR",
      "description": "Harga acuan listrik per kWh"
    }
  ]
}
```

---

### `GET /api/v1/master-settings/:key`
Ambil nilai pengaturan berdasarkan key.

**Akses:** User/Admin (Auth required)

**Response `200`:**
```json
{
  "success": true,
  "message": "Setting retrieved",
  "data": {
    "key": "bbm_price_per_liter",
    "value": "10000",
    "unit": "IDR",
    "description": "Harga acuan BBM per liter"
  }
}
```

---

### `PUT /api/v1/master-settings/:key`
Buat atau update nilai pengaturan.

**Akses:** Admin

**Request Body:**
```json
{
  "value": 11000,
  "unit": "IDR",
  "description": "Harga acuan BBM per liter (updated)"
}
```

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `value` | float | ✅ | Nilai pengaturan |
| `unit` | string | ❌ | Satuan (IDR, kWh, dll) |
| `description` | string | ❌ | Deskripsi pengaturan |

**Response `200`:** *(data pengaturan yang diperbarui)*

---

## 📊 Modul 12 — Laporan & Analitik (`/api/v1/reports`)

### `GET /api/v1/reports/bookings`
Ringkasan statistik booking.

**Akses:** Admin

**Query Parameters:**
| Parameter | Type | Keterangan |
|-----------|------|-----------|
| `startDate` | string | RFC3339 |
| `endDate` | string | RFC3339 |

**Response `200`:**
```json
{
  "success": true,
  "message": "Booking summary",
  "data": {
    "totalBookings": 120,
    "pendingCount": 5,
    "approvedCount": 10,
    "completedCount": 95,
    "cancelledCount": 7,
    "rejectedCount": 3,
    "vehicleBookings": 80,
    "roomBookings": 40
  }
}
```

---

### `GET /api/v1/reports/resource-usage`
Laporan utilisasi kendaraan dan ruangan.

**Akses:** Admin

**Response `200`:**
```json
{
  "success": true,
  "message": "Resource usage report",
  "data": [
    {
      "resourceId": 10,
      "resourceName": "Toyota Avanza",
      "resourceType": "VEHICLE",
      "totalBookings": 25,
      "totalHoursUsed": 187.5,
      "utilizationRate": 78.5
    }
  ]
}
```

---

### `GET /api/v1/reports/fuel-expenses`
Laporan pengeluaran BBM & listrik.

**Akses:** Admin

**Response `200`:**
```json
{
  "success": true,
  "message": "Fuel expense report",
  "data": [
    {
      "vehicleId": 1,
      "plateNumber": "B 1234 ABC",
      "vehicleName": "Toyota Avanza",
      "totalLiter": 250.5,
      "totalKwh": 0,
      "totalCost": 2505000,
      "fuelType": "BBM"
    }
  ]
}
```

---

### `GET /api/v1/reports/maintenance-cost`
Rekapitulasi biaya pemeliharaan.

**Akses:** Admin

**Response `200`:**
```json
{
  "success": true,
  "message": "Maintenance cost report",
  "data": [
    {
      "resourceId": 10,
      "resourceName": "Toyota Avanza",
      "resourceType": "VEHICLE",
      "totalMaintenanceCount": 4,
      "totalCost": 2000000
    }
  ]
}
```

---

### `GET /api/v1/reports/driver-ratings`
Laporan rating semua driver.

**Akses:** Admin

**Response `200`:**
```json
{
  "success": true,
  "message": "Driver ratings report",
  "data": [
    {
      "driverId": 1,
      "driverName": "Joko Susilo",
      "averageRating": 4.7,
      "totalReviews": 23
    }
  ]
}
```

---

### `GET /api/v1/reports/driver-activity`
Laporan aktivitas dan pengeluaran BBM driver.

**Akses:** Admin

**Response `200`:**
```json
{
  "success": true,
  "message": "Driver activity report",
  "data": [
    {
      "driverId": 1,
      "driverName": "Joko Susilo",
      "totalTrips": 25,
      "totalFuelExpenses": 3500000
    }
  ]
}
```

---

### `GET /api/v1/reports/overdue-bookings`
Daftar booking yang melewati batas waktu.

**Akses:** Admin

**Response `200`:**
```json
{
  "success": true,
  "message": "Overdue bookings",
  "data": [
    {
      "id": 15,
      "status": "ONGOING",
      "user": { "id": 1, "name": "Budi Santoso" },
      "resource": { "id": 10, "name": "Toyota Avanza", "type": "VEHICLE" },
      "startDate": "2025-05-20T08:00:00Z",
      "endDate": "2025-05-20T18:00:00Z",
      "overdueHours": 36.5
    }
  ]
}
```

---

### `GET /api/v1/reports/audit-logs`
Riwayat aktivitas sistem (Audit Trail).

**Akses:** Admin

**Query Parameters:**
| Parameter | Type | Keterangan |
|-----------|------|-----------|
| `page` | integer | Halaman (default: 1) |
| `limit` | integer | Jumlah per halaman (default: 50) |
| `entityType` | string | Filter berdasarkan tipe entitas (User, Booking, dll) |
| `userId` | integer | Filter berdasarkan user |

**Response `200`:**
```json
{
  "success": true,
  "message": "Audit logs retrieved",
  "data": [
    {
      "id": 1,
      "userId": 2,
      "userName": "Admin Utama",
      "action": "APPROVE_BOOKING",
      "entityType": "Booking",
      "entityId": 1,
      "description": "Booking #1 disetujui oleh Admin Utama",
      "createdAt": "2025-05-27T09:00:00Z"
    }
  ],
  "pagination": { "total": 200, "page": 1, "limit": 50, "totalPages": 4 }
}
```

---

## 📁 Static Files

### `GET /files/*`
Serve file statis yang diupload (foto profil, foto kendaraan/ruangan, attachment).

**Akses:** Public

**Contoh URL:**
- `/files/photos/profile-1.jpg` — Foto profil user
- `/files/vehicles/avanza.jpg` — Foto katalog kendaraan
- `/files/rooms/ruang-a.jpg` — Foto katalog ruangan
- `/files/attachments/stnk.pdf` — Dokumen lampiran

---

## 🔑 Autentikasi — Cara Penggunaan

1. **Login** via `POST /api/v1/auth/login` untuk mendapatkan `accessToken`.
2. Sertakan token di setiap request yang membutuhkan autentikasi:
   ```
   Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
   ```
3. `accessToken` akan expired. Gunakan `POST /api/v1/auth/refresh` dengan `refreshToken` untuk mendapatkan token baru.
4. Pastikan logout dengan `POST /api/v1/auth/logout` saat selesai.

---

## ❌ Contoh Error Response

### 401 Unauthorized
```json
{
  "success": false,
  "message": "Unauthorized",
  "error": {
    "code": "UNAUTHORIZED",
    "message": "token is invalid or expired"
  }
}
```

### 403 Forbidden
```json
{
  "success": false,
  "message": "Forbidden",
  "error": {
    "code": "FORBIDDEN",
    "message": "access denied: insufficient role"
  }
}
```

### 404 Not Found
```json
{
  "success": false,
  "message": "Not Found",
  "error": {
    "code": "NOT_FOUND",
    "message": "data not found"
  }
}
```

### 409 Conflict
```json
{
  "success": false,
  "message": "Conflict",
  "error": {
    "code": "CONFLICT",
    "message": "schedule conflict: resource already booked on selected date range"
  }
}
```

### 422 Unprocessable Entity
```json
{
  "success": false,
  "message": "Validation failed",
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Key: 'CreateBookingRequest.Purpose' Error:Field validation for 'Purpose' failed on the 'required' tag"
  }
}
```