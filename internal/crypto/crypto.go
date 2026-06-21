// Package crypto implements EnvSync's client-side cryptographic engine.
//
// EnvSync is zero-knowledge: the server only ever stores opaque AES-256-GCM
// ciphertext. The symmetric key is derived locally from the team passphrase and
// a per-organization salt via PBKDF2-HMAC-SHA256 and never leaves the machine.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// KeySize is the AES-256 key length in bytes.
	KeySize = 32
	// SaltSize is the recommended salt length in bytes.
	SaltSize = 16
	// pbkdf2Iterations follows the OWASP recommendation for PBKDF2-HMAC-SHA256.
	pbkdf2Iterations = 600_000
)

// ErrInvalidKey is returned when a key of the wrong length is supplied.
var ErrInvalidKey = errors.New("crypto: key must be exactly 32 bytes (AES-256)")

// ErrMalformedCiphertext is returned when a ciphertext blob is too short or
// otherwise structurally invalid before authentication is even attempted.
var ErrMalformedCiphertext = errors.New("crypto: malformed ciphertext")

// ErrDecryptionFailed is returned when GCM authentication fails. This almost
// always means a wrong passphrase/salt or a tampered/corrupted payload.
var ErrDecryptionFailed = errors.New("crypto: decryption failed (wrong passphrase or corrupted data)")

// DeriveKey derives a deterministic 32-byte AES-256 key from a passphrase and
// salt using PBKDF2-HMAC-SHA256. Identical (passphrase, salt) pairs always
// produce the same key, which is what lets independent teammates decrypt the
// same blob without ever exchanging the key itself.
func DeriveKey(passphrase string, salt []byte) ([]byte, error) {
	if len(passphrase) == 0 {
		return nil, errors.New("crypto: passphrase must not be empty")
	}
	if len(salt) == 0 {
		return nil, errors.New("crypto: salt must not be empty")
	}
	return pbkdf2.Key([]byte(passphrase), salt, pbkdf2Iterations, KeySize, sha256.New), nil
}

// Encrypt seals plaintext with AES-256-GCM using the supplied 32-byte key. A
// fresh random nonce is generated per call and prepended to the returned blob:
//
//	[ nonce (12 bytes) | ciphertext+tag ]
//
// The caller is responsible for transport encoding (EnvSync base64-encodes the
// blob for JSON transport).
func Encrypt(key, plaintext []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("crypto: failed to generate nonce: %w", err)
	}

	// Seal appends the ciphertext to nonce, giving us nonce||ciphertext in one
	// allocation.
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt reverses Encrypt. It expects a blob of the form nonce||ciphertext+tag
// produced by Encrypt and verifies the GCM authentication tag.
func Decrypt(key, blob []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(blob) < nonceSize {
		return nil, ErrMalformedCiphertext
	}

	nonce, ciphertext := blob[:nonceSize], blob[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}
	return plaintext, nil
}

// newGCM constructs an AES-256-GCM AEAD from a validated 32-byte key.
func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != KeySize {
		return nil, ErrInvalidKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: failed to initialize AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("crypto: failed to initialize GCM: %w", err)
	}
	return gcm, nil
}

// NewSalt generates a cryptographically random salt of SaltSize bytes, suitable
// for seeding a new organization's key derivation.
func NewSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("crypto: failed to generate salt: %w", err)
	}
	return salt, nil
}
