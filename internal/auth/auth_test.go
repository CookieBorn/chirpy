package auth

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
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

func TestJWTCreationAndValidation(t *testing.T) {
	// Create a test user ID
	userID := uuid.New()

	// Define a secret key
	secret := "test_secret_key"

	// Create a token that expires in 1 hour
	token, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create JWT: %v", err)
	}

	// Validate the token
	returnedID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("Failed to validate JWT: %v", err)
	}

	// Check if the returned ID matches the original
	if returnedID != userID {
		t.Errorf("User ID mismatch: got %v, want %v", returnedID, userID)
	}
}

func TestJWTExpiration(t *testing.T) {
	// Create a test user ID
	userID := uuid.New()

	// Define a secret key
	secret := "test_secret_key"

	// Create a token that expires immediately
	token, err := MakeJWT(userID, secret, -time.Hour)
	if err != nil {
		t.Fatalf("Failed to create JWT: %v", err)
	}

	// Validate the token, should fail because it's expired
	_, err = ValidateJWT(token, secret)
	if err == nil {
		t.Error("Expected validation to fail for expired token, but it succeeded")
	}
}

func TestJWTInvalidSecret(t *testing.T) {
	// Create a test user ID
	userID := uuid.New()

	// Define a secret key
	secret := "test_secret_key"
	wrongSecret := "wrong_secret_key"

	// Create a token with the correct secret
	token, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("Failed to create JWT: %v", err)
	}

	// Validate with the wrong secret, should fail
	_, err = ValidateJWT(token, wrongSecret)
	if err == nil {
		t.Error("Expected validation to fail with wrong secret, but it succeeded")
	}
}

func TestJWTMalformedToken(t *testing.T) {
	// Test with a completely invalid token
	_, err := ValidateJWT("not.a.valid.token", "any_secret")
	if err == nil {
		t.Error("Expected validation to fail for malformed token, but it succeeded")
	}
}

func GetBearerTokenTest(t *testing.T) {
	header := http.Header{}
	header.Add("Authorization", "Bearer TOKEN_STRING")
	_, err := GetBearerToken(header)
	if err != nil {
		fmt.Printf("%v\n", err)
		return
	}
}
