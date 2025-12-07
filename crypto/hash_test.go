package crypto

import (
	"strings"
	"testing"
)

func TestArgon2Hasher_Hash(t *testing.T) {
	hasher := NewArgon2Hasher()

	password := "my-secure-password-123"
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}

	if hash == "" {
		t.Fatal("Expected hash, got empty string")
	}

	// Check hash format
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("Expected hash to start with $argon2id$, got %s", hash)
	}

	// Hash should be different each time (due to random salt)
	hash2, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Second hash failed: %v", err)
	}

	if hash == hash2 {
		t.Error("Expected different hashes for same password (different salts)")
	}
}

func TestArgon2Hasher_Verify(t *testing.T) {
	hasher := NewArgon2Hasher()

	password := "my-secure-password-123"
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Hash failed: %v", err)
	}

	// Verify correct password
	valid, err := hasher.Verify(password, hash)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if !valid {
		t.Error("Expected password to be valid")
	}

	// Verify incorrect password
	valid, err = hasher.Verify("wrong-password", hash)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if valid {
		t.Error("Expected password to be invalid")
	}
}

func TestArgon2Hasher_VerifyInvalidFormat(t *testing.T) {
	hasher := NewArgon2Hasher()

	tests := []struct {
		name string
		hash string
	}{
		{
			name: "too few parts",
			hash: "$argon2id$v=19$m=65536",
		},
		{
			name: "invalid algorithm",
			hash: "$bcrypt$v=19$m=65536,t=3,p=2$c2FsdA$aGFzaA",
		},
		{
			name: "empty hash",
			hash: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := hasher.Verify("password", tt.hash)
			if err == nil {
				t.Error("Expected error for invalid hash format")
			}
		})
	}
}

func TestArgon2Hasher_ConstantTime(t *testing.T) {
	hasher := NewArgon2Hasher()

	password := "my-secure-password-123"
	hash, _ := hasher.Hash(password)

	// These should take roughly the same time (constant-time comparison)
	// This is more of a smoke test than a rigorous timing test
	hasher.Verify("wrong-password-1", hash)
	hasher.Verify("wrong-password-2", hash)
	hasher.Verify(password, hash)
}

func BenchmarkArgon2Hasher_Hash(b *testing.B) {
	hasher := NewArgon2Hasher()
	password := "benchmark-password"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hasher.Hash(password)
	}
}

func BenchmarkArgon2Hasher_Verify(b *testing.B) {
	hasher := NewArgon2Hasher()
	password := "benchmark-password"
	hash, _ := hasher.Hash(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hasher.Verify(password, hash)
	}
}
