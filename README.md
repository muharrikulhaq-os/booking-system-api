# 📘 API Documentation — Booking System API

> **Base URL:** `http://localhost:8080`  
> **API Prefix:** `/api/v1`  
> **Auth:** Bearer JWT Token via `Authorization: Bearer <access_token>`  
> **Content-Type:** `application/json` (kecuali upload file: `multipart/form-data`)

---
## 🚀 Recent Updates (KCE - Bookings, Drivers, Fuel, Maintenance, & Export)
1. **Fuel Expense Optimization**: Migrated fuel price source from error-prone Master Settings strings to a robust `fuel_types` database table. Fuel expenses (`BBM` and `Listrik`) now reliably fetch `default_price` using `fuelTypeId`.
2. **Merge Booking & Driver Assignment**: Admin users can now specify a `DriverID` when merging bookings (`POST /bookings/:id/merge`). The system validates vehicle capacity conflicts and handles driver assignment out-of-the-gate via `assignedDriverId`.
3. **WebSocket Real-time Notifications**: Implemented a WebSocket hub at `/api/v1/ws`. Connected drivers receive `NEW_BOOKING` notifications when assigned, and users receive `BOOKING_APPROVED` notifications upon admin approval.
4. **Maintenance Tracking Enhancements**: Upgraded `PATCH /maintenance/:id/complete` to accept `multipart/form-data` uploads (proof photos), mark completion dates automatically, and instantly set vehicle status back to `AVAILABLE`.
5. **Comprehensive Reports & Excel Export**: Fully integrated reporting endpoints (e.g., `/cost-summary`, `/bookings/by-department`) along with an advanced `GET /reports/export/excel` endpoint that exports all metrics across 18 sheets with native Pie/Line Charts built directly into the generated `.xlsx`.

## 📐 Standar Response Format

### WebSocket Connection (Real-time Notifications)
Sistem menggunakan WebSocket untuk push notification secara real-time ke *Driver* maupun *User*. 

**Endpoint:** `GET /api/v1/ws`

**Auth & Koneksi:**
Karena WebSocket bawaan browser tidak mendukung custom header `Authorization`, Anda bisa menyisipkan token via query parameter:
```javascript
const token = "eyJhbGciOiJIUzI1...";
const ws = new WebSocket(`ws://localhost:8080/api/v1/ws?token=${token}`);

ws.onmessage = function(event) {
  const data = JSON.parse(event.data);
  console.log("New Notification:", data);
};
```

**Struktur Payload Notifikasi (JSON):**
```json
{
  "type": "NEW_BOOKING", 
  "message": "You have been assigned to a new booking",
  "data": {
    "bookingId": 999,
    "vehicleId": 10
  }
}
```
*Event Type yang didukung saat ini:*
- `NEW_BOOKING`: Diterima oleh Driver saat ditugaskan ke booking baru (saat booking di-approve atau re-assign).
- `BOOKING_APPROVED`: Diterima oleh User saat booking mereka disetujui oleh Admin.

---

### REST API Success Response
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
| `EMPLOYEE` | Pengguna internal, bisa membuat booking |
| `DRIVER` | Pengemudi, bisa mulai perjalanan, input BBM, dan submit return report booking kendaraan |
| `ROOM_KEEPER` | Pengawas ruangan, bisa start & complete booking ruangan |

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
| `search` | string | Cari berdasarkan nama, email, atau employeeId |
| `roleId` | integer | Filter berdasarkan role |
| `departmentId` | integer | Filter berdasarkan departemen |

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

### `POST /api/v1/users/departments`
Buat departemen baru.

**Akses:** Admin

**Request Body:**
```json
{ "name": "Legal" }
```

**Response `201`:**
```json
{
  "success": true,
  "message": "Department created",
  "data": { "id": 4, "name": "Legal" }
}
```

---

### `PUT /api/v1/users/departments/:id`
Update nama departemen.

**Akses:** Admin

**Request Body:**
```json
{ "name": "Legal & Compliance" }
```

**Response `200`:**
```json
{
  "success": true,
  "message": "Department updated",
  "data": { "id": 4, "name": "Legal & Compliance" }
}
```

---

### `DELETE /api/v1/users/departments/:id`
Hapus departemen.

**Akses:** Admin

**Response `200`:**
```json
{ "success": true, "message": "Department deleted", "data": null }
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
    ↓           ↓         ↓
 IGNORED     EXPIRED   OVERDUE
    ↑
 REJECTED
    ↑
 CANCELLED (dari PENDING)
