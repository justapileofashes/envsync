package crypto

import (
	"bytes"
	"testing"
)

func TestDeriveKeyDeterministic(t *testing.T) {
	salt := []byte("a-shared-salt-16")
	k1, err := DeriveKey("correct horse battery staple", salt)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}
	k2, _ := DeriveKey("correct horse battery staple", salt)
	if !bytes.Equal(k1, k2) {
		t.Fatal("same passphrase+salt must derive identical keys")
	}
	if len(k1) != KeySize {
		t.Fatalf("key length = %d, want %d", len(k1), KeySize)
	}
	k3, _ := DeriveKey("different passphrase", salt)
	if bytes.Equal(k1, k3) {
		t.Fatal("different passphrase must derive a different key")
	}
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	key, _ := DeriveKey("team-passphrase", []byte("salty-salty-salt"))
	plaintext := []byte(`{"v":1,"values":{"API_KEY":"s3cr3t","DB_URL":"postgres://x"}}`)

	blob, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if bytes.Contains(blob, []byte("s3cr3t")) {
		t.Fatal("ciphertext leaks plaintext")
	}

	got, err := Decrypt(key, blob)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatal("round-trip mismatch")
	}
}

func TestDecryptWrongKeyFails(t *testing.T) {
	good, _ := DeriveKey("right", []byte("salty-salty-salt"))
	bad, _ := DeriveKey("wrong", []byte("salty-salty-salt"))
	blob, _ := Encrypt(good, []byte("hello"))
	if _, err := Decrypt(bad, blob); err == nil {
		t.Fatal("expected decryption with wrong key to fail")
	}
}

func TestDecryptTamperedFails(t *testing.T) {
	key, _ := DeriveKey("k", []byte("salty-salty-salt"))
	blob, _ := Encrypt(key, []byte("hello world"))
	blob[len(blob)-1] ^= 0xFF // flip a tag bit
	if _, err := Decrypt(key, blob); err == nil {
		t.Fatal("expected tampered ciphertext to fail authentication")
	}
}

func TestInvalidKeySize(t *testing.T) {
	if _, err := Encrypt([]byte("short"), []byte("x")); err != ErrInvalidKey {
		t.Fatalf("want ErrInvalidKey, got %v", err)
	}
}
