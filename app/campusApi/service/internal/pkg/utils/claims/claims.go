package claims

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-kratos/kratos/v2/middleware/auth/jwt"
	sharedauth "lehu-video/pkg/auth"
)

type Claims = sharedauth.Claims

func New(userId string) *Claims {
	return sharedauth.NewClaims(userId, sharedauth.DefaultTokenTTL)
}

func GetUserId(ctx context.Context) (string, error) {
	anyClaims, ok := jwt.FromContext(ctx)
	if !ok {
		return "0", errors.New("no claims in context")
	}

	claims, ok := anyClaims.(*Claims)
	if !ok {
		return "0", errors.New("claims type error")
	}

	return claims.UserId, nil
}

func GenerateToken(claim *Claims) (string, error) {
	tokenString, err := sharedauth.GenerateToken("token", claim)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return tokenString, nil
}