```

| Status | Keterangan |
|--------|-----------|
| `PENDING` | Menunggu persetujuan admin |
| `APPROVED` | Disetujui admin, menunggu dimulai |
| `ONGOING` | Sedang berlangsung |
| `COMPLETED` | Selesai |
| `REJECTED` | Ditolak admin |
| `CANCELLED` | Dibatalkan oleh pemohon |
| `OVERDUE` | ONGOING tetapi melewati `endDate` (belum selesai tepat waktu) |
| `EXPIRED` | APPROVED tetapi tidak pernah dimulai dan `endDate` sudah lewat |
| `IGNORED` | PENDING tetapi admin tidak merespons sampai `endDate` lewat |

### Alur Selesai Booking Kendaraan (Vehicle Return Flow)
```
[Driver] ONGOING → POST /return-report (note + foto + lokasi)
                        ↓
[Admin]  GET /return-report → review laporan driver
                        ↓
[Admin]  PATCH /complete → COMPLETED
```

> Driver wajib submit return report terlebih dahulu agar admin dapat melihat kondisi kendaraan sebelum menyelesaikan booking.

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
| `status` | string | `PENDING` \| `APPROVED` \| `ONGOING` \| `COMPLETED` \| `REJECTED` \| `CANCELLED` \| `OVERDUE` \| `EXPIRED` \| `IGNORED` |
| `resourceId` | integer | Filter berdasarkan resource |
| `resourceType` | string | `VEHICLE` \| `ROOM` |
| `driverId` | integer | Filter berdasarkan driver |
| `startDate` | string | RFC3339 format: `2025-05-01T00:00:00Z` |
| `endDate` | string | RFC3339 format: `2025-05-31T23:59:59Z` |
| `search` | string | Cari berdasarkan nama resource atau nama karyawan |

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
  "purpose": "Kunjungan klien ke Bandung",
  "passengerCount": 4,
  "assignedDriverId": 3
}
```

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `resourceId` | integer | ✅ | ID resource (kendaraan atau ruangan) |
| `startDate` | string | ✅ | Format RFC3339 |
| `endDate` | string | ✅ | Format RFC3339 |
| `purpose` | string | ✅ | Tujuan peminjaman |
| `passengerCount` | integer | ✅ | Jumlah penumpang (min: 1) |
| `assignedDriverId` | integer | ❌ | ID Supir (opsional, Auto-assign jika null) |

> ⚠️ Sistem akan mengecek jadwal. Konflik untuk ruangan bersifat strict. Namun untuk kendaraan (berdasarkan paradigma System-Validated), beberapa request dapat berstatus PENDING secara paralel sebelum di-_approve_ oleh Admin.

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

> **Catatan (Soft Warning):** Jika kapasitas mobil tersisa tidak mencukupi untuk jumlah `passengerCount` saat ini, API tidak menolak secara hard-block, tetapi akan me-return properti `"warning"` di dalam response body JSON. Admin berhak untuk melakukan _override_ ini (misalnya nanti mensubstitusi resource).

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

**Catatan Pengalihan Resource:** Jika kendaraan yang ditugaskan berbeda dari resource yang di-booking user, sistem secara otomatis:
1. Memperbarui `resourceId` booking ke resource kendaraan aktual (kalender & cek konflik akan akurat)
2. Menyimpan resource awal user di `originalResourceId`

Response akan menyertakan `isReassigned: true` dan field `originalResource` berisi info resource awal.

**Response `200`:**
```json
{
  "success": true,
  "message": "Vehicle assigned",
  "data": {
    "id": 11,
    "resource": { "id": 8, "name": "Toyota Tuktuk - A 1234 CFD", "type": "VEHICLE" },
    "isReassigned": true,
    "originalResource": { "id": 2, "name": "Honda CR-V - B 5678 AB", "type": "VEHICLE" },
    "assignedDriver": { "id": 1, "name": "Pak Supir Satu" },
    "assignedVehicle": { "id": 8, "plateNumber": "A 1234 CFD", "brand": "Toyota", "model": "Tuktuk" }
  }
}
```

---

### `PATCH /api/v1/bookings/:id/start`
Mulai perjalanan/penggunaan booking (status: `ONGOING`).

