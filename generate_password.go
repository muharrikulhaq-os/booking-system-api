package main

import (
	"os"

	"golang.org/x/crypto/bcrypt"
)

// Membuat hash bcrypt dari password: `go run generate_password.go <password>`.
// Tanpa argumen memakai "admin" (perilaku lama). Dipakai juga oleh runner
// tests/skenario untuk membuat admin uji langsung di database test.

func GeneratePassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func main() {
	password := "admin"
	if len(os.Args) > 1 {
		password = os.Args[1]
	}
	hashedPassword, err := GeneratePassword(password)
	if err != nil {
		panic(err)
	}
	println("Hashed password:", hashedPassword)
}
