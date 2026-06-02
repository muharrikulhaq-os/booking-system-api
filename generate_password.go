package main

import "golang.org/x/crypto/bcrypt"

// make generate password from text to hash

func GeneratePassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func main() {
	password := "admin"
	hashedPassword, err := GeneratePassword(password)
	if err != nil {
		panic(err)
	}
	println("Hashed password:", hashedPassword)
}
