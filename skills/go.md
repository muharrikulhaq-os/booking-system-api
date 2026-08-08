### 🚀 Technical Skills & Concepts: Resource Booking System API

Dokumen ini menjabarkan teknologi, alat, arsitektur, dan pola desain perangkat lunak yang diterapkan dalam pengembangan backend Resource Booking System.

### 1. Bahasa Pemrograman & Framework

* Golang (Go) 1.26: Digunakan untuk membangun backend yang highly performant, strongly typed, dan mendukung concurrency secara efisien.

* Go Fiber (`gofiber/fiber/v2`): Framework web Golang berperforma tinggi yang terinspirasi dari Express.js untuk menangani HTTP routing, request/response cycle, dan middleware.

### 2. Arsitektur & Pola Desain (Design Patterns)

* Clean Architecture / Layered Pattern: Struktur kode dipisahkan secara modular untuk menjaga separation of concerns:

  * Delivery (HTTP Handlers): Menangani request HTTP dan mengembalikan response JSON.

  * Service (Use Cases): Berisi logika bisnis dan aturan aplikasi.

  * Repository: Lapisan yang berinteraksi langsung dengan database.

* Dependency Injection: Dependency seperti Repository diinjeksi ke Service, lalu Service ke Handler melalui `cmd/api/main.go`, sehingga memudahkan mocking dan unit testing.

* RESTful API Design: API dirancang mengikuti prinsip REST untuk operasi CRUD pada entitas seperti Users, Vehicles, Rooms, dan Bookings.

### 3. Database & Pengelolaan Data (Database Engineering)

* PostgreSQL: Relational Database Management System (RDBMS) yang andal dan kaya fitur.

* SQLC (`sqlc.dev`): Menggunakan pendekatan database-first untuk menghasilkan kode Go yang type-safe langsung dari raw SQL query, tanpa ORM konvensional demi performa dan kontrol query yang optimal.

* Advanced SQL Schema Design:

  * Penggunaan ENUM types seperti `booking_status`, `role_name`, dan `fuel_type`.

  * Penerapan constraints seperti `CHECK` untuk validasi tahun kendaraan, rentang tanggal booking, serta validasi kapasitas baterai/BBM.

  * Pembuatan indexes pada kolom yang sering digunakan dalam pencarian untuk optimasi performa query.

  * Penggunaan database triggers melalui fungsi `trigger_set_updated_at()` untuk memperbarui kolom `updatedAt` secara otomatis.

  * Penerapan foreign keys dengan aksi `ON DELETE CASCADE` guna menjaga integritas data referensial.

### 4. Keamanan (Security) & Autentikasi

* JWT (JSON Web Tokens) Authentication: Implementasi autentikasi stateless menggunakan `golang-jwt/jwt/v5` untuk melindungi endpoint API.

* Role-Based Access Control (RBAC): Pembatasan hak akses berdasarkan peran pengguna seperti Admin, Employee, Driver, dan Room Keeper.

* Password Cryptography (`golang.org/x/crypto/bcrypt`): Pengamanan kata sandi menggunakan algoritma bcrypt untuk proses hashing dan verifikasi password.

* Refresh Token & OTP Mechanism: Sistem mendukung manajemen sesi lanjutan menggunakan refresh token dan OTP untuk proses reset password.

### 5. Validasi & Middleware

* Struct Data Validation (`go-playground/validator/v10`): Validasi otomatis terhadap payload JSON menggunakan struct tags di Golang, seperti validasi required, email, dan minimum length.

* Custom Middleware:

  * Middleware autentikasi dan otorisasi berbasis JWT.

  * Global Error Handler untuk menghasilkan format response error JSON yang konsisten.

### 6. Fitur Domain Spesifik (Business Logic)

* Resource Scheduling & Conflict Resolution: Validasi konflik jadwal (overlapping schedule) pada `startDate` dan `endDate` ketika melakukan booking kendaraan atau ruangan.

* State Machine Lifecycle: Pengelolaan transisi status booking secara terstruktur, misalnya `PENDING → APPROVED → ONGOING → COMPLETED / REJECTED`.

* Audit Logging: Pencatatan riwayat setiap aksi penting dalam sistem secara immutable untuk kebutuhan audit dan pelacakan aktivitas.

* Multipart/Form-Data (File Uploads): Mendukung unggahan file seperti foto profil, lampiran laporan pengembalian kendaraan, dan dokumen pendukung lainnya.

### 7. Tooling & Environment

* Environment Variables (`godotenv`): Pengelolaan konfigurasi aplikasi secara aman melalui file `.env`, termasuk kredensial database, port aplikasi, dan JWT secret.

* Postman Collection: Dokumentasi dan koleksi pengujian API tersedia dalam file `booking-system-collection.postman.json`, mencakup berbagai request dan test script untuk kebutuhan development dan QA.

### ✨ Ringkasan

Backend Resource Booking System dibangun menggunakan Golang + Fiber + PostgreSQL + SQLC dengan pendekatan Clean Architecture. Sistem ini menekankan performa tinggi, keamanan berbasis JWT & RBAC, integritas data database, serta logika bisnis penjadwalan dan approval workflow yang kompleks namun terstruktur.
