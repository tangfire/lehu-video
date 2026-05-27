package auth

import (
	"errors"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"
)

const DefaultTokenTTL = 7 * 24 * time.Hour

type Claims struct {
	jwtv5.RegisteredClaims
	UserId string `json:"user_id"`
}

func NewClaims(userID string, ttl time.Duration) *Claims {
	now := time.Now()
	if ttl <= 0 {
		ttl = DefaultTokenTTL
	}
	return &Claims{
		RegisteredClaims: jwtv5.RegisteredClaims{
			IssuedAt:  jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(now.Add(ttl)),
		},
		UserId: userID,
	}
}

func GenerateToken(secret string, claims *Claims) (string, error) {
	if secret == "" {
		return "", errors.New("jwt secret is required")
	}
	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseToken(tokenStr, secret string) (*Claims, error) {
	if secret == "" {
		return nil, errors.New("jwt secret is required")
	}
	token, err := jwtv5.ParseWithClaims(tokenStr, &Claims{}, func(token *jwtv5.Token) (interface{}, error) {
		if token.Method != jwtv5.SigningMethodHS256 {
			return nil, errors.New("unexpected jwt signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid jwt claims")
}