**Akses:**
- `ADMIN` — dapat memulai booking kendaraan atau ruangan
- `DRIVER` — hanya booking kendaraan yang ditugaskan kepada dirinya
- `ROOM_KEEPER` — hanya booking ruangan

**Response `200`:** *(data booking dengan status `ONGOING`)*

---

### `PATCH /api/v1/bookings/:id/complete`
Selesaikan booking (status: `COMPLETED`).

**Akses:**
- `ADMIN` — dapat menyelesaikan booking kendaraan atau ruangan
- `ROOM_KEEPER` — hanya booking ruangan

> Untuk booking kendaraan, disarankan admin melihat return report driver terlebih dahulu via `GET /bookings/:id/return-report` sebelum melakukan complete.

**Response `200`:** *(data booking dengan status `COMPLETED`)*

---

### `POST /api/v1/bookings/:id/return-report`
Driver mengirimkan laporan akhir perjalanan (note, lokasi, foto kondisi kendaraan). Admin kemudian mereview laporan ini sebelum menyelesaikan booking.

**Akses:** `DRIVER` (hanya driver yang di-assign ke booking ini)

**Syarat:**
- Booking bertipe `VEHICLE`
- Status booking `ONGOING` atau `OVERDUE`
- Belum pernah submit return report untuk booking ini

**Content-Type:** `multipart/form-data`

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `note` | string | ✅ | Catatan driver tentang perjalanan/kondisi kendaraan |
| `location` | string | ✅ | Lokasi saat ini (string bebas, contoh: "Kantor Pusat Jl. Sudirman") |
| `photos[]` | file | ❌ | Foto kondisi kendaraan (bisa multiple, jpg/png) |

**Response `201`:**
```json
{
  "success": true,
  "message": "Return report submitted",
  "data": {
    "bookingId": 6,
    "note": "Kendaraan dikembalikan dalam kondisi baik.",
    "location": "Kantor Pusat, Jl. Sudirman No. 1 Jakarta",
    "photos": [
      {
        "id": 12,
        "filePath": "booking/2026/06/abc123.jpg",
        "fileName": "kondisi-depan.jpg",
        "fileType": "image/jpeg"
      }
    ]
  }
}
```

---

### `GET /api/v1/bookings/:id/return-report`
Lihat laporan akhir perjalanan yang sudah disubmit driver.

**Akses:** `ADMIN` atau `DRIVER` yang di-assign ke booking ini

**Response `200`:**
```json
{
  "success": true,
  "message": "Return report retrieved",
  "data": {
    "id": 1,
    "bookingId": 6,
    "submittedBy": {
      "id": 6,
      "name": "Pak Supir Satu"
    },
    "note": "Kendaraan dikembalikan dalam kondisi baik. Tidak ada kerusakan.",
    "location": "Kantor Pusat, Jl. Sudirman No. 1 Jakarta",
    "submittedAt": "2026-06-07T14:30:00Z",
    "photos": [
      {
        "id": 12,
        "filePath": "booking/2026/06/abc123.jpg",
        "fileName": "kondisi-depan.jpg",
        "fileType": "image/jpeg"
      }
    ]
  }
}
```

**Error `404`:** Jika driver belum submit return report.

---

### `POST /api/v1/bookings/:id/merge`
Gabungkan dua booking kendaraan ke satu perjalanan bersama.

**Akses:** Admin

**Request Body:**
```json
{
  "targetBookingId": 2,
  "reason": "Tujuan yang sama, kapasitas kendaraan cukup",
  "startDate": "2026-06-10T08:00:00Z",
  "endDate": "2026-06-10T10:00:00Z"
}
```

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `targetBookingId` | integer | ✅ | ID booking yang akan digabungkan |
| `reason` | string | ❌ | Alasan penggabungan |
| `startDate` | datetime | ❌ | Override waktu mulai perjalanan gabungan. Jika tidak diisi, otomatis diambil dari waktu tercepat antara kedua booking |
| `endDate` | datetime | ❌ | Override waktu selesai perjalanan gabungan. Jika tidak diisi, otomatis diambil dari waktu terlama antara kedua booking |

