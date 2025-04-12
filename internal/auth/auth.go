package auth

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
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

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	regClaim := jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		Subject:   userID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, regClaim)
	tString, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		fmt.Printf("Sighned error: %v\n", err)
		return "", err
	}
	return tString, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
		return uuid.Nil, err
	}
	if !token.Valid {
		return uuid.Nil, errors.New("invalid token")
	}
	id := claims.Subject
	UsrId, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, err
	}
	return UsrId, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	BearerToken := headers.Get("Authorization")
	splitToken := strings.Split(BearerToken, " ")
	if len(splitToken) == 1 {
		return "", errors.New("missing token")
	}
	return splitToken[1], nil
}
