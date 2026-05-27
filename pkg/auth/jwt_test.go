package auth

import "testing"

func TestGenerateAndParseToken(t *testing.T) {
	token, err := GenerateToken("secret", NewClaims("42", DefaultTokenTTL))
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	claims, err := ParseToken(token, "secret")
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.UserId != "42" {
		t.Fatalf("claims.UserId = %q, want 42", claims.UserId)
	}
}

func TestParseTokenRejectsWrongSecret(t *testing.T) {
	token, err := GenerateToken("secret", NewClaims("42", DefaultTokenTTL))
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if _, err := ParseToken(token, "other"); err == nil {
		t.Fatal("ParseToken() error = nil, want error")
	}
}