**Aturan merge:**
- Kedua booking harus berstatus `PENDING` atau `APPROVED`
- Kedua booking harus untuk resource tipe `VEHICLE`
- Kedua booking belum pernah digabungkan sebelumnya
- Booking yang menjadi target endpoint (`:id`) adalah **primary** (pemegang kendaraan)
- Sistem otomatis memperluas waktu primary booking ke union kedua rentang waktu (start paling awal, end paling akhir). Admin dapat mengoverride dengan `startDate`/`endDate`
- Jika perluasan waktu primary booking bertabrakan dengan booking lain pada resource yang sama, merge ditolak

**Response `201`:**
```json
{
  "success": true,
  "message": "Bookings merged",
  "data": {
    "mergeId": 1,
    "primaryBookingId": 1,
    "mergedBookingId": 2,
    "mergedBy": 1,
    "reason": "Tujuan yang sama",
    "effectiveStartDate": "2026-06-10T08:00:00Z",
    "effectiveEndDate": "2026-06-10T10:00:00Z",
    "createdAt": "2026-06-06T10:00:00Z"
  }
}
```

---

### `GET /api/v1/bookings/:id/merge-info`
Lihat informasi penggabungan untuk suatu booking.

**Akses:** Auth required (Employee hanya dapat melihat booking sendiri)

**Response `200`:**
```json
{
  "success": true,
  "message": "Merge info retrieved",
  "data": [
    {
      "mergeId": 1,
      "primaryBookingId": 1,
      "mergedBookingId": 2,
      "isPrimary": true,
      "mergedBy": "Admin Utama",
      "reason": "Tujuan yang sama",
      "createdAt": "2026-06-06T10:00:00Z",
      "linkedBooking": {
        "bookingId": 2,
        "userId": 3,
        "userName": "Jane Smith",
        "employeeId": "EMP002",
        "department": "Finance & Accounting",
        "purpose": "Kunjungan klien Bandung"
      }
    }
  ]
}
```

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
| `action` | string | `CREATE` \| `APPROVE` \| `REJECT` \| `CANCEL` \| `ASSIGN` \| `START` \| `COMPLETE` \| `RATE_DRIVER` \| `SUBSTITUTE_RESOURCE` \| `MERGE` \| `SUBMIT_RETURN_REPORT` |
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

### `GET /api/v1/drivers/available`
Ambil daftar driver beserta kendaraan yang diassign yang tersedia untuk jadwal booking, termasuk kalkulasi dynamic sisa kursi (remaining seats).

**Akses:** User (Auth required)

**Query Parameters:**
| Parameter | Type | Required | Keterangan |
|-----------|------|----------|-----------|
| `startDate` | string | ✅ | RFC3339 format |
| `endDate` | string | ✅ | RFC3339 format |

**Response `200`:**
```json
{
  "success": true,
  "message": "Available drivers retrieved",
  "data": [
    {
      "driverId": 1,
      "driverName": "Budi",
      "employeeId": "EMP-001",
      "vehicleId": 1,
      "plateNumber": "B 1234 CD",
      "vehicleCapacity": 6,
      "overlappingPassengers": 2,
      "remainingSeats": 4
    }
  ]
}
```

---

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

## ⛽ Modul Master Data — Fuel Types (`/api/v1/fuel-types`)
Manajemen master data jenis bahan bakar yang digunakan sebagai *reference* (acuan harga) saat pelaporan *Fuel Expenses*.

### `GET /api/v1/fuel-types`
Melihat daftar bahan bakar.

**Akses:** User / Driver / Admin (Auth required)

**Response `200`:**
```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "name": "Pertamax",
      "type": "BBM",
      "unit": "LITER",
      "defaultPrice": 12900.0,
      "isActive": true
    },
    {
      "id": 4,
      "name": "SPKLU PLN",
      "type": "LISTRIK",
      "unit": "KWH",
      "defaultPrice": 2466.0,
      "isActive": true
    }
  ]
}
```

---

### `POST /api/v1/fuel-types`
Menambahkan jenis bahan bakar baru.

**Akses:** Admin (Auth required)

**Request Body:**
```json
{
  "name": "Dexlite",
  "type": "BBM",
  "unit": "LITER",
  "defaultPrice": 14550.0,
  "isActive": true
}
```

**Response `201`:** *(data fuel type baru)*

---

### `PUT /api/v1/fuel-types/:id`
Mengubah data bahan bakar (misalnya update harga *defaultPrice* terbaru).

**Akses:** Admin (Auth required)

**Request Body:** Sama dengan POST.

**Response `200`:** *(data fuel type terupdate)*

