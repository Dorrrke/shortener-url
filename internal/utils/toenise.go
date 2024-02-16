package utils

import (
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// SecretKey - Секретный ключ для создания JWT токена.
const SecretKey = "Secret123Key345Super"

type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

func CreateJWTToken(uuid string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 3)),
		},
		UserID: uuid,
	})

	tokenString, err := token.SignedString([]byte(SecretKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func GetUID(tokenString string) string {
	claim := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claim, func(t *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})
	if err != nil {
		return ""
	}

	if !token.Valid {
		return ""
	}

	return claim.UserID
}

func ValidationURL(URL string) bool {
	if strings.HasPrefix(URL, "http://") || strings.HasPrefix(URL, "https://") {
		return true
	}
	return false
}
