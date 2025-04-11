package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	bytes := []byte(password)
	pwd, err := bcrypt.GenerateFromPassword(bytes, 10)
	if err != nil {
		fmt.Printf("Generate Password error: %v\n", err)
		return "", err
	}
	pwdString := string(pwd)
	return pwdString, nil
}

func CheckPasswordHash(hash, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		fmt.Printf("Check Password Error: %v\n", err)
		return err
	}
	return nil
}