---

### `DELETE /api/v1/fuel-types/:id`
Menghapus master data bahan bakar.

**Akses:** Admin (Auth required)

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

### `POST /api/v1/fuel-expenses`
Input pengeluaran BBM atau pengisian listrik (EV) baru (mendukung _multipart/form-data_ untuk _upload_ bukti foto).

**Akses:** Driver (Auth required)

**Content-Type:** `multipart/form-data`

**Form-Data Fields:**

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `vehicleId` | integer | ✅ | ID kendaraan |
| `bookingId` | integer | ❌ | ID booking terkait |
| `fuelTypeId`| integer | ✅ | ID jenis bahan bakar |
| `fuelGrade` | string | ❌ | RON/Grade bahan bakar |
| `liter` | float | ❌ | Jumlah liter (jika BBM) |
| `pricePerLiter` | float | ❌ | Harga per liter |
| `kwh` | float | ❌ | Jumlah kWh (jika EV) |
| `pricePerKwh` | float | ❌ | Harga per kWh |
| `odometerBefore` | integer | ❌ | Odometer sebelum |
| `odometerAfter` | integer | ❌ | Odometer sesudah |
| `note` | string | ❌ | Catatan tambahan |
| `proofPhoto` | file | ✅ | Bukti foto nota/struk |

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

### `PATCH /api/v1/maintenance/:id/complete`
Menyelesaikan *maintenance* dan mengunggah foto bukti (akan mengubah status `resource` menjadi `AVAILABLE`).

**Akses:** Admin

**Content-Type:** `multipart/form-data`

| Field | Type | Required | Keterangan |
|-------|------|----------|-----------|
| `photos` | file (array) | ❌ | Beberapa file foto bukti |

**Response `200`:**
```json
{
  "success": true,
  "message": "Maintenance completed",
  "data": { "id": 1, "status": "COMPLETED", "completedAt": "2026-07-06T..." }
}
```

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

### `GET /api/v1/reports/overview`
Ringkasan KPI periode ini vs periode sebelumnya (perbandingan persentase perubahan).

**Akses:** Admin

**Query Parameters:**
| Parameter | Type | Keterangan |
|-----------|------|-----------|
| `period` | string | `monthly` (default) \| `quarterly` \| `yearly` |

**Response `200`:**
```json
{
  "success": true,
  "message": "Overview report",
  "data": {
    "totalBookings": 45,
    "totalCost": 15000000,
    "avgUtilization": 72.5,
    "overdueCount": 2,
    "previousPeriod": { "totalBookings": 38, "totalCost": 12000000, "avgUtilization": 65.0, "overdueCount": 4 },
    "changePercent": { "bookings": 18.4, "cost": 25.0, "utilization": 11.5, "overdue": -50.0 }
  }
}
```

---

### `GET /api/v1/reports/bookings/trend`
Tren jumlah booking per periode waktu.

**Akses:** Admin

**Query Parameters:**
| Parameter | Type | Keterangan |
|-----------|------|-----------|
| `groupBy` | string | `daily` (default) \| `weekly` \| `monthly` |
| `periods` | integer | Jumlah periode ke belakang (default: 12) |

**Response `200`:**
```json
{
  "success": true,
  "message": "Booking trend",
  "data": [
    { "period": "2026-01", "totalBookings": 42, "approvedCount": 35, "rejectedCount": 4, "overdueCount": 3 }
  ]
}
```

---

### `GET /api/v1/reports/bookings/by-department`
Statistik booking dikelompokkan per departemen.

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
  "message": "Bookings by department",
  "data": [
    { "departmentId": 1, "departmentName": "Operations", "totalBookings": 28, "approvedCount": 24, "rejectedCount": 2, "overdueCount": 2, "totalCost": 8500000 }
  ]
}
```

---

### `GET /api/v1/reports/bookings/by-resource`
Statistik booking dikelompokkan per resource (kendaraan/ruangan).

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
  "message": "Bookings by resource",
  "data": [
    { "resourceId": 10, "resourceName": "Toyota Avanza", "resourceType": "VEHICLE", "totalBookings": 18, "approvedCount": 16, "rejectedCount": 1, "overdueCount": 1, "totalHours": 144 }
  ]
}
```

---

