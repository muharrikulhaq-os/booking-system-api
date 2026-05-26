package repository

import (
	"database/sql"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var DB *sql.DB

func Connect(dsn string) {
	var err error
	DB, err = sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("unable to open database: %v", err)
	}
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(10)
	if err = DB.Ping(); err != nil {
		log.Fatalf("database ping failed: %v", err)
	}
	log.Println("database connected")
}
