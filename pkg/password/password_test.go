package password

import "testing"

func TestHashVerifyAndModern(t *testing.T) {
	hash, salt, err := Hash("12345678")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if salt != "" {
		t.Fatalf("modern salt should be empty, got %q", salt)
	}
	if !IsModern(hash) {
		t.Fatalf("hash should be modern: %s", hash)
	}
	ok, needsUpgrade := Verify("12345678", hash, salt)
	if !ok || needsUpgrade {
		t.Fatalf("Verify modern = (%v, %v), want (true, false)", ok, needsUpgrade)
	}
}

func TestVerifyLegacyMD5NeedsUpgrade(t *testing.T) {
	legacy := MD5WithSalt("12345678", "salt")
	ok, needsUpgrade := Verify("12345678", legacy, "salt")
	if !ok || !needsUpgrade {
		t.Fatalf("Verify legacy = (%v, %v), want (true, true)", ok, needsUpgrade)
	}
}