### `GET /api/v1/reports/bookings/approval-performance`
Performa proses approval (rata-rata waktu dari pengajuan ke keputusan).

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
  "message": "Approval performance",
  "data": [
    { "approverId": 1, "approverName": "Admin Utama", "totalDecisions": 30, "approvedCount": 26, "rejectedCount": 4, "avgHoursToDecision": 3.2 }
  ]
}
```

---

### `GET /api/v1/reports/cost-summary`
Ringkasan total biaya (BBM + maintenance) dengan perbandingan periode sebelumnya.

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
  "message": "Cost summary",
  "data": {
    "totalFuelCost": 8500000,
    "totalMaintenanceCost": 3200000,
    "totalCost": 11700000,
    "previousPeriod": { "totalFuelCost": 7200000, "totalMaintenanceCost": 2900000, "totalCost": 10100000 },
    "changePercent": { "fuel": 18.1, "maintenance": 10.3, "total": 15.8 }
  }
}
```

---

### `GET /api/v1/reports/cost/by-vehicle`
Biaya BBM dan maintenance per kendaraan.

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
  "message": "Cost by vehicle",
  "data": [
    { "vehicleId": 1, "vehicleName": "Toyota Avanza", "plateNumber": "B 1234 ABC", "totalFuelCost": 2100000, "totalMaintenanceCost": 850000, "totalCost": 2950000 }
  ]
}
```

---

### `GET /api/v1/reports/cost/by-department`
Biaya total per departemen.

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
  "message": "Cost by department",
  "data": [
    { "departmentId": 1, "departmentName": "Operations", "totalFuelCost": 4200000, "totalMaintenanceCost": 1500000, "totalCost": 5700000, "bookingCount": 22 }
  ]
}
```

---

### `GET /api/v1/reports/cost/trend`
Tren biaya per periode.

**Akses:** Admin

**Query Parameters:**
| Parameter | Type | Keterangan |
|-----------|------|-----------|
| `groupBy` | string | `daily` \| `weekly` \| `monthly` (default) |
| `periods` | integer | Jumlah periode ke belakang (default: 12) |

**Response `200`:**
```json
{
  "success": true,
  "message": "Cost trend",
  "data": [
    { "period": "2026-01", "totalFuelCost": 2100000, "totalMaintenanceCost": 750000, "totalCost": 2850000 }
  ]
}
```

---

### `GET /api/v1/reports/driver-performance`
Performa driver: jumlah trip, on-time rate, total BBM.

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
  "message": "Driver performance",
  "data": [
    { "driverId": 3, "driverName": "Joko Susilo", "totalTrips": 18, "onTimeRate": 94.4, "overdueCount": 1, "totalFuelCost": 2100000, "avgRating": 4.7 }
  ]
}
```

---

### `GET /api/v1/reports/department-summary`
Ringkasan booking dan biaya per departemen dalam satu tabel.

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
  "message": "Department summary",
  "data": [
    {
      "departmentId": 1,
      "departmentName": "Operations",
      "totalBookings": 28,
      "approvedCount": 24,
      "rejectedCount": 2,
      "overdueCount": 2,
      "totalCost": 5700000,
      "avgCostPerBooking": 203571
    }
  ]
}
```

---

## 📊 Modul Dashboard (`/api/v1/dashboard`)

### `GET /api/v1/dashboard/summary`
Mengambil data ringkasan dashboard (total booking, kendaraan aktif, dll).

**Akses:** User (Auth required)

**Response `200`:**
```json
{
  "success": true,
  "data": {
    "totalBookings": 150,
    "activeVehicles": 10,
    "pendingApprovals": 5
  }
}
```

---

## 🔔 Modul Notifications (`/api/v1/users/me/notifications`)

### `GET /api/v1/users/me/notifications`
Melihat history notifikasi pengguna saat ini (termasuk pagination).

**Akses:** User / Driver (Auth required)

**Query Parameters:** `?page=1&limit=20`

**Response `200`:**
```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "title": "Booking Update",
      "body": "Your booking has been approved",
      "type": "BOOKING_APPROVED",
      "is_read": false,
      "created_at": "2026-07-06T10:00:00Z"
    }
  ]
}
```

---

### `PATCH /api/v1/users/me/notifications/:id/read`
Menandai satu notifikasi sebagai sudah dibaca.

**Akses:** User (Auth required)

**Response `200`:**
```json
{
  "success": true,
  "message": "Notification marked as read"
}
```

---

