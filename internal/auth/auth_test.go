package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPwdHash(t *testing.T) {
	passHash, _ := HashPassword("test")
	err := bcrypt.CompareHashAndPassword([]byte(passHash), []byte("test"))
	if err != nil {
		t.Errorf("Does not match: %v\n", err)
	}
}

func TestCheckPwdHash(t *testing.T) {
	passHash, _ := HashPassword("test")
	err := CheckPasswordHash(passHash, "test")
	if err != nil {
		t.Errorf("Does not match: %v\n", err)
	}
}
