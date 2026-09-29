package repository

import (
	"database/sql"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// dbSessionTimeZone dipasang sebagai zona sesi di SETIAP koneksi, supaya
// date_trunc/TO_CHAR/generate_series di SQL (cek SPD per hari, tren harian/
// mingguan/bulanan, statistik bulan berjalan) memotong hari & bulan menurut
// WIB — bukan menurut setelan server PostgreSQL (lokal terpasang
// Asia/Bangkok; offset-nya sama dengan WIB hanya kebetulan konfigurasi).
// Nilai yang disimpan tidak terpengaruh: timestamptz tetap titik waktu
// absolut, dan driver membacanya dalam biner UTC. Selaras util.WIB.
const dbSessionTimeZone = "Asia/Jakarta"

var DB *sql.DB

func Connect(dsn string) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		log.Fatalf("invalid database url: %v", err)
	}
	cfg.RuntimeParams["timezone"] = dbSessionTimeZone
	DB = stdlib.OpenDB(*cfg)
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(10)
	if err = DB.Ping(); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("database connected")
}