### `PATCH /api/v1/users/me/notifications/read-all`
Menandai seluruh notifikasi milik pengguna sebagai sudah dibaca.

**Akses:** User (Auth required)

**Response `200`:**
```json
{
  "success": true,
  "message": "All notifications marked as read"
}
```

---

## 📥 Laporan & Ekspor File

### `GET /api/v1/reports/export/excel`
Mengunduh (*download*) seluruh laporan secara komprehensif ke dalam format `.xlsx`. File Excel ini memuat 18 *sheets* beserta *chart* (grafik) otomatis.

**Akses:** Admin / Manager

**Response:**
*File Download (`application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`)*

---

## 📁 Static Files

### `GET /uploads/*` *(public — tanpa autentikasi)*
Serve file yang diupload langsung di browser (untuk `<img src="">` di frontend).

**Akses:** Publik (tidak perlu Bearer Token)

**Catatan:** File gambar (`.jpg`, `.jpeg`, `.png`, `.gif`, `.webp`) dikirim dengan header `Content-Disposition: inline` sehingga browser langsung menampilkan gambar tanpa meminta download.

**Contoh penggunaan di frontend:**
```html
<img src="http://server:8080/uploads/profile/2026/06/abc123.jpg" />
<img src="http://server:8080/uploads/booking/2026/06/return_photo.jpg" />
```

---

### `GET /files/*` *(deprecated — perlu autentikasi)*
Serve file statis yang diupload (foto profil, foto kendaraan/ruangan, attachment).

**Akses:** Auth required (Bearer Token)

**Struktur Folder Upload:**

File disimpan berdasarkan kategori dan bulan upload:

| Kategori | Path | Digunakan untuk |
|----------|------|----------------|
| `profile` | `uploads/profile/YYYY/MM/` | Foto profil user |
| `vehicle` | `uploads/vehicle/YYYY/MM/` | Foto katalog kendaraan & attachment kendaraan |
| `room` | `uploads/room/YYYY/MM/` | Foto katalog ruangan & attachment ruangan |
| `booking` | `uploads/booking/YYYY/MM/` | Attachment booking & foto return report driver |

**Contoh URL akses file:**
- `/files/profile/2026/06/abc123.jpg` — Foto profil user
- `/files/vehicle/2026/06/def456.jpg` — Foto katalog kendaraan
- `/files/room/2026/06/ghi789.jpg` — Foto katalog ruangan
- `/files/booking/2026/06/jkl012.jpg` — Attachment/foto return report booking

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
---

## 💾 Database Migration Guide (Update Terbaru)

Sehubungan dengan pembaruan fitur, berikut perintah SQL yang perlu dijalankan untuk memperbarui schema lama:

```sql
-- 1. Tambahkan passengerCount ke bookings
ALTER TABLE bookings ADD COLUMN passenger_count INT NOT NULL DEFAULT 1;

-- 2. Buat tabel Fuel Types sebagai master data bahan bakar
CREATE TYPE fuel_category AS ENUM ('BBM', 'LISTRIK');
CREATE TYPE fuel_unit AS ENUM ('LITER', 'KWH');
CREATE TABLE fuel_types (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type fuel_category NOT NULL,
    unit fuel_unit NOT NULL,
    default_price NUMERIC(12,2) NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Modifikasi Fuel Expenses
ALTER TABLE fuel_expenses RENAME COLUMN resource_id TO vehicle_id;
ALTER TABLE fuel_expenses DROP COLUMN resource_type;
ALTER TABLE fuel_expenses ADD COLUMN fuel_type_id INT NOT NULL REFERENCES fuel_types(id);
ALTER TABLE fuel_expenses ALTER COLUMN price_per_unit DROP NOT NULL;
ALTER TABLE fuel_expenses ALTER COLUMN total_cost DROP NOT NULL;

-- 4. Modifikasi Maintenance Records
ALTER TABLE maintenance_records RENAME COLUMN resource_id TO vehicle_id;
ALTER TABLE maintenance_records DROP COLUMN resource_type;

-- 5. Buat Tabel Notifikasi
CREATE TABLE notifications (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    body TEXT NOT NULL,
    type VARCHAR(50) NOT NULL,
    related_entity_id INT,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```
*(Catatan: Schema utama baru sudah di-bundle pada `000001_init.up.sql` secara utuh jika melakukan inisialisasi awal database yang fresh).*
