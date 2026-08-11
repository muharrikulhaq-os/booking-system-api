package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv  string
	AppPort string

	DatabaseURL string

	JWTSecret            string
	JWTAccessExpireMin   int
	JWTRefreshExpireDays int

	OTPLength        int
	OTPExpireMinutes int

	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPFrom string

	UploadDir     string
	MaxFileSizeMB int64

	FrontendOrigin string

	// Path file service-account Firebase (untuk push FCM). Bila file tidak
	// ada, push FCM dinonaktifkan otomatis (WebSocket tetap jalan).
	FirebaseCredFile string
}

var C Config

func Load() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file, reading from environment")
	}

	C = Config{
		AppEnv:               getEnv("APP_ENV", "development"),
		AppPort:              getEnv("APP_PORT", "8080"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/reservation"),
		JWTSecret:            getEnv("JWT_SECRET", "changeme"),
		JWTAccessExpireMin:   getEnvInt("JWT_ACCESS_EXPIRE_MINUTES", 15),
		JWTRefreshExpireDays: getEnvInt("JWT_REFRESH_EXPIRE_DAYS", 7),
		OTPLength:            getEnvInt("OTP_LENGTH", 6),
		OTPExpireMinutes:     getEnvInt("OTP_EXPIRE_MINUTES", 15),
		SMTPHost:             getEnv("SMTP_HOST", ""),
		SMTPPort:             getEnvInt("SMTP_PORT", 587),
		SMTPUser:             getEnv("SMTP_USER", ""),
		SMTPPass:             getEnv("SMTP_PASS", ""),
		SMTPFrom:             getEnv("SMTP_FROM", "noreply@company.com"),
		UploadDir:            getEnv("UPLOAD_DIR", "./uploads"),
		MaxFileSizeMB:        int64(getEnvInt("MAX_FILE_SIZE_MB", 10)),
		FirebaseCredFile:     getEnv("FIREBASE_CREDENTIALS_FILE", "./firebase-service-account.json"),
		FrontendOrigin:       getEnv("FRONTEND_ORIGIN", "http://localhost:3000"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
