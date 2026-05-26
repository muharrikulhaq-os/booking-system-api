package util

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"booking-system-api/internal/config"
)

func GenerateOTP(length int) string {
	const digits = "0123456789"
	otp := make([]byte, length)
	for i := range otp {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		otp[i] = digits[n.Int64()]
	}
	return string(otp)
}

func OTPExpiry() time.Time {
	return time.Now().Add(time.Duration(config.C.OTPExpireMinutes) * time.Minute)
}

func GenerateToken(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b[i] = chars[idx.Int64()]
	}
	return string(b)
}

func GenerateUniqueFilename(original string) string {
	token := GenerateToken(16)
	return fmt.Sprintf("%s_%s", token, sanitizeFilename(original))
}

func sanitizeFilename(name string) string {
	safe := make([]byte, 0, len(name))
	for _, c := range []byte(name) {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '.' || c == '-' || c == '_' {
			safe = append(safe, c)
		} else {
			safe = append(safe, '_')
		}
	}
	return string(safe)
}
