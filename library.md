# Golang Project Requirements (Fiber + sqlc)

Dokumen ini berisi daftar pustaka (libraries) dan alat (tools) yang digunakan untuk migrasi *Resource Booking System* dari Python (FastAPI + SQLAlchemy) ke Golang (Fiber + sqlc).

## 1. Alat Baris Perintah (CLI Tools)
Tools ini tidak diinstal di dalam `go.mod`, melainkan diinstal di sistem lokal untuk membantu proses *development*.

* **sqlc** (`github.com/sqlc-dev/sqlc`)
    * **Fungsi:** Men-generate kode Go (*type-safe*) dari file SQL mentah (`schema.sql` & `query.sql`).
    * **Pengganti:** SQLAlchemy (ORM).
    * **Instalasi:** `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
* **golang-migrate** (`github.com/golang-migrate/migrate`)
    * **Fungsi:** Menjalankan migrasi skema database.
    * **Pengganti:** Alembic.
    * **Instalasi:** `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`

## 2. Pustaka Utama (go.mod Dependencies)
Jalankan perintah `go get` berikut untuk menginstal dependensi ke dalam proyek.

### Web Framework & HTTP
* **Fiber:** `go get github.com/gofiber/fiber/v2`
    * **Fungsi:** Web framework, routing, HTTP parsing.
    * **Pengganti:** FastAPI + Uvicorn.

### Database Driver
* **pgx:** `go get github.com/jackc/pgx/v5`
    * **Fungsi:** Driver PostgreSQL berkinerja tinggi.
    * **Pengganti:** `psycopg2`.

### Keamanan & Autentikasi
* **Bcrypt:** `go get golang.org/x/crypto/bcrypt`
    * **Fungsi:** Hashing dan verifikasi password.
    * **Pengganti:** `passlib`.
* **JWT:** `go get github.com/golang-jwt/jwt/v5`
    * **Fungsi:** Pembuatan dan validasi token sesi (Access & Refresh).
    * **Pengganti:** `python-jose`.

### Validasi Data
* **Validator:** `go get github.com/go-playground/validator/v10`
    * **Fungsi:** Validasi *request body* menggunakan *struct tags* (misal: `validate:"required,email"`).
    * **Pengganti:** Pydantic.

### Manajemen Konfigurasi
* **Godotenv:** `go get github.com/joho/godotenv`
    * **Fungsi:** Memuat variabel lingkungan dari file `.env`.
    * **Pengganti:** `pydantic-settings` / `python-dotenv`.

## 3. Struktur Direktori yang Disarankan
```text
.
├── cmd/
│   └── api/
│       └── main.go           # Entry point aplikasi
├── internal/
│   ├── config/               # Load .env dan setup konfigurasi
│   ├── delivery/
│   │   └── http/             # Fiber Handlers / Controllers
│   ├── middleware/           # Auth, Error Handler, Logger
│   ├── repository/           # Kode hasil generate sqlc
│   ├── service/              # Business logic (translasi dari Python services)
│   └── util/                 # Helper (hash, jwt, response formatter)
├── sql/
│   ├── schema/               # File migrasi skema database
│   └── query/                # File instruksi SQL untuk sqlc
├── .env
├── sqlc.yaml                 # Konfigurasi sqlc
└── go.mod