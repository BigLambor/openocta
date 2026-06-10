package rbac

import "testing"

func TestHashPasswordArgon2idAndVerify(t *testing.T) {
	hash, salt := HashPasswordArgon2id("secure-pass-123")
	if salt != "" {
		t.Fatalf("expected empty salt column for argon2, got %q", salt)
	}
	if !IsArgon2Hash(hash) {
		t.Fatalf("expected argon2 hash prefix, got %q", hash)
	}
	if !VerifyPassword("secure-pass-123", hash, salt) {
		t.Fatal("expected password to verify")
	}
	if VerifyPassword("wrong-pass", hash, salt) {
		t.Fatal("expected wrong password to fail")
	}
}

func TestVerifyPasswordLegacySHA256(t *testing.T) {
	salt := "test-salt"
	hash := HashPasswordLegacy("legacy-pass", salt)
	if !VerifyPassword("legacy-pass", hash, salt) {
		t.Fatal("expected legacy password to verify")
	}
	if VerifyPassword("wrong", hash, salt) {
		t.Fatal("expected wrong legacy password to fail")
	}
}
