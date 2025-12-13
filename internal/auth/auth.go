package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v4"
)

const TOKEN_EXP = time.Minute * 30

type JWTService struct {
	SecretKey []byte
}

func NewJWTService(key []byte) *JWTService {
	return &JWTService{SecretKey: key}
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func (j *JWTService) GenerateToken(userID string) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TOKEN_EXP)),
		},
		UserID: userID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.SecretKey)
}

func (j *JWTService) GetUserID(tokenString string) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(j.SecretKey), nil
	})

	if err != nil || !token.Valid {
		return "", err
	}
	return claims.UserID, nil
}
